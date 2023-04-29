package open

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/browser"
)

const (
	helpText = `Open opens issue in a browser. If the issue key is not given, it will open the project page.`
	examples = `$ jira open
$ jira open ISSUE-1`
)

// NewCmdOpen is an open command.
func NewCmdOpen() *cobra.Command {
	cmd := cobra.Command{
		Use:     "open [ISSUE-KEY]",
		Short:   "Open issue in a browser",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"browse", "navigate"},
		Annotations: map[string]string{
			"cmd:main":  "true",
			"help:args": "[ISSUE-KEY]\tIssue key, eg: ISSUE-1",
		},
		Run: open,
	}

	cmd.Flags().BoolP("no-browser", "n", false, `Skip opening destination URL in the browser`)

	return &cmd
}

func open(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")

	var url string

	if len(args) == 0 {
		url = fmt.Sprintf("%s", cmdutil.GenerateServerURL(server, project))
	} else {
		url = fmt.Sprintf("%s", cmdutil.GenerateServerURL(server, cmdutil.GetJiraIssueKey(project, args[0])))
	}

	fmt.Println(url)

	noBrowser, err := cmd.Flags().GetBool("no-browser")
	cmdutil.ExitIfError(err)

	if !noBrowser {
		cmdutil.ExitIfError(browser.Browse(url))
	}
}
