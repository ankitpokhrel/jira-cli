package jira

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

var boardCmd = &cobra.Command{
	Use:     "board",
	Short:   "Board lists all boards in a project",
	Long:    "Board lists all boards in a project",
	Aliases: []string{"boards"},
	Run:     board,
}

func board(*cobra.Command, []string) {
	project := viper.GetString("project")

	resp, err := jiraClient.Boards(project, jira.BoardTypeAll)
	exitIfError(err)

	v := view.NewBoard(resp.Boards)

	exitIfError(v.Render())
}

func init() {
	rootCmd.AddCommand(boardCmd)
}
