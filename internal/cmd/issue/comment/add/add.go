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
`
)

// NewCmdCommentAdd is a comment add command.
func NewCmdCommentAdd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "add ISSUE_KEY COMMENT_BODY",
		Short:   "Add adds comment to an issue",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "ISSUE_KEY\tIssue key of the source issue, eg: ISSUE-1\n" +
				"COMMENT_BODY\tBody of the comment you want to add",
		},
		Run: add,
	}

	cmd.Flags().Bool("web", false, "Open issue in web browser after adding comment")

	return &cmd
}

func add(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.Client(jira.Config{Debug: params.debug})
	ac := addCmd{
		client:    client,
		linkTypes: nil,
		params:    params,
	}

	cmdutil.ExitIfError(ac.setIssueKey())

	qs := ac.getQuestions()
	if len(qs) > 0 {
		ans := struct{ IssueKey, Body string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.issueKey == "" {
			params.issueKey = ans.IssueKey
		}
		if params.body == "" {
			params.body = ans.Body
		}

		answer := struct{ Action string }{}
		err = survey.Ask([]*survey.Question{ac.getNextAction()}, &answer)
		cmdutil.ExitIfError(err)

		if answer.Action == cmdcommon.ActionCancel {
			cmdutil.Errorf("\033[0;31m✗\033[0m Action aborted")
		}
	}

	func() {
		s := cmdutil.Info("Adding comment")
		defer s.Stop()

		err := client.AddIssueComment(ac.params.issueKey, ac.params.body)
		cmdutil.ExitIfError(err)
	}()

	server := viper.GetString("server")

	fmt.Printf("\u001B[0;32m✓\u001B[0m Comment added to issue \"%s\"\n", ac.params.issueKey)
	fmt.Printf("%s/browse/%s\n", server, ac.params.issueKey)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, ac.params.issueKey)
		cmdutil.ExitIfError(err)
	}
}

type addParams struct {
	issueKey string
	body     string
	debug    bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *addParams {
	var issueKey, body string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = strings.ToUpper(args[0])
	}
	if nargs >= 2 {
		body = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &addParams{
		issueKey: issueKey,
		body:     body,
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
	ac.params.issueKey = ans

	return nil
}

func (ac *addCmd) getQuestions() []*survey.Question {
	var qs []*survey.Question

	if ac.params.issueKey == "" {
		qs = append(qs, &survey.Question{
			Name:     "issueKey",
			Prompt:   &survey.Input{Message: "Issue key"},
			Validate: survey.Required,
		})
	}
	if ac.params.body == "" {
		qs = append(qs, &survey.Question{
			Name: "body",
			Prompt: &surveyext.JiraEditor{
				Editor:       &survey.Editor{Message: "Comment body", HideDefault: true},
				BlankAllowed: false,
			},
		})
	}

	return qs
}

func (ac *addCmd) getNextAction() *survey.Question {
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
