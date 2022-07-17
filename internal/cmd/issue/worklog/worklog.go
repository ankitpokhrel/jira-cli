package worklog

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/worklog/add"
)

const helpText = `Worklog command helps you manage issue worklogs. See available commands below.`

// NewCmdWorklog is a comment command.
func NewCmdWorklog() *cobra.Command {
	cmd := cobra.Command{
		Use:     "worklog",
		Short:   "Manage issue worklog",
		Long:    helpText,
		Aliases: []string{"wlg"},
		RunE:    comment,
	}

	cmd.AddCommand(add.NewCmdWorklogAdd())

	return &cmd
}

func comment(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
