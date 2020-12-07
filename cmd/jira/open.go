package jira

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var openCmd = &cobra.Command{
	Use:     "open [ISSUE KEY]",
	Short:   "Open issue in a browser",
	Long:    "Open opens issue in a browser. If the issue key is not given, it will open the project page.",
	Aliases: []string{"browse", "navigate"},
	Run:     open,
}

func open(_ *cobra.Command, args []string) {
	server := viper.GetString("server")

	var url string

	if len(args) == 0 {
		url = fmt.Sprintf("%s/browse/%s", server, viper.GetString("project"))
	} else {
		url = fmt.Sprintf("%s/browse/%s", server, args[0])
	}

	exitIfError(browser.OpenURL(url))
}

func init() {
	rootCmd.AddCommand(openCmd)
}
