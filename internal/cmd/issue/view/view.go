package view

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdcommon"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	tuiView "github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

const (
	helpText = `View displays contents of an issue.`
	examples = `$ jira issue view ISSUE-1

# Show 5 recent comments when viewing the issue
$ jira issue view ISSUE-1 --comments 5

# Get the raw JSON data
$ jira issue view ISSUE-1 --raw

# Get JSON output with human-readable custom field names
$ jira issue view ISSUE-1 --json`

	flagRaw      = "raw"
	flagJSON     = "json"
	flagDebug    = "debug"
	flagComments = "comments"
	flagPlain    = "plain"

	configProject = "project.key"
	configServer  = "server"

	messageFetchingData = "Fetching issue details..."
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

	cmd.Flags().Uint(flagComments, 1, "Show N comments")
	cmd.Flags().Bool(flagPlain, false, "Display output in plain mode")
	cmd.Flags().Bool(flagRaw, false, "Print raw Jira API response")
	cmd.Flags().Bool(flagJSON, false, "Print JSON output with human-readable custom field names")

	return &cmd
}

func view(cmd *cobra.Command, args []string) {
	jsonOutput, err := cmd.Flags().GetBool(flagJSON)
	cmdutil.ExitIfError(err)

	raw, err := cmd.Flags().GetBool(flagRaw)
	cmdutil.ExitIfError(err)

	if jsonOutput {
		viewJSON(cmd, args)
		return
	}
	if raw {
		viewRaw(cmd, args)
		return
	}
	viewPretty(cmd, args)
}

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

func viewJSON(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool(flagDebug)
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString(configProject), args[0])

	// Get raw JSON
	apiResp, err := func() (string, error) {
		s := cmdutil.Info(messageFetchingData)
		defer s.Stop()

		client := api.DefaultClient(debug)
		return api.ProxyGetIssueRaw(client, key)
	}()
	cmdutil.ExitIfError(err)

	// Get custom field mappings from config
	fieldMappings, err := cmdcommon.GetConfiguredCustomFields()
	if err != nil {
		cmdutil.Warn("Unable to load custom field mappings: %s", err)
		fieldMappings = []jira.IssueTypeField{}
	}

	// Convert custom field IDs to readable names
	jsonOutput, err := jira.TransformIssueFields([]byte(apiResp), fieldMappings)
	if err != nil {
		cmdutil.Failed("Failed to format JSON output: %s", err)
		return
	}

	fmt.Println(string(jsonOutput))
}

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

	key := cmdutil.GetJiraIssueKey(viper.GetString(configProject), args[0])
	iss, err := func() (*jira.Issue, error) {
		s := cmdutil.Info(messageFetchingData)
		defer s.Stop()

		client := api.DefaultClient(debug)
		return api.ProxyGetIssue(client, key, issue.NewNumCommentsFilter(comments))
	}()
	cmdutil.ExitIfError(err)

	plain, err := cmd.Flags().GetBool(flagPlain)
	cmdutil.ExitIfError(err)

	v := tuiView.Issue{
		Server:  viper.GetString(configServer),
		Data:    iss,
		Display: tuiView.DisplayFormat{Plain: plain},
		Options: tuiView.IssueOption{NumComments: comments},
	}
	cmdutil.ExitIfError(v.Render())
}
