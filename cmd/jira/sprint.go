package jira

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const numSprints = 25

var sprintCmd = &cobra.Command{
	Use:   "sprint [SPRINT ID]",
	Short: fmt.Sprintf("Sprint lists top %d sprints in a board", numSprints),
	Long: fmt.Sprintf("Sprint lists top %d sprints in a board.\n", numSprints) +
		`
Sprints are displayed in an explorer view by default. You can use --list
and --plain flags to display output in different modes.

EG:
	# Display sprints or sprint issues in an interactive list
	jira sprint --list
	jira sprint <SPRINT_ID> --list

	# Display sprints or sprint issues in a plain table view
	jira sprint --list --plain
	jira sprint <SPRINT_ID> --list --plain

	# Display sprints or sprint issues in a plain table view without headers
	jira sprint --list --plain --no-headers
	jira sprint <SPRINT_ID> --plain --no-headers

	# Display some columns of sprint or sprint issues in a plain table view
	jira sprint --list --plain --columns name,start,end
	jira sprint <SPRINT_ID> --plain --columns type,key,summary

	# Display sprint issues in a plain table view and show all fields
	jira sprint <SPRINT_ID> --list --plain --no-truncate
`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"sprints"},
	Run:     sprint,
}

func sprint(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")
	boardID := viper.GetInt("board.id")

	if len(args) == 0 {
		sprintExplorerView(cmd.Flags(), boardID, project, server)
	} else {
		sprintID, err := strconv.Atoi(args[0])
		exitIfError(err)

		singleSprintView(cmd.Flags(), boardID, sprintID, project, server)
	}
}

func singleSprintView(flags query.FlagParser, boardID, sprintID int, project, server string) {
	plain, err := flags.GetBool("plain")
	exitIfError(err)

	issues, total := func() ([]*jira.Issue, int) {
		if !plain {
			s := info("Fetching sprint issues...")
			defer s.Stop()
		}

		q, err := query.NewIssue(project, flags)
		exitIfError(err)

		resp, err := jiraClient.SprintIssues(boardID, sprintID, q.Get())
		exitIfError(err)

		return resp.Issues, resp.Total
	}()

	if total == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)
		return
	}

	noHeaders, err := flags.GetBool("no-headers")
	exitIfError(err)

	noTruncate, err := flags.GetBool("no-truncate")
	exitIfError(err)

	columns, err := flags.GetString("columns")
	exitIfError(err)

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

	exitIfError(v.Render())
}

func sprintExplorerView(flags query.FlagParser, boardID int, project, server string) {
	plain, err := flags.GetBool("plain")
	exitIfError(err)

	sprints := func() []*jira.Sprint {
		if !plain {
			s := info("Fetching sprints...")
			defer s.Stop()
		}

		resp, err := jiraClient.Boards(project, jira.BoardTypeScrum)
		exitIfError(err)

		boardIDs := make([]int, 0, resp.Total)
		for _, board := range resp.Boards {
			boardIDs = append(boardIDs, board.ID)
		}

		q, err := query.NewSprint(flags)
		exitIfError(err)

		return jiraClient.SprintsInBoards([]int{boardID}, q.Get(), numSprints)
	}()

	if len(sprints) == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)
		return
	}

	noHeaders, err := flags.GetBool("no-headers")
	exitIfError(err)

	columns, err := flags.GetString("columns")
	exitIfError(err)

	v := view.SprintList{
		Project: project,
		Board:   viper.GetString("board.name"),
		Server:  server,
		Data:    sprints,
		Issues: func(boardID, sprintID int) []*jira.Issue {
			resp, err := jiraClient.SprintIssues(boardID, sprintID, "")
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

	list, err := flags.GetBool("list")
	exitIfError(err)

	if list {
		exitIfError(v.RenderInTable())
	} else {
		exitIfError(v.Render())
	}
}

func init() {
	rootCmd.AddCommand(sprintCmd)

	sprintCmd.Flags().String("state", "", "Filter sprint by its state (comma separated).\n"+
		"Valid values are future, active and closed.\n"+
		`Defaults to "active,closed"`)
	sprintCmd.Flags().Bool("list", false, "Display sprints in list view")

	injectIssueFlags(sprintCmd)

	sprintCmd.Flags().String("columns", "", "Comma separated list of columns to display in the plain mode.\n"+
		fmt.Sprintf("Accepts: %s", strings.Join(view.ValidSprintColumns(), ", ")))

	exitIfError(sprintCmd.Flags().MarkHidden("history"))
	exitIfError(sprintCmd.Flags().MarkHidden("watching"))
	exitIfError(sprintCmd.Flags().MarkHidden("type"))
	exitIfError(sprintCmd.Flags().MarkHidden("resolution"))
	exitIfError(sprintCmd.Flags().MarkHidden("status"))
	exitIfError(sprintCmd.Flags().MarkHidden("priority"))
	exitIfError(sprintCmd.Flags().MarkHidden("reporter"))
	exitIfError(sprintCmd.Flags().MarkHidden("assignee"))
	exitIfError(sprintCmd.Flags().MarkHidden("created"))
	exitIfError(sprintCmd.Flags().MarkHidden("updated"))
	exitIfError(sprintCmd.Flags().MarkHidden("created-after"))
	exitIfError(sprintCmd.Flags().MarkHidden("updated-after"))
	exitIfError(sprintCmd.Flags().MarkHidden("created-before"))
	exitIfError(sprintCmd.Flags().MarkHidden("updated-before"))
	exitIfError(sprintCmd.Flags().MarkHidden("label"))
	exitIfError(sprintCmd.Flags().MarkHidden("reverse"))
}
