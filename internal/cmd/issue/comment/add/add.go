package add

import (
	"fmt"

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

	if !params.noInput {
		answer := struct{ Action string }{}
		err := survey.Ask([]*survey.Question{getNextAction()}, &answer)
		cmdutil.ExitIfError(err)

		if answer.Action == cmdcommon.ActionCancel {
			cmdutil.Failed("Action aborted")
		}
	}

	err := func() error {
		s := cmdutil.Info("Adding comment")
		defer s.Stop()

		return client.AddIssueComment(ac.params.issueKey, ac.params.body)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Comment added to issue %q", ac.params.issueKey)
	fmt.Printf("%s/browse/%s\n", server, ac.params.issueKey)

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

	return &addParams{
		issueKey: issueKey,
		body:     body,
		template: template,
		noInput:  noInput,
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
