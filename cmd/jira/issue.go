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

	issues, total := func() ([]*jira.Issue, int) {
		s := info("Fetching issues...")
		defer s.Stop()

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
		"Accepts: today, week, month, year")
	issueCmd.Flags().String("updated", "", "Filter issues by updated date\n"+
		"Accepts: today, week, month, year")
	issueCmd.Flags().StringArrayP("label", "l", []string{}, "Filter issues by label")
	issueCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")
}
