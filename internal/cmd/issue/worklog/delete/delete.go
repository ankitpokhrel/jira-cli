package delete

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Delete removes a worklog from an issue.`
	examples = `$ jira issue worklog delete

# Delete a specific worklog
$ jira issue worklog delete ISSUE-1 10001

# Delete worklog without confirmation prompt
$ jira issue worklog delete ISSUE-1 10001 --force

# Delete worklog interactively (select from list)
$ jira issue worklog delete ISSUE-1`
)

// NewCmdWorklogDelete is a worklog delete command.
func NewCmdWorklogDelete() *cobra.Command {
	cmd := cobra.Command{
		Use:     "delete ISSUE-KEY [WORKLOG-ID]",
		Short:   "Delete a worklog from an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"remove", "rm"},
		Annotations: map[string]string{
			"help:args": "ISSUE-KEY\tIssue key of the source issue, eg: ISSUE-1\n" +
				"WORKLOG-ID\tID of the worklog to delete (optional, will prompt to select if not provided)",
		},
		Run: deleteWorklog,
	}

	cmd.Flags().SortFlags = false

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	return &cmd
}

func deleteWorklog(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.DefaultClient(params.debug)
	dc := deleteCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(dc.setIssueKey())
	cmdutil.ExitIfError(dc.setWorklogID())

	if !params.force {
		var confirm bool
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to delete worklog %s from issue %s?",
				dc.params.worklogID, dc.params.issueKey),
			Default: false,
		}
		if err := survey.AskOne(prompt, &confirm); err != nil {
			cmdutil.Failed("Confirmation failed: %s", err.Error())
		}
		if !confirm {
			cmdutil.Failed("Action cancelled")
		}
	}

	err := func() error {
		s := cmdutil.Info("Deleting worklog")
		defer s.Stop()

		return client.DeleteIssueWorklog(dc.params.issueKey, dc.params.worklogID)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Worklog deleted from issue %q", dc.params.issueKey)
	fmt.Printf("%s\n", cmdutil.GenerateServerBrowseURL(server, dc.params.issueKey))
}

type deleteParams struct {
	issueKey  string
	worklogID string
	force     bool
	debug     bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *deleteParams {
	var issueKey, worklogID string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), args[0])
	}
	if nargs >= 2 {
		worklogID = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	force, err := flags.GetBool("force")
	cmdutil.ExitIfError(err)

	return &deleteParams{
		issueKey:  issueKey,
		worklogID: worklogID,
		force:     force,
		debug:     debug,
	}
}

type deleteCmd struct {
	client *jira.Client
	params *deleteParams
}

func (dc *deleteCmd) setIssueKey() error {
	if dc.params.issueKey != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "issueKey",
		Prompt:   &survey.Input{Message: "Issue key"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	dc.params.issueKey = cmdutil.GetJiraIssueKey(viper.GetString("project.key"), ans)

	return nil
}

func (dc *deleteCmd) setWorklogID() error {
	if dc.params.worklogID != "" {
		return nil
	}

	// Fetch worklogs for the issue
	worklogs, err := dc.client.GetIssueWorklogs(dc.params.issueKey)
	if err != nil {
		return err
	}

	if worklogs.Total == 0 {
		return fmt.Errorf("no worklogs found for issue %s", dc.params.issueKey)
	}

	// Create options for selection
	options := make([]string, len(worklogs.Worklogs))
	for i, wl := range worklogs.Worklogs {
		options[i] = fmt.Sprintf("%s - %s by %s (%s)", wl.ID, wl.TimeSpent, wl.Author.Name, wl.Started)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select worklog to delete:",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Extract worklog ID from selection (format: "ID - ...")
	var id string
	_, _ = fmt.Sscanf(selected, "%s -", &id)
	dc.params.worklogID = id

	return nil
}
