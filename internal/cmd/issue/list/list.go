package list

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

const (
	helpText = `List lists issues in a given project.

You can combine different flags to create a unique query. For instance,
	
# Issues that are of high priority, is in progress, was created this month, and has given labels
jira issue list -yHigh -s"In Progress" --created month -lbackend -l"high prio"

Issues are displayed in an interactive list view by default. You can use a --plain flag
to display output in a plain text mode. A --no-headers flag will hide the table headers
in plain view. A --no-truncate flag will display all available fields in plain mode.`

	examples = `$ jira issue list

# Limit list to 20 items
$ jira issue list --paginate 20

# Get 50 items starting from 10
$ jira issue list --paginate 10:50

# List issues in a plain table view without headers
$ jira issue list --plain --no-headers

# List some columns of the issue in a plain table view
$ jira issue list --plain --columns key,assignee,status

# List issues in a plain table view and show all fields
$ jira issue list --plain --no-truncate

# List issues of type "Epic" in status "Done"
$ jira issue list -tEpic -sDone

# List issues in status other than "Open" and is assigned to no one
$ jira issue list -s~Open -ax

# List issues from all projects
$ jira issue list -q"project IS NOT EMPTY"`
)

// NewCmdList is a list command.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List lists issues in a project",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"lists", "ls"},
		Run:     List,
	}
}

// List displays a list view.
func List(cmd *cobra.Command, _ []string) {
	loadList(cmd)
}

func loadList(cmd *cobra.Command) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	pk, err := cmd.Flags().GetString("parent")
	cmdutil.ExitIfError(err)

	err = cmd.Flags().Set("parent", cmdutil.GetJiraIssueKey(project, pk))
	cmdutil.ExitIfError(err)

	issues, total, err := func() ([]*jira.Issue, int, error) {
		s := cmdutil.Info("Fetching issues...")
		defer s.Stop()

		q, err := query.NewIssue(project, cmd.Flags())
		if err != nil {
			return nil, 0, err
		}

		resp, err := api.ProxySearch(api.DefaultClient(debug), q.Get(), q.Params().From, q.Params().Limit)
		if err != nil {
			return nil, 0, err
		}

		return resp.Issues, resp.Total, nil
	}()
	cmdutil.ExitIfError(err)

	if total == 0 {
		fmt.Println()
		cmdutil.Failed("No result found for given query in project %q", project)
		return
	}

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := cmd.Flags().GetBool("no-headers")
	cmdutil.ExitIfError(err)

	noTruncate, err := cmd.Flags().GetBool("no-truncate")
	cmdutil.ExitIfError(err)

	fixedColumns, err := cmd.Flags().GetUint("fixed-columns")
	cmdutil.ExitIfError(err)

	columns, err := cmd.Flags().GetString("columns")
	cmdutil.ExitIfError(err)

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
		Refresh: func() {
			loadList(cmd)
		},
		Display: view.DisplayFormat{
			Plain:        plain,
			NoHeaders:    noHeaders,
			NoTruncate:   noTruncate,
			FixedColumns: fixedColumns,
			Columns: func() []string {
				if columns != "" {
					return strings.Split(columns, ",")
				}
				return []string{}
			}(),
			TableStyle: cmdutil.GetTUIStyleConfig(),
		},
	}

	cmdutil.ExitIfError(v.Render())
}

// SetFlags sets flags supported by a list command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("type", "t", "", "Filter issues by type")
	cmd.Flags().StringP("resolution", "R", "", "Filter issues by resolution type")
	cmd.Flags().StringArrayP("status", "s", []string{}, "Filter issues by status")
	cmd.Flags().StringP("priority", "y", "", "Filter issues by priority")
	cmd.Flags().StringP("reporter", "r", "", "Filter issues by reporter (email or display name)")
	cmd.Flags().StringP("assignee", "a", "", "Filter issues by assignee (email or display name)")
	cmd.Flags().StringP("component", "C", "", "Filter issues by component")
	cmd.Flags().StringArrayP("label", "l", []string{}, "Filter issues by label")
	cmd.Flags().StringP("parent", "P", "", "Filter issues by parent")
	cmd.Flags().Bool("history", false, "Issues you accessed recently")
	cmd.Flags().BoolP("watching", "w", false, "Issues you are watching")
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
	cmd.Flags().StringP("jql", "q", "", "Run a raw JQL query in a given project context")
	cmd.Flags().String("order-by", "created", "Field to order the list with")
	cmd.Flags().Bool("reverse", false, "Reverse the display order (default \"DESC\")")
	cmd.Flags().String("paginate", "0:100", "Paginate the result. Max 100 at a time, format: <from>:<limit> where <from> is optional")
	cmd.Flags().Bool("plain", false, "Display output in plain mode")
	cmd.Flags().Bool("no-headers", false, "Don't display table headers in plain mode. Works only with --plain")
	cmd.Flags().Bool("no-truncate", false, "Show all available columns in plain mode. Works only with --plain")

	if cmd.HasParent() && cmd.Parent().Name() != "sprint" {
		cmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
			fmt.Sprintf("Accepts: %s", strings.Join(view.ValidIssueColumns(), ", ")))
		cmd.Flags().Uint("fixed-columns", 1, "Number of fixed columns in the interactive mode")
	}
}
