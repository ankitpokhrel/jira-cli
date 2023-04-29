package remote

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
	helpText = `Adds a remote web link to an issue`
	examples = `$ jira issue link remote ISSUE-1 http://weblink.com weblink-title`
)

// NewCmdRemoteLink is a link command.
func NewCmdRemoteLink() *cobra.Command {
	cmd := cobra.Command{
		Use:     "remote ISSUE_KEY WEBLINK_URL WEBLINK_TITLE",
		Short:   "Adds a remote web link to an issue",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"rmln"},
		Annotations: map[string]string{
			"help:args": "ISSUE_KEY\tIssue key, eg: ISSUE-1\n" +
				"WEBLINK_URL\tUrl of the weblink\n" +
				"WEBLINK_TITLE\tTitle of the weblink",
		},
		Run: remotelink,
	}

	return &cmd
}

func remotelink(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)
	lc := linkCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(lc.setIssueKey(project))
	cmdutil.ExitIfError(lc.setRemoteLinkURL())
	cmdutil.ExitIfError(lc.setRemoteLinkTitle())

	err := func() error {
		s := cmdutil.Info("Creating remote web link for issue")
		defer s.Stop()

		return client.RemoteLinkIssue(lc.params.issueKey, lc.params.title, lc.params.url)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Remote web link created for Issue %s", lc.params.issueKey)
	fmt.Printf("%s\n", cmdutil.GenerateServerURL(server, lc.params.issueKey))

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, lc.params.issueKey)
		cmdutil.ExitIfError(err)
	}
}

type linkParams struct {
	issueKey string
	url      string
	title    string
	debug    bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *linkParams {
	var issueKey, url, title string
	nargs := len(args)
	if nargs >= 1 {
		issueKey = cmdutil.GetJiraIssueKey(project, args[0])
	}
	if nargs >= 2 {
		url = args[1]
	}
	if nargs >= 3 {
		title = args[2]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &linkParams{
		issueKey: issueKey,
		url:      url,
		title:    title,
		debug:    debug,
	}
}

type linkCmd struct {
	client *jira.Client
	params *linkParams
}

func (lc *linkCmd) setIssueKey(project string) error {
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

func (lc *linkCmd) setRemoteLinkURL() error {
	if lc.params.url != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "remoteLinkURL",
		Prompt:   &survey.Input{Message: "Remote link URL"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	lc.params.url = ans

	return nil
}

func (lc *linkCmd) setRemoteLinkTitle() error {
	if lc.params.title != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "remoteLinkTitle",
		Prompt:   &survey.Input{Message: "Remote link title"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	lc.params.title = ans

	return nil
}
