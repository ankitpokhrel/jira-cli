// Package view implements the issue view command with raw and formatted output.
package view

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	tuiView "github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

const (
	helpText = `View displays contents of an issue.

Experimental --comment-threads reads native threads on Jira Cloud using API-token
Basic auth. Server/local installations and other authentication types are unsupported.
--comments counts whole root threads with all replies in this mode; 0 hides comments.
Roots are newest-first, replies oldest-first. Default output remains flat. Read-only
thread queries use private GraphQL persisted operations, which may change; failures
are reported rather than guessing reply relationships. Uses configured credentials,
not HAR files or browser cookies. In the interactive list's in-app details, c posts a
top-level comment and r chooses a thread for a reply. Replies use a separate mutation
after explicit Post and support unrestricted Software/Business threads only.`
	examples = `$ jira issue view ISSUE-1

# Show 5 recent comments when viewing the issue
$ jira issue view ISSUE-1 --comments 5

# Show 5 complete native comment threads (experimental Jira Cloud, API-token Basic auth)
$ jira issue view ISSUE-1 --comment-threads --comments 5

# Get the raw JSON data
$ jira issue view ISSUE-1 --raw`

	flagRaw            = "raw"
	flagDebug          = "debug"
	flagComments       = "comments"
	flagCommentThreads = "comment-threads"
	flagPlain          = "plain"

	configProject = "project.key"
	configServer  = "server"

	messageFetchingData = "Fetching issue details..."
)

// NewCmdView creates the single-issue command and registers its raw, plain, comments, and thread options.
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

	cmd.Flags().Uint(flagComments, 1, "Show N comments, or N whole root threads with all replies with --comment-threads (0 hides comments)")
	cmd.Flags().Bool(flagCommentThreads, false, "Read native comment threads (experimental Jira Cloud only; API-token Basic auth)")
	cmd.Flags().Bool(flagPlain, false, "Display output in plain mode")
	cmd.Flags().Bool(flagRaw, false, "Print raw Jira API response")
	cmd.MarkFlagsMutuallyExclusive(flagRaw, flagCommentThreads)

	return &cmd
}

// view selects raw output before any rich-text rendering or thread loading;
// Cobra must already have validated that an issue argument is present.
func view(cmd *cobra.Command, args []string) {
	raw, err := cmd.Flags().GetBool(flagRaw)
	cmdutil.ExitIfError(err)

	if raw {
		viewRaw(cmd, args)
		return
	}
	viewPretty(cmd, args)
}

// viewRaw expands project-relative keys but leaves the API response body intact,
// avoiding the conversions used by formatted issue output.
func viewRaw(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool(flagDebug)
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString(configProject), args[0])

	apiResp, err := func() (string, error) {
		s := cmdutil.Info(messageFetchingData)
		defer s.Stop()

		client := api.DefaultClient(debug)
		return api.ProxyGetIssueRaw(client, key)
	}()
	cmdutil.ExitIfError(err)

	fmt.Println(apiResp)
}

// viewPretty honors explicit zero-comment requests and opts into native thread
// loading only when requested, leaving flat reads on the installation-aware proxy.
func viewPretty(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool(flagDebug)
	cmdutil.ExitIfError(err)

	var comments uint
	if cmd.Flags().Changed(flagComments) {
		comments, err = cmd.Flags().GetUint(flagComments)
		cmdutil.ExitIfError(err)
	} else {
		numComments := viper.GetUint("num_comments")
		comments = max(numComments, 1)
	}

	commentThreads, err := cmd.Flags().GetBool(flagCommentThreads)
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString(configProject), args[0])
	iss, err := func() (*jira.Issue, error) {
		s := cmdutil.Info(messageFetchingData)
		defer s.Stop()

		client := api.DefaultClient(debug)
		if commentThreads {
			return api.ProxyGetIssueWithCommentThreads(client, key, issue.NewNumCommentsFilter(comments))
		}
		return api.ProxyGetIssue(client, key, issue.NewNumCommentsFilter(comments))
	}()
	cmdutil.ExitIfError(err)

	plain, err := cmd.Flags().GetBool(flagPlain)
	cmdutil.ExitIfError(err)

	v := tuiView.Issue{
		Server:  viper.GetString(configServer),
		Data:    iss,
		Display: tuiView.DisplayFormat{Plain: plain},
		Options: tuiView.IssueOption{NumComments: comments, CommentThreads: commentThreads},
	}
	cmdutil.ExitIfError(v.Render())
}
