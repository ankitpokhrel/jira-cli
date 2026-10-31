package download

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Download downloads attachments of an issue.`
	examples = `$ jira issue attach download ISSUE-1 10000 10001 --dir ./downloads

# Download all attachments of an issue
$ jira issue attach download ISSUE-1 --all --dir ./downloads`
)

// NewCmdDownload is an attachment download command.
func NewCmdDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download ISSUE-KEY [ATTACHMENT-ID...]",
		Short:   "Download attachments of an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"get"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1
ATTACHMENT-ID	ID of the attachment to download`,
		},
		Run: download,
	}

	cmd.Flags().String("dir", ".", "Directory to save downloaded attachments")
	cmd.Flags().Bool("all", false, "Download all attachments of the issue")

	return cmd
}

func download(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)
	ac := downloadCmd{client: client, params: params}

	cmdutil.ExitIfError(ac.validate())
	key := cmdutil.GetJiraIssueKey(project, ac.params.key)

	err := func() error {
		ids := ac.params.ids
		if ac.params.all {
			attachments, err := client.ListAttachments(key)
			if err != nil {
				return err
			}
			if len(attachments) == 0 {
				return fmt.Errorf("no attachments found for issue %q", key)
			}
			ids = make([]string, 0, len(attachments))
			for _, a := range attachments {
				ids = append(ids, a.ID)
			}
		}
		if len(ids) == 0 {
			return fmt.Errorf("attachment id is required (or use --all): jira issue attach download ISSUE-KEY ATTACHMENT-ID [--dir DIR]")
		}

		if err := os.MkdirAll(ac.params.dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %q: %w", ac.params.dir, err)
		}
		for _, id := range ids {
			data, name, err := client.DownloadAttachment(id)
			if err != nil {
				return err
			}
			dest := filepath.Join(ac.params.dir, name)
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return fmt.Errorf("cannot write file %q: %w", dest, err)
			}
			fmt.Printf("Downloaded %s (%d bytes) -> %s\n", name, len(data), dest)
		}
		return nil
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("Attachments downloaded successfully")
}

type downloadParams struct {
	key   string
	ids   []string
	dir   string
	all   bool
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, _ string) *downloadParams {
	var key string
	var ids []string

	if len(args) >= 1 {
		key = args[0]
	}
	if len(args) >= 2 {
		ids = args[1:]
	}

	dir, err := flags.GetString("dir")
	cmdutil.ExitIfError(err)

	all, err := flags.GetBool("all")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &downloadParams{key: key, ids: ids, dir: dir, all: all, debug: debug}
}

type downloadCmd struct {
	client *jira.Client
	params *downloadParams
}

func (dc *downloadCmd) validate() error {
	if dc.params.key == "" {
		return fmt.Errorf("issue key is required: jira issue attach download ISSUE-KEY [ATTACHMENT-ID...]")
	}
	if len(dc.params.ids) == 0 && !dc.params.all {
		// Allow bare `download ISSUE-KEY` to fall back to listing first;
		// the runner errors only if the issue has no attachments.
		return nil
	}
	return nil
}
