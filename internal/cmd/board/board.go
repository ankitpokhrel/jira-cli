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
	project := viper.GetString("project.key")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	boards, total, err := func() ([]*jira.Board, int, error) {
		s := cmdutil.Info(fmt.Sprintf("Fetching boards in project %s...", project))
		defer s.Stop()

		resp, err := api.Client(jira.Config{Debug: debug}).Boards(project, jira.BoardTypeAll)
		if err != nil {
			return nil, 0, err
		}
		return resp.Boards, resp.Total, nil
	}()
	cmdutil.ExitIfError(err)

	if total == 0 {
		cmdutil.Failed("No boards found in project \"%s\"", project)
		return
	}

	v := view.NewBoard(boards)

	cmdutil.ExitIfError(v.Render())
}
