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
	Use:     "issue",
	Short:   "Issue lists issues in a project",
	Long:    `Issue lists all issues in a given project.`,
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

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
		Plain:   plain,
	}

	exitIfError(v.Render())
}

func init() {
	rootCmd.AddCommand(issueCmd)

	issueCmd.Flags().Bool("history", false, "Issues you accessed recently")
	issueCmd.Flags().BoolP("watching", "w", false, "Issues you are watching")
	issueCmd.Flags().StringP("type", "t", "", "Filter issues by type")
	issueCmd.Flags().StringP("resolution", "r", "", "Filter issues by resolution type")
	issueCmd.Flags().StringP("status", "s", "", "Filter issues by status")
	issueCmd.Flags().StringP("priority", "y", "", "Filter issues by priority")
	issueCmd.Flags().StringP("reporter", "e", "", "Filter issues by reporter (email or display name)")
	issueCmd.Flags().StringP("assignee", "a", "", "Filter issues by assignee (email or display name)")
	issueCmd.Flags().String("created", "", "Filter issues by created date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Created filter will have precedence over created-after and created-before filter")
	issueCmd.Flags().String("updated", "", "Filter issues by updated date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Updated filter will have precedence over updated-after and updated-before filter")
	issueCmd.Flags().String("created-after", "", "Filter by issues created after certain date")
	issueCmd.Flags().String("updated-after", "", "Filter by issues updated after certain date")
	issueCmd.Flags().String("created-before", "", "Filter by issues created before certain date")
	issueCmd.Flags().String("updated-before", "", "Filter by issues updated before certain date")
	issueCmd.Flags().StringArrayP("label", "l", []string{}, "Filter issues by label")
	issueCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")
	issueCmd.Flags().Bool("plain", false, "Display output in plain mode")
}
