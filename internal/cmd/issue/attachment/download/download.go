package download

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Download downloads all attachments from an issue.`
	examples = `$ jira issue attachment download ISSUE-1

# Download to a custom directory
$ jira issue attachment download ISSUE-1 --output-dir ./downloads

# Using short flags
$ jira issue attachment download ISSUE-1 -o ./my-folder`
)

// NewCmdDownload is a download command.
func NewCmdDownload() *cobra.Command {
	cmd := cobra.Command{
		Use:     "download ISSUE-KEY",
		Short:   "Download attachments from an issue",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key, eg: ISSUE-1",
		},
		Args: cobra.MinimumNArgs(1),
		Run:  download,
	}

	cmd.Flags().StringP("output-dir", "o", "", "Output directory (default: ./<ISSUE-KEY>/)")

	return &cmd
}

func download(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])

	outputDir, err := cmd.Flags().GetString("output-dir")
	cmdutil.ExitIfError(err)

	if outputDir == "" {
		outputDir = "./" + key
	}

	client := api.DefaultClient(debug)

	attachments, err := func() ([]jira.Attachment, error) {
		s := cmdutil.Info(fmt.Sprintf("Fetching attachments for %s", key))
		defer s.Stop()

		return api.ProxyGetIssueAttachments(client, key)
	}()
	cmdutil.ExitIfError(err)

	if len(attachments) == 0 {
		cmdutil.Success("No attachments found for %s", key)
		return
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		cmdutil.ExitIfError(fmt.Errorf("failed to create directory %s: %w", outputDir, err))
	}

	var (
		downloaded int
		failed     int
	)

	for _, att := range attachments {
		targetPath := filepath.Join(outputDir, att.Filename)

		err := func() error {
			s := cmdutil.Info(fmt.Sprintf("Downloading %s (%s)", att.Filename, formatSize(att.Size)))
			defer s.Stop()

			return client.DownloadAttachment(att.Content, targetPath)
		}()

		if err != nil {
			cmdutil.Fail("Failed to download %s: %v", att.Filename, err)
			failed++
		} else {
			cmdutil.Success("Downloaded %s", att.Filename)
			downloaded++
		}
	}

	fmt.Println()
	if failed > 0 {
		cmdutil.Warn("Downloaded %d of %d attachments to %s (%d failed)", downloaded, len(attachments), outputDir, failed)
	} else {
		cmdutil.Success("Downloaded %d attachments to %s", downloaded, outputDir)
	}
}

func formatSize(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
