package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jql"
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
	exitIfError(err)

	watching, err := cmd.Flags().GetBool("watching")
	exitIfError(err)

	resolution, err := cmd.Flags().GetString("resolution")
	exitIfError(err)

	issueType, err := cmd.Flags().GetString("type")
	exitIfError(err)

	status, err := cmd.Flags().GetString("status")
	exitIfError(err)

	priority, err := cmd.Flags().GetString("priority")
	exitIfError(err)

	reporter, err := cmd.Flags().GetString("reporter")
	exitIfError(err)

	assignee, err := cmd.Flags().GetString("assignee")
	exitIfError(err)

	reverse, err := cmd.Flags().GetBool("reverse")
	exitIfError(err)

	obf := "created"
	project := viper.GetString("project")

	q := jql.NewJQL(project)

	q.And(func() {
		if latest {
			q.History()
			obf = "lastViewed"
		}

		if watching {
			q.Watching()
		}

		q.FilterBy("type", issueType).
			FilterBy("resolution", resolution).
			FilterBy("status", status).
			FilterBy("priority", priority).
			FilterBy("reporter", reporter).
			FilterBy("assignee", assignee)
	})

	if reverse {
		q.OrderBy(obf, jql.DirectionAscending)
	} else {
		q.OrderBy(obf, jql.DirectionDescending)
	}

	resp, err := jiraClient.Search(q.String())
	exitIfError(err)

	if resp.Total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)

		return
	}

	v := view.List{
		Total:   resp.Total,
		Project: project,
		Data:    resp.Issues,
	}

	exitIfError(v.Render())
}

func init() {
	rootCmd.AddCommand(issueCmd)

	issueCmd.Flags().BoolP("latest", "l", false, "Latest issues based on your activity")
	issueCmd.Flags().BoolP("watching", "w", false, "Issues you are watching")
	issueCmd.Flags().StringP("type", "t", "", "Filter issues by type")
	issueCmd.Flags().StringP("resolution", "r", "", "Filter issues by resolution type")
	issueCmd.Flags().StringP("status", "s", "", "Filter issues by status")
	issueCmd.Flags().StringP("priority", "y", "", "Filter issues by priority")
	issueCmd.Flags().StringP("reporter", "e", "", "Filter issues by reporter (email or display name)")
	issueCmd.Flags().StringP("assignee", "a", "", "Filter issues by assignee (email or display name)")
	issueCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")
}
