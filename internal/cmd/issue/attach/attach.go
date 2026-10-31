package attach

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/attach/del"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/attach/download"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/attach/list"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue/attach/upload"
)

const helpText = `Attach command helps you manage issue attachments. See available commands below.`

// NewCmdAttach is an attach command.
func NewCmdAttach() *cobra.Command {
	cmd := cobra.Command{
		Use:     "attach",
		Short:   "Manage issue attachments",
		Long:    helpText,
		Aliases: []string{"attachments"},
		RunE:    attach,
	}

	cmd.AddCommand(upload.NewCmdUpload())
	cmd.AddCommand(list.NewCmdList())
	cmd.AddCommand(download.NewCmdDownload())
	cmd.AddCommand(del.NewCmdDelete())

	return &cmd
}

func attach(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
