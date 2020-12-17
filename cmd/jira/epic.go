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

	epicCmd.Flags().Bool("list", false, "Display epics in list view")

	injectIssueFlags(epicCmd)

	exitIfError(epicCmd.Flags().MarkHidden("type"))
}
