package view

import (
	"fmt"
	"strings"

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

# Get raw JSON with only specific fields
$ jira issue view ISSUE-1 --raw --api-fields "key,summary,status"

# Get JSON output with human-readable custom field names
$ jira issue view ISSUE-1 --json

# Get JSON with only specific fields from API
$ jira issue view ISSUE-1 --json --api-fields "key,summary,status,Story Points"

# Get JSON filtered to specific nested paths
$ jira issue view ISSUE-1 --json --json-filter "key,fields.summary,fields.status.statusCategory.name"

# Combine both for maximum efficiency and precision
$ jira issue view ISSUE-1 --json --api-fields "key,summary,status" --json-filter "key,fields.summary,fields.status.statusCategory.name"`

	flagRaw        = "raw"
	flagJSON       = "json"
	flagDebug      = "debug"
	flagComments   = "comments"
	flagPlain      = "plain"
	flagNoWarnings = "no-warnings"

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
	cmd.Flags().String("api-fields", "", "Comma-separated list of fields to fetch from Jira API (e.g., 'key,summary,status,assignee'). "+
		"Use Jira field names, human-readable names from your config (e.g., 'Story Points', 'Sprint'), "+
		"custom field IDs (e.g., 'customfield_10001'), or special values like '*navigable' (common fields), '*all' (most fields). "+
		"Only works with --json or --raw. If not specified, returns all fields.")
	cmd.Flags().String("json-filter", "", "Comma-separated list of JSON paths to include in output (e.g., 'key,fields.summary,fields.status.statusCategory.name'). "+
		"Allows precise filtering of nested JSON fields after API response. "+
		"Only works with --json. If not specified, includes all fields from API response.")
	cmd.Flags().Bool(flagNoWarnings, false, "Suppress warnings about field name collisions. Only works with --json")

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

	// Get fields for API-level filtering and translate names to IDs
	fieldsStr, err := cmd.Flags().GetString("api-fields")
	cmdutil.ExitIfError(err)

	fields := cmdcommon.TranslateFieldNames(fieldsStr)

	apiResp, err := func() (string, error) {
		s := cmdutil.Info(messageFetchingData)
		defer s.Stop()

		client := api.DefaultClient(debug)
		return api.ProxyGetIssueRaw(client, key, fields)
	}()
	cmdutil.ExitIfError(err)

	fmt.Println(apiResp)
}

func viewJSON(cmd *cobra.Command, args []string) {
	debug, err := cmd.Flags().GetBool(flagDebug)
	cmdutil.ExitIfError(err)

	key := cmdutil.GetJiraIssueKey(viper.GetString(configProject), args[0])

	// Get fields for API-level filtering and translate names to IDs
	fieldsStr, err := cmd.Flags().GetString("api-fields")
	cmdutil.ExitIfError(err)

	fields := cmdcommon.TranslateFieldNames(fieldsStr)

	// Get raw JSON with API field filtering
	apiResp, err := func() (string, error) {
		s := cmdutil.Info(messageFetchingData)
		defer s.Stop()

		client := api.DefaultClient(debug)
		return api.ProxyGetIssueRaw(client, key, fields)
	}()
	cmdutil.ExitIfError(err)

	// Get custom field mappings from config
	fieldMappings, err := cmdcommon.GetConfiguredCustomFields()
	if err != nil {
		cmdutil.Warn("Unable to load custom field mappings: %s", err)
		fieldMappings = []jira.IssueTypeField{}
	}

	// Get filter for output-level filtering (optional)
	jsonFilter, err := cmd.Flags().GetString("json-filter")
	cmdutil.ExitIfError(err)

	var filterFields []string
	if jsonFilter != "" {
		// For json-filter, the user provides direct JSON paths
		// Split by comma and trim spaces
		filterFields = []string{}
		for _, field := range strings.Split(jsonFilter, ",") {
			field = strings.TrimSpace(field)
			if field != "" {
				filterFields = append(filterFields, field)
			}
		}
	}

	result, err := jira.TransformIssueFields([]byte(apiResp), fieldMappings, filterFields)
	if err != nil {
		cmdutil.Failed("Failed to format JSON output: %s", err)
		return
	}

	// Display warnings if any (unless suppressed)
	noWarnings, err := cmd.Flags().GetBool(flagNoWarnings)
	cmdutil.ExitIfError(err)

	if !noWarnings {
		for _, warning := range result.Warnings {
			cmdutil.Warn(warning)
		}
	}

	fmt.Println(string(result.Data))
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
