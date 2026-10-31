package del

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Delete deletes one or more attachments from an issue.`
	examples = `$ jira issue attach delete ISSUE-1 10000 10001`
)

// NewCmdDelete is an attachment delete command.
func NewCmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:     "delete ISSUE-KEY ATTACHMENT-ID [ATTACHMENT-ID...]",
		Short:   "Delete attachments from an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"remove", "rm", "del"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1
ATTACHMENT-ID	ID of the attachment to delete`,
		},
		Run: del,
	}
}

func del(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)
	ac := deleteCmd{client: client, params: params}

	cmdutil.ExitIfError(ac.validate())

	err := func() error {
		s := cmdutil.Info(fmt.Sprintf("Deleting %d attachment(s)", len(ac.params.ids)))
		defer s.Stop()

		for _, id := range ac.params.ids {
			if err := client.DeleteAttachment(id); err != nil {
				return err
			}
			fmt.Printf("Deleted attachment %s\n", id)
		}
		return nil
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("Attachments deleted successfully")
}

type deleteParams struct {
	key   string
	ids   []string
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, _ string) *deleteParams {
	var key string
	var ids []string

	if len(args) >= 1 {
		key = args[0]
	}
	if len(args) >= 2 {
		ids = args[1:]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &deleteParams{key: key, ids: ids, debug: debug}
}

type deleteCmd struct {
	client *jira.Client
	params *deleteParams
}

func (dc *deleteCmd) validate() error {
	if dc.params.key == "" {
		return fmt.Errorf("issue key is required: jira issue attach delete ISSUE-KEY ATTACHMENT-ID [ATTACHMENT-ID...]")
	}
	if len(dc.params.ids) == 0 {
		return fmt.Errorf("at least one attachment id is required: jira issue attach delete ISSUE-KEY ATTACHMENT-ID [ATTACHMENT-ID...]")
	}
	return nil
}
