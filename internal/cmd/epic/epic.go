package epic

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const helpText = `Epic lists top 100 epics.

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
`

// NewCmdEpic is an epic command.
func NewCmdEpic() *cobra.Command {
	cmd := cobra.Command{
		Use:     "epic [ISSUE KEY]",
		Short:   "Epic lists top 100 epics",
		Long:    helpText,
		Aliases: []string{"epics"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			list, err := cmd.Flags().GetBool("list")
			cmdutil.ExitIfError(err)

			err = cmd.Flags().Set("type", "Epic")
			cmdutil.ExitIfError(err)

			if list {
				issue.Issue(cmd, args)
			} else {
				epic(cmd, args)
			}
		},
	}

	issue.SetFlags(&cmd)
	setFlags(&cmd)
	hideFlags(&cmd)

	return &cmd
}

func epic(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.Client(debug)

	if len(args) == 0 {
		epicExplorerView(cmd.Flags(), project, server, client)
	} else {
		singleEpicView(cmd.Flags(), args[0], project, server, client)
	}
}

func singleEpicView(flags query.FlagParser, key, project, server string, client *jira.Client) {
	err := flags.Set("type", "") // Unset issue type.
	cmdutil.ExitIfError(err)

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	issues, total := func() ([]*jira.Issue, int) {
		if !plain {
			s := cmdutil.Info("Fetching epic issues...")
			defer s.Stop()
		}

		q, err := query.NewIssue(project, flags)
		cmdutil.ExitIfError(err)

		resp, err := client.EpicIssues(key, q.Get())
		cmdutil.ExitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		cmdutil.PrintErrF("No result found for given query in project \"%s\"", project)
		return
	}

	noHeaders, err := flags.GetBool("no-headers")
	cmdutil.ExitIfError(err)

	columns, err := flags.GetString("columns")
	cmdutil.ExitIfError(err)

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

	cmdutil.ExitIfError(v.Render())
}

func epicExplorerView(flags query.FlagParser, project, server string, client *jira.Client) {
	epics, total := func() ([]*jira.Issue, int) {
		s := cmdutil.Info("Fetching epics...")
		defer s.Stop()

		q, err := query.NewIssue(project, flags)
		cmdutil.ExitIfError(err)

		resp, err := client.Search(q.Get())
		cmdutil.ExitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		cmdutil.PrintErrF("No result found for given query in project \"%s\"", project)
		return
	}

	v := view.EpicList{
		Total:   total,
		Project: project,
		Server:  server,
		Data:    epics,
		Issues: func(key string) []*jira.Issue {
			resp, err := client.EpicIssues(key, "")
			if err != nil {
				return []*jira.Issue{}
			}
			return resp.Issues
		},
	}

	cmdutil.ExitIfError(v.Render())
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("list", false, "Display epics in list view")
}

func hideFlags(cmd *cobra.Command) {
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("type"))
}
