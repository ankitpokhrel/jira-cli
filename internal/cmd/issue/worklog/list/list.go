package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `List displays worklogs for an issue.`
	examples = `$ jira issue worklog list ISSUE-1

# List worklogs in plain mode
$ jira issue worklog list ISSUE-1 --plain

# List worklogs in table mode
$ jira issue worklog list ISSUE-1 --table`
)

// NewCmdWorklogList is a worklog list command.
func NewCmdWorklogList() *cobra.Command {
	cmd := cobra.Command{
		Use:     "list ISSUE-KEY",
		Short:   "List worklogs for an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"ls"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key to list worklogs for, eg: ISSUE-1",
		},
		Run: list,
	}

	cmd.Flags().SortFlags = false

	cmd.Flags().Bool("plain", false, "Display output in plain mode")
	cmd.Flags().Bool("table", false, "Display output in table mode")

	return &cmd
}

func list(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)

	if params.issueKey == "" {
		cmdutil.Failed("Issue key is required")
	}

	worklogs, err := func() (*jira.WorklogResponse, error) {
		s := cmdutil.Info(fmt.Sprintf("Fetching worklogs for issue %s...", params.issueKey))
		defer s.Stop()

		return client.GetIssueWorklogs(params.issueKey)
	}()
	cmdutil.ExitIfError(err)

	if worklogs.Total == 0 {
		fmt.Println("No worklogs found")
		return
	}

	server := viper.GetString("server")
	project := viper.GetString("project.key")

	v := view.WorklogList{
		Project:  project,
		Server:   server,
		Worklogs: worklogs.Worklogs,
		Total:    worklogs.Total,
		Display:  params.display,
	}

	cmdutil.ExitIfError(v.Render())
}

type listParams struct {
	issueKey string
	display  view.DisplayFormat
	debug    bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *listParams {
	var issueKey string

	if len(args) >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	_, err = flags.GetBool("table")
	cmdutil.ExitIfError(err)

	return &listParams{
		issueKey: issueKey,
		display:  view.DisplayFormat{Plain: plain},
		debug:    debug,
	}
}
