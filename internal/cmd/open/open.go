package open

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
)

const helpText = `Open opens issue in a browser. If the issue key is not given, it will open the project page.`

// NewCmdOpen is an open command.
func NewCmdOpen() *cobra.Command {
	return &cobra.Command{
		Use:     "open [ISSUE KEY]",
		Short:   "Open issue in a browser",
		Long:    helpText,
		Aliases: []string{"browse", "navigate"},
		Run:     open,
	}
}

func open(_ *cobra.Command, args []string) {
	server := viper.GetString("server")

	var url string

	if len(args) == 0 {
		url = fmt.Sprintf("%s/browse/%s", server, viper.GetString("project"))
	} else {
		url = fmt.Sprintf("%s/browse/%s", server, args[0])
	}

	cmdutil.ExitIfError(browser.OpenURL(url))
}
