package board

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// NewCmdBoard is a board command.
func NewCmdBoard() *cobra.Command {
	return &cobra.Command{
		Use:         "board",
		Short:       "Board lists all boards in a project",
		Long:        "Board lists all boards in a project.",
		Aliases:     []string{"boards"},
		Annotations: map[string]string{"cmd:main": "true"},
		Run:         board,
	}
}

func board(cmd *cobra.Command, _ []string) {
	project := viper.GetString("project")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	boards, total := func() ([]*jira.Board, int) {
		s := cmdutil.Info(fmt.Sprintf("Fetching boards in project %s...", project))
		defer s.Stop()

		resp, err := api.Client(jira.Config{Debug: debug}).Boards(project, jira.BoardTypeAll)
		cmdutil.ExitIfError(err)

		return resp.Boards, resp.Total
	}()
	if total == 0 {
		cmdutil.PrintErrF("No boards found in project \"%s\"", project)
		return
	}

	v := view.NewBoard(boards)

	cmdutil.ExitIfError(v.Render())
}
