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
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const (
	helpText = `Edit updates an existing worklog in an issue.`
	examples = `$ jira issue worklog edit

# Edit a specific worklog
$ jira issue worklog edit ISSUE-1 10001 "3h 30m"

# Edit worklog with new comment
$ jira issue worklog edit ISSUE-1 10001 "3h 30m" --comment "Updated work description"

# Edit worklog with new start date
$ jira issue worklog edit ISSUE-1 10001 "3h 30m" --started "2024-11-05 09:30:00"

# Use --no-input to skip prompts
$ jira issue worklog edit ISSUE-1 10001 "3h 30m" --no-input`
)

// NewCmdWorklogEdit is a worklog edit command.
func NewCmdWorklogEdit() *cobra.Command {
	cmd := cobra.Command{
		Use:     "edit ISSUE-KEY WORKLOG-ID TIME_SPENT",
		Short:   "Edit a worklog in an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"update"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key of the source issue, eg: ISSUE-1\n" +
				"WORKLOG-ID\tID of the worklog to edit\n" +
				"TIME_SPENT\tNew time to log as days (d), hours (h), or minutes (m), eg: 2d 1h 30m",
		},
		Run: edit,
	}

	cmd.Flags().SortFlags = false

	cmd.Flags().String("started", "", "The datetime on which the worklog effort was started, eg: 2024-01-01 09:30:00")
	cmd.Flags().String("timezone", "UTC", "The timezone to use for the started date in IANA timezone format, eg: Europe/Berlin")
	cmd.Flags().String("comment", "", "Comment about the worklog")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")

	return &cmd
}

func edit(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)
	ec := editCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(ec.setIssueKey())
	cmdutil.ExitIfError(ec.setWorklogID())

	qs := ec.getQuestions()
	if len(qs) > 0 {
		ans := struct{ TimeSpent, Comment string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.timeSpent == "" {
			params.timeSpent = ans.TimeSpent
		}
		if ans.Comment != "" {
			params.comment = ans.Comment
		}
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
		s := cmdutil.Info("Updating worklog")
		defer s.Stop()

		return client.UpdateIssueWorklog(ec.params.issueKey, ec.params.worklogID, ec.params.started, ec.params.timeSpent, ec.params.comment)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Worklog updated in issue %q", ec.params.issueKey)
	fmt.Printf("%s\n", cmdutil.GenerateServerBrowseURL(server, ec.params.issueKey))
}

type editParams struct {
	issueKey  string
	worklogID string
	started   string
	timezone  string
	timeSpent string
	comment   string
	noInput   bool
	debug     bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *editParams {
	var issueKey, worklogID, timeSpent string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if nargs >= 2 {
		worklogID = args[1]
	}
	if nargs >= 3 {
		timeSpent = args[2]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	started, err := flags.GetString("started")
	cmdutil.ExitIfError(err)

	timezone, err := flags.GetString("timezone")
	cmdutil.ExitIfError(err)

	startedWithTZ, err := cmdutil.DateStringToJiraFormatInLocation(started, timezone)
	cmdutil.ExitIfError(err)

	comment, err := flags.GetString("comment")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	return &editParams{
		issueKey:  issueKey,
		worklogID: worklogID,
		started:   startedWithTZ,
		timezone:  timezone,
		timeSpent: timeSpent,
		comment:   comment,
		noInput:   noInput,
		debug:     debug,
	}
}

type editCmd struct {
	client *jira.Client
	params *editParams
}

func (ec *editCmd) setIssueKey() error {
	if ec.params.issueKey != "" {
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
	ec.params.issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), ans)

	return nil
}

func (ec *editCmd) setWorklogID() error {
	if ec.params.worklogID != "" {
		return nil
	}

	// Fetch worklogs for the issue
	worklogs, err := ec.client.GetIssueWorklogs(ec.params.issueKey)
	if err != nil {
		return err
	}

	if worklogs.Total == 0 {
		return fmt.Errorf("no worklogs found for issue %s", ec.params.issueKey)
	}

	// Create options for selection
	options := make([]string, len(worklogs.Worklogs))
	for i, wl := range worklogs.Worklogs {
		options[i] = fmt.Sprintf("%s - %s by %s (%s)", wl.ID, wl.TimeSpent, wl.Author.Name, wl.Started)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select worklog to edit:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Extract worklog ID from selection (format: "ID - ...")
	var id string
	_, _ = fmt.Sscanf(selected, "%s -", &id)
	ec.params.worklogID = id

	return nil
}

func (ec *editCmd) getQuestions() []*survey.Question {
	var qs []*survey.Question

	if ec.params.timeSpent == "" {
		qs = append(qs, &survey.Question{
			Name: "timeSpent",
			Prompt: &survey.Input{
				Message: "Time spent",
				Help:    "Time to log as days (d), hours (h), or minutes (m), separated by space eg: 2d 1h 30m",
			},
			Validate: survey.Required,
		})
	}

	if !ec.params.noInput && ec.params.comment == "" {
		qs = append(qs, &survey.Question{
			Name: "comment",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Comment body",
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: true,
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
