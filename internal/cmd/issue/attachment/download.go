package attachment

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func newCmdAttachmentDownload() *cobra.Command {
	return &cobra.Command{
		Use:     "download ISSUE-KEY ATTACHMENT-NAME [OUTPUT-NAME]",
		Short:   "Download an attachment from an issue",
		Args:    cobra.RangeArgs(2, 3),
		Aliases: []string{"dl"},
		Run:     download,
	}
}

func download(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	key := cmdutil.GetJiraIssueKey(project, args[0])
	attachmentName := args[1]
	
	outputName := attachmentName
	if len(args) == 3 {
		outputName = args[2]
	}

	debug, _ := cmd.Flags().GetBool("debug")
	client := api.DefaultClient(debug)
	issue, err := client.GetIssueV2(key)
	cmdutil.ExitIfError(err)

	var matches []jira.Attachment
	for _, a := range issue.Fields.Attachments {
		if a.Filename == attachmentName {
			matches = append(matches, a)
		}
	}

	if len(matches) == 0 {
		cmdutil.Failed("No attachment found with name %q for issue %s", attachmentName, key)
	}

	var target jira.Attachment
	if len(matches) > 1 {
		options := make([]string, len(matches))
		for i, a := range matches {
			options[i] = fmt.Sprintf("%s (ID: %s, Size: %s, Created: %s)", a.Filename, a.ID, cmdutil.FormatBytes(int64(a.Size)), a.Created)
		}

		var selected string
		prompt := &survey.Select{
			Message: "Multiple attachments found with the same name. Please select one:",
			Options: options,
		}
		err := survey.AskOne(prompt, &selected)
		cmdutil.ExitIfError(err)

		for i, opt := range options {
			if opt == selected {
				target = matches[i]
				break
			}
		}
	} else {
		target = matches[0]
	}

	outputPath := getUniqueFilename(outputName)

	s := cmdutil.Info(fmt.Sprintf("Downloading attachment %q to %q...", target.Filename, outputPath))
	defer s.Stop()

	body, err := client.DownloadAttachment(target.Content)
	cmdutil.ExitIfError(err)
	defer body.Close()

	out, err := os.Create(outputPath)
	cmdutil.ExitIfError(err)
	defer out.Close()

	_, err = io.Copy(out, body)
	cmdutil.ExitIfError(err)

	s.Stop()
	cmdutil.Success("Attachment downloaded successfully to %q", outputPath)
}

func getUniqueFilename(name string) string {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return name
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(newName); os.IsNotExist(err) {
			return newName
		}
	}
}
