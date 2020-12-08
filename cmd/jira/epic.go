package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

var epicCmd = &cobra.Command{
	Use:     "epic",
	Short:   "Epic lists top 50 epics",
	Long:    `Epic lists top 50 epics.`,
	Aliases: []string{"epics"},
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		list, err := cmd.Flags().GetBool("list")
		exitIfError(err)

		err = cmd.Flags().Set("type", "Epic")
		exitIfError(err)

		if list {
			issue(cmd, args)
		} else {
			epic(cmd, args)
		}
	},
}

func epic(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	if len(args) == 0 {
		epicExplorerView(cmd.Flags(), project, server)
	} else {
		singleEpicView(cmd.Flags(), args[0], project, server)
	}
}

func singleEpicView(flags query.FlagParser, key, project, server string) {
	err := flags.Set("type", "") // Unset issue type.
	exitIfError(err)

	q, err := query.NewIssue(project, flags)
	exitIfError(err)

	resp, err := jiraClient.EpicIssues(key, q.Get())
	exitIfError(err)

	if resp.Total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)

		return
	}

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   resp.Total,
		Data:    resp.Issues,
	}

	exitIfError(v.Render())
}

func epicExplorerView(flags query.FlagParser, project, server string) {
	q, err := query.NewIssue(project, flags)
	exitIfError(err)

	resp, err := jiraClient.Search(q.Get())
	exitIfError(err)

	if resp.Total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)

		return
	}

	v := view.EpicList{
		Total:   resp.Total,
		Project: project,
		Server:  server,
		Data:    resp.Issues,
		Issues: func(key string) []jira.Issue {
			resp, err := jiraClient.EpicIssues(key, "")
			if err != nil {
				return []jira.Issue{}
			}

			return resp.Issues
		},
	}

	exitIfError(v.Render())
}

func init() {
	rootCmd.AddCommand(epicCmd)

	epicCmd.Flags().Bool("history", false, "Epics you accessed recently")
	epicCmd.Flags().BoolP("watching", "w", false, "Epics you are watching")
	epicCmd.Flags().StringP("type", "t", "", "Filter epics by type")
	epicCmd.Flags().StringP("resolution", "r", "", "Filter epics by resolution type")
	epicCmd.Flags().StringP("status", "s", "", "Filter epics by status")
	epicCmd.Flags().StringP("priority", "y", "", "Filter epics by priority")
	epicCmd.Flags().StringP("reporter", "e", "", "Filter epics by reporter (email or display name)")
	epicCmd.Flags().StringP("assignee", "a", "", "Filter epics by assignee (email or display name)")
	epicCmd.Flags().String("created", "", "Filter issues by created date\n"+
		"Accepts: today, week, month, year")
	epicCmd.Flags().String("updated", "", "Filter issues by updated date\n"+
		"Accepts: today, week, month, year")
	epicCmd.Flags().StringArrayP("label", "l", []string{}, "Filter epics by label")
	epicCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")
	epicCmd.Flags().Bool("list", false, "Display epics in list view")

	exitIfError(epicCmd.Flags().MarkHidden("type"))
}
