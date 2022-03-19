package remotelink

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
	helpText = `Remote link creates or update a remote issue link for an issue.`
	examples = `$ jira issue remotelink ISSUE-1 https://example.com "Example page"`
)

// NewCmdRemoteLink is a remotelink command.
func NewCmdRemoteLink() *cobra.Command {
	cmd := cobra.Command{
		Use:     "remotelink ISSUE_KEY REMOTE_LINK_URL REMOTE_LINK_TITLE",
		Short:   "Remotelink adds link to an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"rln"},
		Annotations: map[string]string{
			"help:args": "ISSUE_KEY\tKey of the issue, eg: ISSUE-1\n" +
				"REMOTE_LINK_URL\tLinked url, eg: https://example.com\n" +
				"REMOTE_LINK_TITLE\tLinked url title, eg: Example page",
		},
		Args: cobra.MinimumNArgs(3),
		Run:  remoteLink,
	}

	cmd.Flags().Bool("web", false, "Open issue in web browser after successful linking")

	return &cmd
}

func remoteLink(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.Client(jira.Config{Debug: params.debug})
	lc := remoteLinkCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(lc.setIssueKey(project))
	cmdutil.ExitIfError(lc.setRemoteLinkURL(project))
	cmdutil.ExitIfError(lc.setRemoteLinkTitle())

	err := func() error {
		s := cmdutil.Info("Linking url")
		defer s.Stop()

		return client.AddIssueRemoteLink(lc.params.issueKey, lc.params.remoteLinkURL, lc.params.remoteLinkTitle)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Remote link \"%s\"(%s) added", lc.params.remoteLinkTitle, lc.params.remoteLinkURL)
	fmt.Printf("%s/browse/%s\n", server, lc.params.issueKey)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, lc.params.issueKey)
		cmdutil.ExitIfError(err)
	}
}

type remoteLinkParams struct {
	issueKey        string
	remoteLinkURL   string
	remoteLinkTitle string
	debug           bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *remoteLinkParams {
	var issueKey, remoteLinkURL, remoteLinkTitle string

	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(project, args[0])
	}
	if nargs >= 2 {
		remoteLinkURL = args[1]
	}
	if nargs >= 3 {
		remoteLinkTitle = args[2]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &remoteLinkParams{
		issueKey:        issueKey,
		remoteLinkURL:   remoteLinkURL,
		remoteLinkTitle: remoteLinkTitle,
		debug:           debug,
	}
}

type remoteLinkCmd struct {
	client *jira.Client
	params *remoteLinkParams
}

func (lc *remoteLinkCmd) setIssueKey(project string) error {
	if lc.params.issueKey != "" {
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
	lc.params.issueKey = cmdutil.GetJiraIssueKey(project, ans)

	return nil
}

func (lc *remoteLinkCmd) setRemoteLinkURL(project string) error {
	if lc.params.remoteLinkURL != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "remoteLinkUrl",
		Prompt:   &survey.Input{Message: "Remote link url"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	lc.params.remoteLinkURL = ans

	return nil
}

func (lc *remoteLinkCmd) setRemoteLinkTitle() error {
	if lc.params.remoteLinkTitle != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name: "remoteLinkTitle",
		Prompt: &survey.Input{
			Message: "Remote link title:",
		},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	lc.params.remoteLinkTitle = ans

	return nil
}
