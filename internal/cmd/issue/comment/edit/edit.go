package edit

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
	"github.com/ankitpokhrel/jira-cli/pkg/md"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const (
	helpText = `Edit an existing comment on an issue.`
	examples = `$ jira issue comment edit ISSUE-1 10042

# Edit with inline body (non-interactive)
$ jira issue comment edit ISSUE-1 10042 -b "Updated text" --no-input

# Mark comment as internal
$ jira issue comment edit ISSUE-1 10042 --internal --no-input`
)

// NewCmdCommentEdit is a comment edit command.
func NewCmdCommentEdit() *cobra.Command {
	cmd := cobra.Command{
		Use:     "edit ISSUE-KEY COMMENT-ID",
		Short:   "Edit a comment on an issue",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key of the issue, eg: ISSUE-1\n" +
				"COMMENT-ID\tNumeric ID of the comment to edit",
		},
		Args: cobra.MinimumNArgs(2),
		Run:  edit,
	}

	cmd.Flags().StringP("body", "b", "", "Edit comment body")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")
	cmd.Flags().Bool("internal", false, "Mark comment as internal")
	cmd.Flags().Bool("raw", false, "Pass comment body as raw Jira wiki markup")

	return &cmd
}

func edit(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)
	ec := editCmd{
		client: client,
		params: params,
	}

	if ec.isNonInteractive() {
		ec.params.noInput = true
	}

	// Always fetch existing comment: validates the ID and provides pre-fill body.
	existing, err := func() (*jira.IssueComment, error) {
		s := cmdutil.Info(fmt.Sprintf("Fetching comment %s...", params.commentID))
		defer s.Stop()

		return client.GetIssueComment(params.issueKey, params.commentID)
	}()
	cmdutil.ExitIfError(err)

	originalBody := md.FromJiraMD(existing.Body)

	// In non-interactive mode without an explicit body, keep the existing body.
	// This allows changing only the --internal flag without editing text.
	if params.noInput && params.body == "" {
		params.body = originalBody
	}

	qs := ec.getQuestions(originalBody)
	if len(qs) > 0 {
		ans := struct{ Body string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		params.body = ans.Body
	}

	// In interactive mode, skip the PUT if the body was not changed.
	if !params.noInput && params.body == originalBody {
		cmdutil.Warn("No changes detected, comment was not updated")
		return
	}

	if !params.noInput {
		answer := struct{ Action string }{}
		err := survey.Ask([]*survey.Question{getNextAction()}, &answer)
		cmdutil.ExitIfError(err)

		if answer.Action == cmdcommon.ActionCancel {
			cmdutil.Failed("Action aborted")
		}
	}

	body := ec.params.body
	if !ec.params.raw {
		body = md.ToJiraMD(body)
	}

	err = func() error {
		s := cmdutil.Info("Updating comment")
		defer s.Stop()

		return client.UpdateIssueComment(ec.params.issueKey, ec.params.commentID, body, ec.params.internal)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Comment updated on issue %q", ec.params.issueKey)
	fmt.Printf("%s\n", cmdutil.GenerateServerBrowseURL(server, ec.params.issueKey))
}

type editParams struct {
	issueKey  string
	commentID string
	body      string
	noInput   bool
	internal  bool
	raw       bool
	debug     bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *editParams {
	var issueKey, commentID string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if nargs >= 2 {
		commentID = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	body, err := flags.GetString("body")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	internal, err := flags.GetBool("internal")
	cmdutil.ExitIfError(err)

	raw, err := flags.GetBool("raw")
	cmdutil.ExitIfError(err)

	return &editParams{
		issueKey:  issueKey,
		commentID: commentID,
		body:      body,
		noInput:   noInput,
		internal:  internal,
		raw:       raw,
		debug:     debug,
	}
}

type editCmd struct {
	client *jira.Client
	params *editParams
}

func (ec *editCmd) getQuestions(originalBody string) []*survey.Question {
	var qs []*survey.Question

	// Skip editor if noInput is set or body was already provided via flag.
	if ec.params.noInput || ec.params.body != "" {
		return qs
	}

	qs = append(qs, &survey.Question{
		Name: "body",
		Prompt: &surveyext.JiraEditor{
			Editor: &survey.Editor{
				Message:       "Comment body",
				Default:       originalBody,
				HideDefault:   true,
				AppendDefault: true,
			},
			BlankAllowed: false,
		},
	})

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

func (ec *editCmd) isNonInteractive() bool {
	return ec.params.noInput
}
