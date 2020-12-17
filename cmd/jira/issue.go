package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue lists issues in a project",
	Long: `Issue lists issues in a given project.

You can combine different flags together to create a unique query. For instance,
	
	# Issues that are of high priority, is in progress, was created this month,
	# and has given labels
	jira issue -yHigh -s"In Progress" --created month -lbackend -l"high prio"

By default issues are displayed in a interactive list view. You can use --plain flag
to display output in a plain text mode. --no-headers flag will hide the table headers
in plain view.

	# Display issues in a plain table view without headers
	jira issue --plain --no-headers
`,
	Aliases: []string{"issues", "list"},
	Run:     issue,
}

func issue(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	plain, err := cmd.Flags().GetBool("plain")
	exitIfError(err)

	issues, total := func() ([]*jira.Issue, int) {
		if !plain {
			s := info("Fetching issues...")
			defer s.Stop()
		}

		q, err := query.NewIssue(project, cmd.Flags())
		exitIfError(err)

		resp, err := jiraClient.Search(q.Get())
		exitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)
		return
	}

	noHeaders, err := cmd.Flags().GetBool("no-headers")
	exitIfError(err)

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
		Display: view.DisplayFormat{
			Plain:     plain,
			NoHeaders: noHeaders,
		},
	}

	exitIfError(v.Render())
}

func init() {
	rootCmd.AddCommand(issueCmd)

	injectIssueFlags(issueCmd)
}

func injectIssueFlags(cmd *cobra.Command) {
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
}
