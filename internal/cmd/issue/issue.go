package issue

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/list"
)

const helpText = `Issue manage issues in a given project. See available commands below.`

// NewCmdIssue is an issue command.
func NewCmdIssue() *cobra.Command {
	cmd := cobra.Command{
		Use:     "issue",
		Short:   "Issue manage issues in a project",
		Long:    helpText,
		Aliases: []string{"issues"},
		RunE:    issue,
	}

	lc := list.NewCmdList()
	cmd.AddCommand(lc)
	list.SetFlags(lc)

	return &cmd
}

func issue(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
