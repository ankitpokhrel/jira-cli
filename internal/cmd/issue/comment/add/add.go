package add

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdcommon"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const (
	helpText = `Add adds comment to an issue.`
	examples = `$ jira issue comment add

# Pass required parameters to skip prompt 
$ jira issue comment add ISSUE-1 "My comment"

# Multi-line comment
$ jira issue comment add ISSUE-1 $'Supports\n\nNew line'

# Load comment body from a template file
$ jira issue comment add ISSUE-1 --template /path/to/template.tmpl

# Get comment body from standard input
$ jira issue comment add ISSUE-1 --template -

# Or, use pipe to read input directly from standard input
$ echo "Comment from stdin" | jira issue comment add ISSUE-1

# Mention a user by display name. The body must contain the matching @Name text.
$ jira issue comment add ISSUE-1 "Hi @Person A, please review." --mention "Person A"

# Positional argument takes precedence over the template flag
# The example below will add "comment from arg" as a comment
$ jira issue comment add ISSUE-1 "comment from arg" --template /path/to/template.tmpl`
)

// NewCmdCommentAdd is a comment add command.
func NewCmdCommentAdd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "add ISSUE-KEY [COMMENT_BODY]",
		Short:   "Add a comment to an issue",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key of the source issue, eg: ISSUE-1\n" +
				"COMMENT_BODY\tBody of the comment you want to add",
		},
		Run: add,
	}

	cmd.Flags().Bool("web", false, "Open issue in web browser after adding comment")
	cmd.Flags().StringP("template", "T", "", "Path to a file to read comment body from")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")
	cmd.Flags().Bool("internal", false, "Make comment internal")
	cmd.Flags().StringArray("mention", []string{}, "Mention a user by display name; repeatable")

	return &cmd
}

func add(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)
	ac := addCmd{
		client:    client,
		linkTypes: nil,
		params:    params,
	}

	if ac.isNonInteractive() {
		ac.params.noInput = true

		if ac.isMandatoryParamsMissing() {
			cmdutil.Failed("`ISSUE-KEY` is mandatory when using a non-interactive mode")
		}
	}

	cmdutil.ExitIfError(ac.setIssueKey())

	qs := ac.getQuestions()
	if len(qs) > 0 {
		ans := struct{ Body string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		params.body = ans.Body
	}

	mentions, err := ac.resolveMentions()
	cmdutil.ExitIfError(err)

	if !params.noInput {
		answer := struct{ Action string }{}
		err := survey.Ask([]*survey.Question{getNextAction()}, &answer)
		cmdutil.ExitIfError(err)

		if answer.Action == cmdcommon.ActionCancel {
			cmdutil.Failed("Action aborted")
		}
	}

	err = func() error {
		s := cmdutil.Info("Adding comment")
		defer s.Stop()

		return client.AddIssueCommentWithMentions(ac.params.issueKey, ac.params.body, mentions, ac.params.internal)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Comment added to issue %q", ac.params.issueKey)
	fmt.Printf("%s\n", cmdutil.GenerateServerBrowseURL(server, ac.params.issueKey))

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, ac.params.issueKey)
		cmdutil.ExitIfError(err)
	}
}

type addParams struct {
	issueKey string
	body     string
	template string
	noInput  bool
	internal bool
	mentions []string
	debug    bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *addParams {
	var issueKey, body string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if nargs >= 2 {
		body = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	template, err := flags.GetString("template")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	internal, err := flags.GetBool("internal")
	cmdutil.ExitIfError(err)

	mentions, err := flags.GetStringArray("mention")
	cmdutil.ExitIfError(err)

	return &addParams{
		issueKey: issueKey,
		body:     body,
		template: template,
		noInput:  noInput,
		internal: internal,
		mentions: mentions,
		debug:    debug,
	}
}

type addCmd struct {
	client    *jira.Client
	linkTypes []*jira.IssueLinkType
	params    *addParams
}

func (ac *addCmd) setIssueKey() error {
	if ac.params.issueKey != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "issueKey",
		Prompt:   &survey.Input{Message: "Issue key"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	ac.params.issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), ans)

	return nil
}

func (ac *addCmd) getQuestions() []*survey.Question {
	var (
		qs          []*survey.Question
		defaultBody string
	)

	if ac.params.template != "" || cmdutil.StdinHasData() {
		b, err := cmdutil.ReadFile(ac.params.template)
		if err != nil {
			cmdutil.Failed("Error: %s", err)
		}
		defaultBody = string(b)
	}

	if ac.params.noInput && ac.params.body == "" {
		ac.params.body = defaultBody
		return qs
	}

	if ac.params.body == "" {
		qs = append(qs, &survey.Question{
			Name: "body",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Comment body",
					Default:       defaultBody,
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: false,
			},
		})
	}

	return qs
}

func (ac *addCmd) resolveMentions() ([]jira.CommentMention, error) {
	if len(ac.params.mentions) == 0 {
		return nil, nil
	}

	project := strings.SplitN(ac.params.issueKey, "-", 2)[0]
	resolved := make([]jira.CommentMention, 0, len(ac.params.mentions))
	for _, value := range ac.params.mentions {
		query := strings.TrimSpace(strings.TrimPrefix(value, "@"))
		if query == "" {
			return nil, fmt.Errorf("mention query cannot be empty")
		}

		users, err := api.ProxySearchUsers(ac.client, &jira.UserSearchOptions{
			Query:      query,
			MaxResults: 20,
			Project:    project,
		})
		if err != nil {
			return nil, fmt.Errorf("searching for mention %q: %w", query, err)
		}

		user, err := findMentionUser(query, users)
		if err != nil {
			return nil, err
		}

		displayName := user.DisplayName
		if displayName == "" {
			displayName = user.Name
		}
		resolved = append(resolved, jira.CommentMention{
			Text:      "@" + displayName,
			AccountID: user.AccountID,
			Name:      user.Name,
		})
	}

	return resolved, nil
}

func findMentionUser(query string, users []*jira.User) (*jira.User, error) {
	matches := make([]*jira.User, 0, len(users))
	for _, user := range users {
		if strings.EqualFold(query, user.DisplayName) ||
			strings.EqualFold(query, user.Name) ||
			strings.EqualFold(query, user.Email) ||
			strings.EqualFold(query, user.AccountID) {
			matches = append(matches, user)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("mention query %q matches multiple users", query)
	}
	candidates := make([]string, 0, len(users))
	for _, user := range users {
		displayName := user.DisplayName
		if displayName == "" {
			displayName = user.Name
		}
		if displayName != "" {
			candidates = append(candidates, displayName)
		}
	}
	if len(candidates) > 0 {
		return nil, fmt.Errorf("mention query %q did not match an exact user; candidates: %s", query, strings.Join(candidates, ", "))
	}
	return nil, fmt.Errorf("mention query %q did not match an exact user", query)
}

func getNextAction() *survey.Question {
	return &survey.Question{
		Name: "action",
		Prompt: &survey.Select{
			Message: "What's next?",
			Options: []string{
				cmdcommon.ActionSubmit,
				cmdcommon.ActionCancel,
			},
		},
		Validate: survey.Required,
	}
}

func (ac *addCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData() || ac.params.template == "-"
}

func (ac *addCmd) isMandatoryParamsMissing() bool {
	return ac.params.issueKey == ""
}
