package open

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
)

const (
	helpText = `Open opens issue in a browser. If the issue key is not given, it will open the project page.`
	examples = `$ jira open
$ jira open ISSUE-1`
)

// NewCmdOpen is an open command.
func NewCmdOpen() *cobra.Command {
	return &cobra.Command{
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
}

func open(_ *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")

	var url string

	if len(args) == 0 {
		url = fmt.Sprintf("%s/browse/%s", server, project)
	} else {
		url = fmt.Sprintf("%s/browse/%s", server, cmdutil.GetJiraIssueKey(project, args[0]))
	}

	fmt.Println(url)
	cmdutil.ExitIfError(browser.OpenURL(url))
}
