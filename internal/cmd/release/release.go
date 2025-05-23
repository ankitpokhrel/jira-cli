package release

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/release/list"
)

const helpText = `Release manages Jira Project versions. See available commands below.`

// NewCmdRelease is a project command.
func NewCmdRelease() *cobra.Command {
	cmd := cobra.Command{
		Use:         "release",
		Short:       "Release manages Jira Project versions",
		Long:        helpText,
		Annotations: map[string]string{"cmd:main": "true"},
		Aliases:     []string{"releases"},
		RunE:        releases,
	}

	cmd.AddCommand(list.NewCmdList())

	return &cmd
}

func releases(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
