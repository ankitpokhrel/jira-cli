package jira

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

var epicCmd = &cobra.Command{
	Use:   "epic [ISSUE KEY]",
	Short: "Epic lists top 100 epics",
	Long: `Epic lists top 100 epics.

By default epics are displayed in an explorer view. You can use --list
and --plain flags to display output in different modes.

	# Display epics or epic issues in an interactive list
	jira epic --list
	jira epic <KEY> --list

	# Display epics or epic issues in a plain table view
	jira epic --list --plain
	jira epic <KEY> --list --plain

	# Display epics or epic issues in a plain table view without headers
	jira epic --list --plain --no-headers
	jira epic <KEY> --list --plain --no-headers

	# Display some columns of epic or epic issues in a plain table view
	jira epic --list --plain --columns key,summary,status
	jira epic <KEY> --plain --columns type,key,summary
`,
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
		printErrF("No result found for given query in project \"%s\"", project)
		return
	}

	noHeaders, err := flags.GetBool("no-headers")
	exitIfError(err)

	columns, err := flags.GetString("columns")
	exitIfError(err)

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
		Display: view.DisplayFormat{
			Plain:     plain,
			NoHeaders: noHeaders,
			Columns: func() []string {
				if columns != "" {
					return strings.Split(columns, ",")
				}
				return []string{}
			}(),
		},
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
		printErrF("No result found for given query in project \"%s\"", project)
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
