package list

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/list"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `List lists top 100 epics.

By default epics are displayed in an explorer view. You can use --table
and --plain flags to display output in different modes.`

	examples = `# Display epics in an explorer view
$ jira epic list

# Display epics or epic issues in an interactive table view
$ jira epic list --table
$ jira epic list <KEY>

# Display epics or epic issues in a plain table view
$ jira epic list --table --plain
$ jira epic list <KEY> --plain

# Display epics or epic issues in a plain table view without headers
$ jira epic list --table --plain --no-headers
$ jira epic list <KEY> --plain --no-headers

# Display some columns of epic or epic issues in a plain table view
$ jira epic list --table --plain --columns key,summary,status
$ jira epic list <KEY> --plain --columns type,key,summary`
)

// NewCmdList is a list command.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list [EPIC_KEY]",
		Short:   "List lists issues in a project",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"lists"},
		Annotations: map[string]string{
			"help:args": "[EPIC_KEY]\tKey for the issue of type epic, eg: ISSUE-1",
		},
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			table, err := cmd.Flags().GetBool("table")
			cmdutil.ExitIfError(err)

			err = cmd.Flags().Set("type", "Epic")
			cmdutil.ExitIfError(err)

			if table {
				list.List(cmd, args)
			} else {
				epicList(cmd, args)
			}
		},
	}
}

// SetFlags sets flags supported by an epic list command.
func SetFlags(cmd *cobra.Command) {
	setFlags(cmd)
	hideFlags(cmd)
}

func epicList(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.Client(jira.Config{Debug: debug})

	if len(args) == 0 {
		epicExplorerView(cmd.Flags(), project, server, client)
	} else {
		singleEpicView(cmd.Flags(), args[0], project, server, client)
	}
}

func singleEpicView(flags query.FlagParser, key, project, server string, client *jira.Client) {
	err := flags.Set("type", "") // Unset issue type.
	cmdutil.ExitIfError(err)

	issues, total := func() ([]*jira.Issue, int) {
		s := cmdutil.Info("Fetching epic issues...")
		defer s.Stop()

		q, err := query.NewIssue(project, flags)
		cmdutil.ExitIfError(err)

		resp, err := client.EpicIssues(key, q.Get(), q.Params().Limit)
		cmdutil.ExitIfError(err)

		return resp.Issues, resp.Total
	}()
	if total == 0 {
		cmdutil.Errorf("No result found for given query in project \"%s\"", project)
		return
	}

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := flags.GetBool("no-headers")
	cmdutil.ExitIfError(err)

	noTruncate, err := flags.GetBool("no-truncate")
	cmdutil.ExitIfError(err)

	columns, err := flags.GetString("columns")
	cmdutil.ExitIfError(err)

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
		Display: view.DisplayFormat{
			Plain:      plain,
			NoHeaders:  noHeaders,
			NoTruncate: noTruncate,
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
	q, err := query.NewIssue(project, flags)
	cmdutil.ExitIfError(err)

	epics, total := func() ([]*jira.Issue, int) {
		s := cmdutil.Info("Fetching epics...")
		defer s.Stop()

		resp, err := client.Search(q.Get(), q.Params().Limit)
		cmdutil.ExitIfError(err)

		return resp.Issues, resp.Total
	}()
	if total == 0 {
		cmdutil.Errorf("No result found for given query in project \"%s\"", project)
		return
	}

	v := view.EpicList{
		Total:   total,
		Project: project,
		Server:  server,
		Data:    epics,
		Issues: func(key string) []*jira.Issue {
			resp, err := client.EpicIssues(key, "", q.Params().Limit)
			if err != nil {
				return []*jira.Issue{}
			}
			return resp.Issues
		},
	}

	cmdutil.ExitIfError(v.Render())
}

func setFlags(cmd *cobra.Command) {
	list.SetFlags(cmd)
	cmd.Flags().Bool("table", false, "Display epics in table view")
}

func hideFlags(cmd *cobra.Command) {
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("type"))
}
