package list

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/list"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	numSprints = 50 // This is the maximum result returned by Jira API at once.
	helpText   = `
Sprints are displayed in an explorer view by default. You can use --list
and --plain flags to display output in different modes.`

	examples = `$ jira sprint list

# Display sprints or sprint issues in an interactive list
$ jira sprint list --table
$ jira sprint list <SPRINT_ID> --table

# Display sprints or sprint issues in a plain table view
$ jira sprint list --table --plain
$ jira sprint list <SPRINT_ID> --plain

# Display sprints or sprint issues in a plain table view without headers
$ jira sprint list --table --plain --no-headers
$ jira sprint list <SPRINT_ID> --no-headers

# Display some columns of sprint or sprint issues in a plain table view
$ jira sprint list --table --plain --columns name,start,end
$ jira sprint list <SPRINT_ID> --plain --columns type,key,summary

# Display sprint issues in a plain table view and show all fields
$ jira sprint list <SPRINT_ID> --plain --no-truncate`
)

// NewCmdList is a sprint list command.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list [SPRINT_ID]",
		Short:   fmt.Sprintf("Sprint lists top %d sprints in a board", numSprints),
		Long:    fmt.Sprintf("Sprint lists top %d sprints in a board\n", numSprints) + helpText,
		Example: examples,
		Args:    cobra.MaximumNArgs(1),
		Aliases: []string{"lists", "ls"},
		Annotations: map[string]string{
			"help:args": "[SPRINT_ID]\tID of the sprint",
		},
		Run: sprintList,
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
	project := viper.GetString("project.key")
	boardID := viper.GetInt("board.id")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.DefaultClient(debug)

	sprintQuery, err := query.NewSprint(cmd.Flags())
	cmdutil.ExitIfError(err)

	if len(args) == 0 {
		sprintExplorerView(sprintQuery, cmd.Flags(), boardID, project, server, client)
	} else {
		sprintID, err := strconv.Atoi(args[0])
		cmdutil.ExitIfError(err)

		singleSprintView(sprintQuery, cmd.Flags(), boardID, sprintID, project, server, client, nil)
	}
}

func singleSprintView(sprintQuery *query.Sprint, flags query.FlagParser, boardID, sprintID int, project, server string, client *jira.Client, sprint *jira.Sprint) {
	issues, total, err := func() ([]*jira.Issue, int, error) {
		s := cmdutil.Info("Fetching sprint issues...")
		defer s.Stop()

		q, err := query.NewIssue(project, flags)
		if err != nil {
			return nil, 0, err
		}
		if sprintQuery.Params().ShowAllIssues {
			q.Params().JQL = "project IS NOT EMPTY"
		}
		resp, err := client.SprintIssues(sprintID, q.Get(), q.Params().From, q.Params().Limit)
		if err != nil {
			return nil, 0, err
		}
		return resp.Issues, resp.Total, nil
	}()
	cmdutil.ExitIfError(err)

	if total == 0 {
		fmt.Println()
		cmdutil.Failed("No result found for given query in project %q", project)
		return
	}

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := flags.GetBool("no-headers")
	cmdutil.ExitIfError(err)

	noTruncate, err := flags.GetBool("no-truncate")
	cmdutil.ExitIfError(err)

	fixedColumns, err := flags.GetUint("fixed-columns")
	cmdutil.ExitIfError(err)

	columns, err := flags.GetString("columns")
	cmdutil.ExitIfError(err)

	var ft string
	if sprint != nil {
		if sprint.Status == jira.SprintStateFuture {
			ft = fmt.Sprintf(
				"Showing %d of %d results for project %q in sprint #%d ➤ %s (Future Sprint)",
				len(issues), total, project, sprint.ID, sprint.Name,
			)
		} else {
			ft = fmt.Sprintf(
				"Showing %d of %d results for project %q in sprint #%d ➤ %s (%s - %s)",
				len(issues), total, project, sprint.ID, sprint.Name,
				cmdutil.FormatDateTimeHuman(sprint.StartDate, time.RFC3339),
				cmdutil.FormatDateTimeHuman(sprint.EndDate, time.RFC3339),
			)
		}
	} else {
		ft = fmt.Sprintf(
			"Showing %d of %d results for project %q in sprint #%d",
			len(issues), total, project, sprintID,
		)
	}

	v := view.IssueList{
		Project:    project,
		Server:     server,
		Total:      total,
		Data:       issues,
		FooterText: ft,
		Refresh: func() {
			singleSprintView(sprintQuery, flags, boardID, sprintID, project, server, client, nil)
		},
		Display: view.DisplayFormat{
			Plain:        plain,
			NoHeaders:    noHeaders,
			NoTruncate:   noTruncate,
			FixedColumns: fixedColumns,
			Columns: func() []string {
				if columns != "" {
					return strings.Split(columns, ",")
				}
				return []string{}
			}(),
			TableStyle: cmdutil.GetTUIStyleConfig(),
		},
	}

	cmdutil.ExitIfError(v.Render())
}

