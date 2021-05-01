package comment

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/comment/add"
)

const helpText = `Comment command helps you manage issue comments. See available commands below.`

// NewCmdComment is a comment command.
func NewCmdComment() *cobra.Command {
	cmd := cobra.Command{
		Use:     "comment",
		Short:   "Manage issue comments",
		Long:    helpText,
		Aliases: []string{"comments"},
		RunE:    comment,
	}

	cmd.AddCommand(add.NewCmdCommentAdd())

	return &cmd
}

func comment(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
