package release

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/release/create"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/release/edit"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/release/list"
)

const helpText = `Release manages Jira Project versions. See available commands below.`

// NewCmdRelease is a release command.
func NewCmdRelease() *cobra.Command {
	cmd := cobra.Command{
		Use:         "release",
		Short:       "Release manages Jira Project versions",
		Long:        helpText,
		Annotations: map[string]string{"cmd:main": "true"},
		Aliases:     []string{"releases"},
		RunE:        releases,
	}

	listCmd := list.NewCmdList()
	createCmd := create.NewCmdCreate()
	editCmd := edit.NewCmdEdit()

	cmd.AddCommand(listCmd, createCmd, editCmd)

	create.SetFlags(createCmd)
	// edit command has its own setFlags call in NewCmdEdit

	return &cmd
}

func releases(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
