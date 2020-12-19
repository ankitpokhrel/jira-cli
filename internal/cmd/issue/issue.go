package issue

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const helpText = `Issue lists issues in a given project.

You can combine different flags to create a unique query. For instance,
	
	# Issues that are of high priority, is in progress, was created this month,
	# and has given labels
	jira issue -yHigh -s"In Progress" --created month -lbackend -l"high prio"

Issues are displayed in an interactive list view by default. You can use a --plain flag
to display output in a plain text mode. A --no-headers flag will hide the table headers
in plain view. A --no-truncate flag will display all available fields in plain mode.

EG:
	# Display issues in a plain table view without headers
	jira issue --plain --no-headers

	# Display some columns of the issue in a plain table view
	jira issue --plain --columns key,assignee,status

	# Display issues in a plain table view and show all fields
	jira issue --plain --no-truncate
`

// NewCmdIssue is an issue command.
func NewCmdIssue() *cobra.Command {
	cmd := cobra.Command{
		Use:     "issue",
		Short:   "Issue lists issues in a project",
		Long:    helpText,
		Aliases: []string{"issues", "list"},
		Run:     Issue,
	}

	SetFlags(&cmd)

	return &cmd
}

// SetFlags sets flags supported by an issue command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("history", false, "Issues you accessed recently")
	cmd.Flags().BoolP("watching", "w", false, "Issues you are watching")
	cmd.Flags().StringP("type", "t", "", "Filter issues by type")
	cmd.Flags().StringP("resolution", "R", "", "Filter issues by resolution type")
	cmd.Flags().StringP("status", "s", "", "Filter issues by status")
	cmd.Flags().StringP("priority", "y", "", "Filter issues by priority")
	cmd.Flags().StringP("reporter", "r", "", "Filter issues by reporter (email or display name)")
	cmd.Flags().StringP("assignee", "a", "", "Filter issues by assignee (email or display name)")
	cmd.Flags().String("created", "", "Filter issues by created date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Created filter will have precedence over created-after and created-before filter")
	cmd.Flags().String("updated", "", "Filter issues by updated date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Updated filter will have precedence over updated-after and updated-before filter")
	cmd.Flags().String("created-after", "", "Filter by issues created after certain date")
	cmd.Flags().String("updated-after", "", "Filter by issues updated after certain date")
	cmd.Flags().String("created-before", "", "Filter by issues created before certain date")
	cmd.Flags().String("updated-before", "", "Filter by issues updated before certain date")
	cmd.Flags().StringArrayP("label", "l", []string{}, "Filter issues by label")
	cmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")
	cmd.Flags().Bool("plain", false, "Display output in plain mode")
	cmd.Flags().Bool("no-headers", false, "Don't display table headers in plain mode. Works only with --plain")
	cmd.Flags().Bool("no-truncate", false, "Show all available columns in plain mode. Works only with --plain")

	if cmd.Name() != "sprint" {
		cmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
			fmt.Sprintf("Accepts: %s", strings.Join(view.ValidIssueColumns(), ", ")))
	}
}

// Issue displays issue list view.
func Issue(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	issues, total := func() ([]*jira.Issue, int) {
		if !plain {
			s := cmdutil.Info("Fetching issues...")
			defer s.Stop()
		}

		q, err := query.NewIssue(project, cmd.Flags())
		cmdutil.ExitIfError(err)

		resp, err := api.Client(jira.Config{Debug: debug}).Search(q.Get())
		cmdutil.ExitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		cmdutil.PrintErrF("No result found for given query in project \"%s\"", project)
		return
	}

	noHeaders, err := cmd.Flags().GetBool("no-headers")
	cmdutil.ExitIfError(err)

	noTruncate, err := cmd.Flags().GetBool("no-truncate")
	cmdutil.ExitIfError(err)

	columns, err := cmd.Flags().GetString("columns")
	cmdutil.ExitIfError(err)

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
		Display: view.DisplayFormat{
			Plain:      plain,
			NoHeaders:  noHeaders,
			NoTruncate: noTruncate,
			Columns: func() []string {
				if columns != "" {
					return strings.Split(columns, ",")
				}
				return []string{}
			}(),
		},
	}

	cmdutil.ExitIfError(v.Render())
}
