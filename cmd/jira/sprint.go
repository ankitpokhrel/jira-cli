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
	Use:     "sprint [SPRINT ID]",
	Short:   fmt.Sprintf("Sprint lists top %d sprints in a board", numSprints),
	Long:    fmt.Sprintf("Sprint lists top %d sprints for a board in a project.", numSprints),
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
	issues, total := func() ([]*jira.Issue, int) {
		s := info("Fetching sprint issues...")
		defer s.Stop()

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

	v := view.IssueList{
		Project: project,
		Server:  server,
		Total:   total,
		Data:    issues,
	}

	exitIfError(v.Render())
}

func sprintExplorerView(flags query.FlagParser, boardID int, project, server string) {
	sprints := func() []*jira.Sprint {
		s := info("Fetching sprints...")
		defer s.Stop()

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

	sprintCmd.Flags().Bool("history", false, "Issues you accessed recently")
	sprintCmd.Flags().BoolP("watching", "w", false, "Issues you are watching")
	sprintCmd.Flags().StringP("type", "t", "", "Filter issues by type")
	sprintCmd.Flags().StringP("resolution", "r", "", "Filter issues by resolution type")
	sprintCmd.Flags().StringP("status", "s", "", "Filter issues by status")
	sprintCmd.Flags().StringP("priority", "y", "", "Filter issues by priority")
	sprintCmd.Flags().StringP("reporter", "e", "", "Filter issues by reporter (email or display name)")
	sprintCmd.Flags().StringP("assignee", "a", "", "Filter issues by assignee (email or display name)")
	sprintCmd.Flags().String("created", "", "Filter issues by created date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Created filter will have precedence over created-after and created-before filter")
	sprintCmd.Flags().String("updated", "", "Filter issues by updated date\n"+
		"Accepts: today, week, month, year, or a date in yyyy-mm-dd and yyyy/mm/dd format,\n"+
		"or a period format using w = weeks, d = days, h = hours, m = minutes. eg: -10d\n"+
		"Updated filter will have precedence over updated-after and updated-before filter")
	sprintCmd.Flags().String("created-after", "", "Filter by issues created after certain date")
	sprintCmd.Flags().String("updated-after", "", "Filter by issues updated after certain date")
	sprintCmd.Flags().String("created-before", "", "Filter by issues created before certain date")
	sprintCmd.Flags().String("updated-before", "", "Filter by issues updated before certain date")
	sprintCmd.Flags().StringArrayP("label", "l", []string{}, "Filter issues by label")
	sprintCmd.Flags().Bool("reverse", false, "Reverse the display order (default is DESC)")

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
