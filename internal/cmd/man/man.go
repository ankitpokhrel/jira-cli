package man

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/man/generate"
)

// NewCmdMan is a man command.
func NewCmdMan() *cobra.Command {
	cmd := cobra.Command{
		Use:   "man",
		Short: "Help generate man(7) pages for Jira CLI.",
		Long:  "Help generate man pages for Jira CLI compatible with UNIX style man pages.",
		RunE:  man,
	}

	gm := generate.NewCmdGenerate()

	cmd.AddCommand(
		gm,
	)

	generate.SetFlags(gm)

	return &cmd
}

func man(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
