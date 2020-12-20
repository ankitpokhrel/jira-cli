package list

import (
	"fmt"
	"strconv"
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
	numSprints = 25
	helpText   = `
Sprints are displayed in an explorer view by default. You can use --list
and --plain flags to display output in different modes.

EG:
	# Display epics in an explorer view
	jira sprint list

	# Display sprints or sprint issues in an interactive list
	jira sprint list --table
	jira sprint list <SPRINT_ID> --table

	# Display sprints or sprint issues in a plain table view
	jira sprint list --table --plain
	jira sprint list <SPRINT_ID> --table --plain

	# Display sprints or sprint issues in a plain table view without headers
	jira sprint list --table --plain --no-headers
	jira sprint list <SPRINT_ID> --plain --no-headers

	# Display some columns of sprint or sprint issues in a plain table view
	jira sprint list --table --plain --columns name,start,end
	jira sprint list <SPRINT_ID> --plain --columns type,key,summary

	# Display sprint issues in a plain table view and show all fields
	jira sprint list <SPRINT_ID> --table --plain --no-truncate
`
)

// NewCmdList is a sprint list command.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list [SPRINT ID]",
		Short:   fmt.Sprintf("Sprint lists top %d sprints in a board", numSprints),
		Long:    fmt.Sprintf("Sprint lists top %d sprints in a board\n", numSprints) + helpText,
		Args:    cobra.MaximumNArgs(1),
		Aliases: []string{"lists"},
		Run:     sprintList,
	}
}

// SetFlags sets flags supported by a sprint list command.
func SetFlags(cmd *cobra.Command) {
	list.SetFlags(cmd)
	setFlags(cmd)
	hideFlags(cmd)
}

func sprintList(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")
	boardID := viper.GetInt("board.id")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.Client(jira.Config{Debug: debug})

	if len(args) == 0 {
		sprintExplorerView(cmd.Flags(), boardID, project, server, client)
	} else {
		sprintID, err := strconv.Atoi(args[0])
		cmdutil.ExitIfError(err)

		singleSprintView(cmd.Flags(), boardID, sprintID, project, server, client)
	}
}

func singleSprintView(flags query.FlagParser, boardID, sprintID int, project, server string, client *jira.Client) {
	issues, total := func() ([]*jira.Issue, int) {
		s := cmdutil.Info("Fetching sprint issues...")
		defer s.Stop()

		q, err := query.NewIssue(project, flags)
		cmdutil.ExitIfError(err)

		resp, err := client.SprintIssues(boardID, sprintID, q.Get())
		cmdutil.ExitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		cmdutil.PrintErrF("No result found for given query in project \"%s\"", project)
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

func sprintExplorerView(flags query.FlagParser, boardID int, project, server string, client *jira.Client) {
	sprints := func() []*jira.Sprint {
		s := cmdutil.Info("Fetching sprints...")
		defer s.Stop()

		resp, err := client.Boards(project, jira.BoardTypeScrum)
		cmdutil.ExitIfError(err)

		boardIDs := make([]int, 0, resp.Total)
		for _, board := range resp.Boards {
			boardIDs = append(boardIDs, board.ID)
		}

		q, err := query.NewSprint(flags)
		cmdutil.ExitIfError(err)

		return client.SprintsInBoards([]int{boardID}, q.Get(), numSprints)
	}()

	if len(sprints) == 0 {
		cmdutil.PrintErrF("No result found for given query in project \"%s\"", project)
		return
	}

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := flags.GetBool("no-headers")
	cmdutil.ExitIfError(err)

	columns, err := flags.GetString("columns")
	cmdutil.ExitIfError(err)

	v := view.SprintList{
		Project: project,
		Board:   viper.GetString("board.name"),
		Server:  server,
		Data:    sprints,
		Issues: func(boardID, sprintID int) []*jira.Issue {
			resp, err := client.SprintIssues(boardID, sprintID, "")
			if err != nil {
				return []*jira.Issue{}
			}
			return resp.Issues
		},
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

	table, err := flags.GetBool("table")
	cmdutil.ExitIfError(err)

	if table {
		cmdutil.ExitIfError(v.RenderInTable())
	} else {
		cmdutil.ExitIfError(v.Render())
	}
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().String("state", "", "Filter sprint by its state (comma separated).\n"+
		"Valid values are future, active and closed.\n"+
		`Defaults to "active,closed"`)
	cmd.Flags().Bool("table", false, "Display sprints in table view")
	cmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
		fmt.Sprintf("Accepts: %s", strings.Join(view.ValidSprintColumns(), ", ")))
}

func hideFlags(cmd *cobra.Command) {
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("history"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("watching"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("type"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("resolution"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("status"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("priority"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("reporter"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("assignee"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("created"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("updated"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("created-after"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("updated-after"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("created-before"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("updated-before"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("label"))
	cmdutil.ExitIfError(cmd.Flags().MarkHidden("reverse"))
}
