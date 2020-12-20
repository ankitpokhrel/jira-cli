package sprint

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/sprint/list"
)

const helpText = `Sprint manage sprints in a project board. See available commands below.`

// NewCmdSprint is a sprint command.
func NewCmdSprint() *cobra.Command {
	cmd := cobra.Command{
		Use:     "sprint",
		Short:   "Sprint manage sprints in a project board",
		Long:    helpText,
		Aliases: []string{"sprints"},
		RunE:    sprint,
	}

	lc := list.NewCmdList()
	cmd.AddCommand(lc)
	list.SetFlags(lc)

	return &cmd
}

func sprint(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
