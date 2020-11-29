package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/view"
)

var issueCmd = &cobra.Command{
	Use:     "issue",
	Short:   "List issues in a project",
	Long:    `List lists all issues in a given project.`,
	Aliases: []string{"issues", "list"},
	Run:     issues,
}

func issues(cmd *cobra.Command, _ []string) {
	latest, err := cmd.Flags().GetBool("latest")
	if err != nil {
		exitWithError(err)
	}

	watching, err := cmd.Flags().GetBool("watching")
	if err != nil {
		exitWithError(err)
	}

	resolution, err := cmd.Flags().GetString("resolution")
	if err != nil {
		exitWithError(err)
	}

	status, err := cmd.Flags().GetString("status")
	if err != nil {
		exitWithError(err)
	}

	priority, err := cmd.Flags().GetString("priority")
	if err != nil {
		exitWithError(err)
	}

	reverse, err := cmd.Flags().GetBool("reverse")
	if err != nil {
		exitWithError(err)
	}

	obf := "created"
	project := viper.GetString("project")
	jql := fmt.Sprintf("project=\"%s\"", project)

	if latest {
		jql += " AND issue IN issueHistory()"
		obf = "lastViewed"
	}

	if watching {
		jql += " AND issue in watchedIssues()"
	}

	if resolution != "" {
		jql += fmt.Sprintf(" AND resolution=\"%s\"", resolution)
	}

	if status != "" {
		jql += fmt.Sprintf(" AND status=\"%s\"", status)
	}

	if priority != "" {
		jql += fmt.Sprintf(" AND priority=\"%s\"", priority)
	}

	jql += fmt.Sprintf(" ORDER BY %s", obf)

	if reverse {
		jql += " ASC"
	} else {
		jql += " DESC"
	}

	resp, err := jiraClient.Search(jql)
	if err != nil {
		exitWithError(err)
	}

	if resp.Total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)

		return
	}

	v := view.List{
		Total:   resp.Total,
		Project: project,
		Data:    resp.Issues,
	}

	if err := v.Render(); err != nil {
		exitWithError(err)
	}
}

func init() {
	rootCmd.AddCommand(issueCmd)

	issueCmd.Flags().BoolP("latest", "l", false, "Latest issues based on user activity")
	issueCmd.Flags().BoolP("watching", "w", false, "Issues that a user is watching")
	issueCmd.Flags().StringP("resolution", "r", "", "Filter issues by resolution type")
	issueCmd.Flags().StringP("status", "s", "", "Filter issues by status")
	issueCmd.Flags().StringP("priority", "y", "", "Filter issues by priority")
	issueCmd.Flags().BoolP("reverse", "v", false, "Reverse the display order (default is DESC)")
}
