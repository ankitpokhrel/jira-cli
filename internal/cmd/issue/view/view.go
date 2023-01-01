package view

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	tuiView "github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

const (
	helpText = `View displays contents of an issue.`
	examples = `$ jira issue view ISSUE-1

# Show 5 recent comments when viewing the issue
$ jira issue view ISSUE-1 --comments 5`
)

// NewCmdView is a view command.
func NewCmdView() *cobra.Command {
	cmd := cobra.Command{
		Use:     "view ISSUE-KEY",
		Short:   "View displays contents of an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"show"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key, eg: ISSUE-1",
		},
		Args: cobra.MinimumNArgs(1),
		Run:  view,
	}

	cmd.Flags().Uint("comments", 1, "Show N comments")
	cmd.Flags().Bool("plain", false, "Display output in plain mode")

	return &cmd
}

func view(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	comments, err := cmd.Flags().GetUint("comments")
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	iss, err := func() (*jira.Issue, error) {
		s := cmdutil.Info("Fetching issue details...")
		defer s.Stop()

		client := api.DefaultClient(debug)
		return api.ProxyGetIssue(client, key, issue.NewNumCommentsFilter(comments))
	}()
	cmdutil.ExitIfError(err)

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	v := tuiView.Issue{
		Server:  viper.GetString("server"),
		Data:    iss,
		Display: tuiView.DisplayFormat{Plain: plain},
		Options: tuiView.IssueOption{NumComments: comments},
	}
	cmdutil.ExitIfError(v.Render())
}
