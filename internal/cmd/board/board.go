package board

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/board/list"
)

const helpText = `Board manages Jira boards in a project. See available commands below.`

// NewCmdBoard is a board command.
func NewCmdBoard() *cobra.Command {
	cmd := cobra.Command{
		Use:         "board",
		Short:       "Board manages Jira boards in a project",
		Long:        helpText,
		Aliases:     []string{"boards"},
		Annotations: map[string]string{"cmd:main": "true"},
		RunE:        board,
	}

	cmd.AddCommand(list.NewCmdList())

	return &cmd
}

func board(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
