package list

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `List lists attachments of an issue.`
	examples = `$ jira issue attach list ISSUE-1`
)

// NewCmdList is an attachment list command.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list ISSUE-KEY",
		Short:   "List attachments of an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"ls"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1`,
		},
		Run: listAttachments,
	}
}

func listAttachments(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)

	var key string
	if params.key != "" {
		key = params.key
	} else {
		cmdutil.ExitIfError(fmt.Errorf("issue key is required: jira issue attach list ISSUE-KEY"))
	}
	key = cmdutil.GetJiraIssueKey(project, key)

	var attachments []jira.Attachment
	err := func() error {
		s := cmdutil.Info(fmt.Sprintf("Fetching attachments for issue %q", key))
		defer s.Stop()

		fetched, err := client.ListAttachments(key)
		if err != nil {
			return err
		}
		attachments = fetched
		return nil
	}()
	cmdutil.ExitIfError(err)

	if len(attachments) == 0 {
		fmt.Printf("No attachments found for issue %q\n", key)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tFILENAME\tSIZE\tAUTHOR\tCREATED\n")
	for _, a := range attachments {
		author := a.Author.DisplayName
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", a.ID, a.Filename, a.Size, author, a.Created)
	}
	_ = w.Flush()
}

type listParams struct {
	key   string
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, _ string) *listParams {
	var key string
	if len(args) >= 1 {
		key = args[0]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &listParams{key: key, debug: debug}
}
