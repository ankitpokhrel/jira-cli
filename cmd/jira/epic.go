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
	Use:     "epic [ISSUE KEY]",
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

	plain, err := flags.GetBool("plain")
	exitIfError(err)

	issues, total := func() ([]*jira.Issue, int) {
		if !plain {
			s := info("Fetching epic issues...")
			defer s.Stop()
		}

		q, err := query.NewIssue(project, flags)
		exitIfError(err)

		resp, err := jiraClient.EpicIssues(key, q.Get())
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

func epicExplorerView(flags query.FlagParser, project, server string) {
	epics, total := func() ([]*jira.Issue, int) {
		s := info("Fetching epics...")
		defer s.Stop()

		q, err := query.NewIssue(project, flags)
		exitIfError(err)

		resp, err := jiraClient.Search(q.Get())
		exitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)

		return
	}

	v := view.EpicList{
		Total:   total,
		Project: project,
		Server:  server,
		Data:    epics,
		Issues: func(key string) []*jira.Issue {
			resp, err := jiraClient.EpicIssues(key, "")
			if err != nil {
				return []*jira.Issue{}
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
	epicCmd.Flags().String("created", "", "Filter epics by created date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Created filter will have precedence over created-after and created-before filter")
	epicCmd.Flags().String("updated", "", "Filter epics by updated date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Updated filter will have precedence over updated-after and updated-before filter")
	epicCmd.Flags().String("created-after", "", "Filter by epics created after certain date")
	epicCmd.Flags().String("updated-after", "", "Filter by epics updated after certain date")
	epicCmd.Flags().String("created-before", "", "Filter by epics created before certain date")
	epicCmd.Flags().String("updated-before", "", "Filter by epics updated before certain date")
	epicCmd.Flags().StringArrayP("label", "l", []string{}, "Filter epics by label")
	epicCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")
	epicCmd.Flags().Bool("list", false, "Display epics in list view")
	epicCmd.Flags().Bool("plain", false, "Display output in plain mode")

	exitIfError(epicCmd.Flags().MarkHidden("type"))
}
