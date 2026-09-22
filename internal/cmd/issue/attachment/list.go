package attachment

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/view"
)

func newCmdAttachmentList() *cobra.Command {
	return &cobra.Command{
		Use:     "list ISSUE-KEY",
		Short:   "List attachments of an issue",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"ls"},
		Run:     list,
	}
}

func list(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	key := cmdutil.GetJiraIssueKey(project, args[0])
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.DefaultClient(debug)
	issue, err := client.GetIssueV2(key)
	cmdutil.ExitIfError(err)

	if len(issue.Fields.Attachments) == 0 {
		fmt.Printf("No attachments found for issue %s\n", key)
		return
	}

	v := view.NewAttachment(issue.Fields.Attachments)
	cmdutil.ExitIfError(v.Render())
}