func sprintExplorerView(sprintQuery *query.Sprint, flags query.FlagParser, boardID int, project, server string, client *jira.Client) {
	sprints := func() []*jira.Sprint {
		s := cmdutil.Info("Fetching sprints...")
		defer s.Stop()

		return client.SprintsInBoards([]int{boardID}, sprintQuery.Get(), numSprints)
	}()
	if len(sprints) == 0 {
		fmt.Println()
		cmdutil.Failed("No result found for given query in project %q", project)
		return
	}

	if sprintQuery.Params().Current || sprintQuery.Params().Prev || sprintQuery.Params().Next {
		sprint := sprints[0]
		if sprintQuery.Params().Next {
			sprint = sprints[len(sprints)-1]
		}
		singleSprintView(sprintQuery, flags, boardID, sprint.ID, project, server, client, sprint)
		return
	}

	plain, err := flags.GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := flags.GetBool("no-headers")
	cmdutil.ExitIfError(err)

	fixedColumns, err := flags.GetUint("fixed-columns")
	cmdutil.ExitIfError(err)

	columns, err := flags.GetString("columns")
	cmdutil.ExitIfError(err)

	v := view.SprintList{
		Project: project,
		Board:   viper.GetString("board.name"),
		Server:  server,
		Data:    sprints,
		Issues: func(boardID, sprintID int) []*jira.Issue {
			iq, err := getIssueQuery(project, flags, sprintQuery.Params().ShowAllIssues)
			if err != nil {
				return []*jira.Issue{}
			}
			resp, err := client.SprintIssues(sprintID, iq, sprintQuery.Params().From, sprintQuery.Params().Limit)
			if err != nil {
				return []*jira.Issue{}
			}
			return resp.Issues
		},
		Display: view.DisplayFormat{
			Plain:        plain,
			NoHeaders:    noHeaders,
			FixedColumns: fixedColumns,
			Columns: func() []string {
				if columns != "" {
					return strings.Split(columns, ",")
				}
				return []string{}
			}(),
			TableStyle: cmdutil.GetTUIStyleConfig(),
		},
	}

	table, err := flags.GetBool("table")
	cmdutil.ExitIfError(err)

	if table || tui.IsDumbTerminal() || tui.IsNotTTY() {
		cmdutil.ExitIfError(v.RenderInTable())
	} else {
		cmdutil.ExitIfError(v.Render())
	}
}

func getIssueQuery(project string, flags query.FlagParser, showAll bool) (string, error) {
	q, err := query.NewIssue(project, flags)
	if err != nil {
		return "", err
	}
	if showAll {
		q.Params().JQL = "project IS NOT EMPTY"
	}
	return q.Get(), nil
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().String("state", "", "Filter sprint by its state (comma separated).\n"+
		"Valid values are future, active and closed.\n"+
		`Defaults to "active,closed"`)
	cmd.Flags().Bool("show-all-issues", false, "Show sprint issues from all projects")
	cmd.Flags().Bool("table", false, "Display sprints in a table view")
	cmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
		fmt.Sprintf("Accepts (for sprint list): %s", strings.Join(view.ValidSprintColumns(), ", "))+
		fmt.Sprintf("\nAccepts (for sprint issues): %s", strings.Join(view.ValidIssueColumns(), ", ")))
	cmd.Flags().Uint("fixed-columns", 1, "Number of fixed columns in the interactive mode")
	cmd.Flags().Bool("current", false, "List issues in current active sprint")
	cmd.Flags().Bool("prev", false, "List issues in previous sprint")
	cmd.Flags().Bool("next", false, "List issues in next planned sprint")
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
