package jira

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const numSprints = 25

var sprintCmd = &cobra.Command{
	Use:     "sprint",
	Short:   fmt.Sprintf("Sprint lists top %d sprints in a board", numSprints),
	Long:    fmt.Sprintf("Sprint lists to %d sprints for a board in a project", numSprints),
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
	q, err := query.NewIssue(project, flags)
	exitIfError(err)

	resp, err := jiraClient.SprintIssues(boardID, sprintID, q.Get())
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

func sprintExplorerView(flags query.FlagParser, boardID int, project, server string) {
	resp, err := jiraClient.Boards(project, jira.BoardTypeScrum)
	exitIfError(err)

	boardIDs := make([]int, 0, resp.Total)
	for _, board := range resp.Boards {
		boardIDs = append(boardIDs, board.ID)
	}

	q, err := query.NewSprint(flags)
	exitIfError(err)

	sprints := jiraClient.SprintsInBoards([]int{boardID}, q.Get(), numSprints)

	if len(sprints) == 0 {
		fmt.Printf("No result found for given query in project \"%s\"\n", project)

		return
	}

	list, err := flags.GetBool("list")
	exitIfError(err)

	v := view.SprintList{
		Project: project,
		Board:   viper.GetString("board.name"),
		Server:  server,
		Data:    sprints,
		Issues: func(boardID, sprintID int) []jira.Issue {
			resp, err := jiraClient.SprintIssues(boardID, sprintID, "")
			if err != nil {
				return []jira.Issue{}
			}

			return resp.Issues
		},
	}

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

	sprintCmd.Flags().Bool("history", false, "Epics you accessed recently")
	sprintCmd.Flags().BoolP("watching", "w", false, "Epics you are watching")
	sprintCmd.Flags().StringP("type", "t", "", "Filter epics by type")
	sprintCmd.Flags().StringP("resolution", "r", "", "Filter epics by resolution type")
	sprintCmd.Flags().StringP("status", "s", "", "Filter epics by status")
	sprintCmd.Flags().StringP("priority", "y", "", "Filter epics by priority")
	sprintCmd.Flags().StringP("reporter", "e", "", "Filter epics by reporter (email or display name)")
	sprintCmd.Flags().StringP("assignee", "a", "", "Filter epics by assignee (email or display name)")
	sprintCmd.Flags().String("created", "", "Filter issues by created date\n"+
		"Accepts: today, week, month, year")
	sprintCmd.Flags().String("updated", "", "Filter issues by updated date\n"+
		"Accepts: today, week, month, year")
	sprintCmd.Flags().StringArrayP("label", "l", []string{}, "Filter epics by label")
	sprintCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")

	exitIfError(sprintCmd.Flags().MarkHidden("history"))
	exitIfError(sprintCmd.Flags().MarkHidden("watching"))
	exitIfError(sprintCmd.Flags().MarkHidden("type"))
	exitIfError(sprintCmd.Flags().MarkHidden("resolution"))
	exitIfError(sprintCmd.Flags().MarkHidden("status"))
	exitIfError(sprintCmd.Flags().MarkHidden("priority"))
	exitIfError(sprintCmd.Flags().MarkHidden("reporter"))
	exitIfError(sprintCmd.Flags().MarkHidden("assignee"))
	exitIfError(sprintCmd.Flags().MarkHidden("label"))
	exitIfError(sprintCmd.Flags().MarkHidden("reverse"))
}
