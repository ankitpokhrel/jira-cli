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
	helpText = `Add adds worklog to an issue.`
	examples = `$ jira issue worklog add

# Pass required parameters and use --no-input to skip prompt
$ jira issue worklog add ISSUE-1 "2d 1h 30m" --no-input

# You can add a comment using --comment flag when adding a worklog
$ jira issue worklog add ISSUE-1 "2d 1h 30m" --comment "This is a comment" --no-input

# You can also add a worklog with the specific start date (defaults to UTC timezone)
$ jira issue worklog add ISSUE-1 "2d 1h 30m" --started "2022-01-01 09:30:00"

# You can specify timezone to use along with the start date in IANA timezone format
$ jira issue worklog add ISSUE-1 3h --started "2022-01-01 09:30:00" --timezone "Europe/Berlin"

# Or, you can use start date in Jira datetime format and skip the timezone flag
$ jira issue worklog add ISSUE-1 "1h 30m" --started "2022-01-01T09:30:00.000+0200"`
)

// NewCmdWorklogAdd is a worklog add command.
func NewCmdWorklogAdd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "add ISSUE-KEY TIME_SPENT",
		Short:   "Add a worklog to an issue",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key of the source issue, eg: ISSUE-1\n" +
				"TIME_SPENT\tTime to log as days (d), hours (h), or minutes (m), separated by space eg: 2d 1h 30m",
		},
		Run: add,
	}

	cmd.Flags().SortFlags = false

	cmd.Flags().String("started", "", "The datetime on which the worklog effort was started, eg: 2022-01-01 09:30:00")
	cmd.Flags().String("timezone", "UTC", "The timezone to use for the started date in IANA timezone format, eg: Europe/Berlin")
	cmd.Flags().String("comment", "", "Comment about the worklog")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")

	return &cmd
}

func add(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)
	ac := addCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(ac.setIssueKey())

	qs := ac.getQuestions()
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
		s := cmdutil.Info("Adding a worklog")
		defer s.Stop()

		return client.AddIssueWorklog(ac.params.issueKey, ac.params.started, ac.params.timeSpent, ac.params.comment)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Worklog added to issue %q", ac.params.issueKey)
	fmt.Printf("%s/browse/%s\n", server, ac.params.issueKey)
}

type addParams struct {
	issueKey  string
	started   string
	timezone  string
	timeSpent string
	comment   string
	noInput   bool
	debug     bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *addParams {
	var issueKey, timeSpent string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if nargs >= 2 {
		timeSpent = args[1]
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

	return &addParams{
		issueKey:  issueKey,
		started:   startedWithTZ,
		timezone:  timezone,
		timeSpent: timeSpent,
		comment:   comment,
		noInput:   noInput,
		debug:     debug,
	}
}

type addCmd struct {
	client *jira.Client
	params *addParams
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
	var qs []*survey.Question

	if ac.params.timeSpent == "" {
		qs = append(qs, &survey.Question{
			Name: "timeSpent",
			Prompt: &survey.Input{
				Message: "Time spent",
				Help:    "Time to log as days (d), hours (h), or minutes (m), separated by space eg: 2d 1h 30m",
			},
			Validate: survey.Required,
		})
	}

	if !ac.params.noInput && ac.params.comment == "" {
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
