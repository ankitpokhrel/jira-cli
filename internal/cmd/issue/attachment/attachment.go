package attachment

import (
	"github.com/spf13/cobra"
)

const helpText = `Attachment command helps you manage issue attachments. See available commands below.`

// NewCmdAttachment is an attachment command.
func NewCmdAttachment() *cobra.Command {
	cmd := cobra.Command{
		Use:     "attachment",
		Short:   "Manage issue attachments",
		Long:    helpText,
		Aliases: []string{"attachments"},
		RunE:    attachment,
	}

	cmd.AddCommand(
		newCmdAttachmentList(),
		newCmdAttachmentDownload(),
	)

	return &cmd
}

func attachment(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
