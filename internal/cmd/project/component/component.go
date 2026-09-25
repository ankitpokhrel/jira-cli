package component

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/project/component/list"
)

const helpText = `Component manages Jira project components. See available commands below.`

// NewCmdComponent is a project component command.
func NewCmdComponent() *cobra.Command {
	cmd := cobra.Command{
		Use:     "component",
		Short:   "Component manages Jira project components",
		Long:    helpText,
		Aliases: []string{"components"},
		RunE:    component,
	}

	cmd.AddCommand(list.NewCmdList())

	return &cmd
}

func component(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
