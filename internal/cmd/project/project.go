package project

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/project/list"
)

const helpText = `Project manages Jira projects. See available commands below.`

// NewCmdProject is a project command.
func NewCmdProject() *cobra.Command {
	cmd := cobra.Command{
		Use:         "project",
		Short:       "Project manages Jira projects",
		Long:        helpText,
		Aliases:     []string{"projects"},
		Annotations: map[string]string{"cmd:main": "true"},
		RunE:        projects,
	}

	cmd.AddCommand(list.NewCmdList())

	return &cmd
}

func projects(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
