// Package list implements issue listing and reusable list flags for related commands.
package list

import (
	"encoding/json"
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
$ jira issue list -yHigh -s"In Progress" --created month -lbackend -l"high prio"

You can also add an optional search query as a positional argument, which functions the same
as entering a search query into the Jira UI's search box.

Issues are displayed in an interactive list view by default. You can use a --plain flag
to display output in a plain text mode. A --no-headers flag will hide the table headers
in plain view. A --no-truncate flag will display all available fields in plain mode.

In the in-app details opened with v, press / to view another ticket by key, or enter
a number for the current ticket's project. Esc/CtrlC cancels the prompt. Switching
tickets preserves the underlying list; q/Esc from details still returns to that list.

Experimental --comment-threads reads native threads in in-app issue details on Jira
Cloud using API-token Basic auth. Server/local installations and other authentication
types are unsupported. --comments counts whole root threads with all replies in this
mode; 0 hides comments. Roots are newest-first, replies oldest-first. Default flat
comments and plain/CSV tables are unchanged. Read-only thread queries use private
GraphQL persisted operations, which may change; failures are reported rather than
guessing reply relationships. Uses configured credentials, not HAR files or browser
cookies. In in-app details, c posts a top-level comment and r chooses a thread for a
reply. Replies use a separate mutation after explicit Post and support unrestricted
Software/Business threads only.`

	examples = `$ jira issue list

# Limit list to 20 items
$ jira issue list --paginate 20

# Get 50 items starting from 10
$ jira issue list --paginate 10:50

# Search for issues containing specific text
$ jira issue list "Feature Request"

# List issues in a plain table view without headers
$ jira issue list --plain --no-headers

# List some columns of the issue in a plain table view
$ jira issue list --plain --columns key,assignee,status

# List issues in a plain table view and show all fields
$ jira issue list --plain --no-truncate

# List issues in a plain table view using custom delimiter (default is "\t")
$ jira issue list --plain --delimiter "|"

# Show 5 complete native comment threads in issue details (experimental Jira Cloud, API-token Basic auth)
$ jira issue list --comment-threads --comments 5

# List issues as raw JSON data
$ jira issue list --raw

# List issues of type "Epic" in status "Done"
$ jira issue list -tEpic -sDone

# List issues in status other than "Open" and is assigned to no one
$ jira issue list -s~Open -ax

# List issues from all projects
$ jira issue list -q"project IS NOT EMPTY"`
)

// NewCmdList is a list command.
func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [optional text to query]",
		Short:   "List lists issues in a project",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"lists", "ls", "search"},
		Args:    cobra.RangeArgs(0, 1),
		Run:     List,
	}
	cmd.Flags().Uint("comments", 1, "Show N comments in issue details, or N whole root threads with all replies with --comment-threads (0 hides comments)")
	cmd.Flags().Bool("comment-threads", false, "Read native comment threads in issue details (experimental Jira Cloud only; API-token Basic auth)")
	cmd.Flags().Bool("raw", false, "Print raw JSON output")
	cmd.MarkFlagsMutuallyExclusive("raw", "comment-threads")
	return cmd
}

// List displays a list view.
func List(cmd *cobra.Command, args []string) {
	loadList(cmd, args)
}

// loadList rebuilds the query and view on refresh, preserving explicit comment
// limits (including zero) over configuration defaults. Command errors terminate via cmdutil.
func loadList(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")
	numComments := viper.GetUint("num_comments")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	pk, err := cmd.Flags().GetString("parent")
	cmdutil.ExitIfError(err)

	err = cmd.Flags().Set("parent", cmdutil.GetJiraIssueKey(project, pk))
	cmdutil.ExitIfError(err)

	if len(args) > 0 {
		searchQuery := fmt.Sprintf(`text ~ %q`, strings.Join(args, " "))

		jqlFlag, err := cmd.Flags().GetString("jql")
		cmdutil.ExitIfError(err)
		if jqlFlag != "" {
			searchQuery = fmt.Sprintf(`%s AND %s`, jqlFlag, searchQuery)
		}
		cmdutil.ExitIfError(cmd.Flags().Set("jql", searchQuery))
	}

	issues, err := func() ([]*jira.Issue, error) {
		s := cmdutil.Info("Fetching issues...")
		defer s.Stop()

		q, err := query.NewIssue(project, cmd.Flags())
		if err != nil {
			return nil, err
		}

		resp, err := api.ProxySearch(api.DefaultClient(debug), q.Get(), q.Params().From, q.Params().Limit)
		if err != nil {
			return nil, err
		}

		return resp.Issues, nil
	}()
	cmdutil.ExitIfError(err)

	if len(issues) == 0 {
		fmt.Println()
		cmdutil.Failed("No result found for given query in project %q", project)
		return
	}

	raw, err := cmd.Flags().GetBool("raw")
	cmdutil.ExitIfError(err)

	if raw {
		outputRawJSON(issues)
		return
	}

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	delimiter, err := cmd.Flags().GetString("delimiter")
	cmdutil.ExitIfError(err)

	csv, err := cmd.Flags().GetBool("csv")
	cmdutil.ExitIfError(err)

	noHeaders, err := cmd.Flags().GetBool("no-headers")
	cmdutil.ExitIfError(err)

	noTruncate, err := cmd.Flags().GetBool("no-truncate")
	cmdutil.ExitIfError(err)

	fixedColumns, err := cmd.Flags().GetUint("fixed-columns")
	cmdutil.ExitIfError(err)

	columns, err := cmd.Flags().GetString("columns")
	cmdutil.ExitIfError(err)

	var comments uint
	if cmd.Flags().Changed("comments") {
		comments, err = cmd.Flags().GetUint("comments")
		cmdutil.ExitIfError(err)
	} else {
		comments = max(numComments, 1)
	}

	var commentThreads bool
	if cmd.Flags().Lookup("comment-threads") != nil {
		commentThreads, err = cmd.Flags().GetBool("comment-threads")
		cmdutil.ExitIfError(err)
	}

	v := view.IssueList{
		Project: project,
		Server:  server,
		Data:    issues,
		Refresh: func() {
			loadList(cmd, args)
		},
		Display: view.DisplayFormat{
			Plain:          plain,
			Delimiter:      delimiter,
			CSV:            csv,
			NoHeaders:      noHeaders,
			NoTruncate:     noTruncate,
			FixedColumns:   fixedColumns,
			Comments:       comments,
			CommentThreads: commentThreads,
			Columns: func() []string {
				if columns != "" {
					return strings.Split(columns, ",")
				}
				return []string{}
			}(),
			TableStyle: cmdutil.GetTUIStyleConfig(),
			Timezone:   viper.GetString("timezone"),
		},
	}

	cmdutil.ExitIfError(v.Render())
}

// outputRawJSON bypasses view formatting and reports encoding failures without
// printing a partial JSON document.
func outputRawJSON(issues []*jira.Issue) {
	data, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		cmdutil.Failed("Failed to marshal issues to JSON: %s", err)
		return
	}
	fmt.Println(string(data))
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
	cmd.Flags().String("delimiter", "\t", "Custom delimiter for columns in plain mode. Works only with --plain")
	if cmd.Flags().Lookup("comments") == nil {
		cmd.Flags().Uint("comments", 1, "Show N comments when viewing the issue")
	}
	if cmd.Flags().Lookup("raw") == nil {
		cmd.Flags().Bool("raw", false, "Print raw JSON output")
	}
	cmd.Flags().Bool("csv", false, "Print output in CSV format")

	if cmd.HasParent() && cmd.Parent().Name() != "sprint" {
		cmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
			fmt.Sprintf("Accepts: %s", strings.Join(view.ValidIssueColumns(), ", ")))
		cmd.Flags().Uint("fixed-columns", 1, "Number of fixed columns in the interactive mode")
	}
}
