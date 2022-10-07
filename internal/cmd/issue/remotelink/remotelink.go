package remotelink

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText     = `Connects an issue to a web link.`
	examples     = `$ jira issue weblink ISSUE-1 http://weblink.com weblink-title`
	optionCancel = "Cancel"
)

// NewCmdWeblink is a link command.
func NewCmdWeblink() *cobra.Command {
	cmd := cobra.Command{
		Use:     "remotelink ISSUE_KEY WEBLINK_URL WEBLINK_TITLE",
		Short:   "Link an issue to a weblink",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"rmln"},
		Annotations: map[string]string{
			"help:args": "ISSUE_KEY\tIssue key, eg: ISSUE-1\n" +
				"WEBLINK_URL\tUrl of the weblink\n" +
				"WEBLINK_TITLE\tTitle of the weblink",
		},
		Run: weblink,
	}

	cmd.Flags().Bool("web", false, "Open issue in web browser after successful linking")

	return &cmd
}

func weblink(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.Client(jira.Config{Debug: params.debug})
	lc := linkCmd{
		client: client,
		params: params,
	}

	err := func() error {
		s := cmdutil.Info("Creating web link for issue")
		defer s.Stop()

		return client.WebLinkIssue(lc.params.issueId, lc.params.title, lc.params.url)
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("Web link created for Issue %s", lc.params.issueId)
	server := viper.GetString("server")

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, lc.params.issueId)
		cmdutil.ExitIfError(err)
	}
}

type linkParams struct {
	issueId string
	url     string
	title   string
	debug   bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *linkParams {
	var issueId, url, title string
	nargs := len(args)
	if nargs >= 1 {
		issueId = cmdutil.GetJiraIssueKey(project, args[0])
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
		issueId: issueId,
		url:     url,
		title:   title,
		debug:   debug,
	}
}

type linkCmd struct {
	client *jira.Client
	params *linkParams
}
