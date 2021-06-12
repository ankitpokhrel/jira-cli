package view

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	tuiView "github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `View displays contents of an issue.`
	examples = `$ jira issue view ISSUE-1`
)

// NewCmdView is a view command.
func NewCmdView() *cobra.Command {
	cmd := cobra.Command{
		Use:     "view ISSUE_KEY",
		Short:   "View displays contents of an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"show"},
		Annotations: map[string]string{
			"help:args": "ISSUE_KEY\tIssue key, eg: ISSUE-1",
		},
		Args: cobra.MinimumNArgs(1),
		Run:  view,
	}

	cmd.Flags().Bool("plain", false, "Display output in plain mode")

	return &cmd
}

func view(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString("project"), args[0])
	issue := func() *jira.Issue {
		s := cmdutil.Info("Fetching issue details...")
		defer s.Stop()

		issue, err := api.Client(jira.Config{Debug: debug}).GetIssue(key)
		cmdutil.ExitIfError(err)

		return issue
	}()

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	v := tuiView.Issue{
		Data:    issue,
		Display: tuiView.DisplayFormat{Plain: plain},
	}
	cmdutil.ExitIfError(v.Render())
}
