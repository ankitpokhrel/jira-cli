package upload

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
	helpText = `Upload uploads one or more files as attachments to an issue.`
	examples = `$ jira issue attach upload ISSUE-1 ./screenshot.png ./logs.txt`
)

// NewCmdUpload is an attachment upload command.
func NewCmdUpload() *cobra.Command {
	return &cobra.Command{
		Use:     "upload ISSUE-KEY FILE [FILE...]",
		Short:   "Upload files as attachments to an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"add"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1
FILE	Path to a file to upload as attachment`,
		},
		Run: upload,
	}
}

func upload(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)
	ac := uploadCmd{client: client, params: params}

	cmdutil.ExitIfError(ac.setIssueKey(project))
	cmdutil.ExitIfError(ac.validateFiles())

	err := func() error {
		s := cmdutil.Info(fmt.Sprintf("Uploading %d file(s) to issue %q", len(params.files), ac.params.key))
		defer s.Stop()

		uploaded, err := client.UploadAttachment(ac.params.key, ac.params.files)
		if err != nil {
			return err
		}
		for _, a := range uploaded {
			fmt.Printf("  %s  id=%s  %d bytes\n", a.Filename, a.ID, a.Size)
		}
		return nil
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success(fmt.Sprintf("Attached %d file(s) to issue %q", len(params.files), ac.params.key))
}

type uploadParams struct {
	key   string
	files []string
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *uploadParams {
	var key string
	var files []string

	if len(args) >= 1 {
		key = cmdutil.GetJiraIssueKey(project, args[0])
	}
	if len(args) >= 2 {
		files = args[1:]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &uploadParams{key: key, files: files, debug: debug}
}

type uploadCmd struct {
	client *jira.Client
	params *uploadParams
}

func (uc *uploadCmd) setIssueKey(project string) error {
	if uc.params.key != "" {
		return nil
	}
	return fmt.Errorf("issue key is required: jira issue attach upload ISSUE-KEY FILE [FILE...]")
}

func (uc *uploadCmd) validateFiles() error {
	if len(uc.params.files) == 0 {
		return fmt.Errorf("at least one file is required: jira issue attach upload ISSUE-KEY FILE [FILE...]")
	}
	return nil
}
