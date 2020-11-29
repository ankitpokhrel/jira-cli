package jira

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var openCmd = &cobra.Command{
	Use:     "open [ISSUE KEY]",
	Short:   "Open issue in a browser.",
	Long:    `Open opens the issue in a browser.`,
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"browse"},
	Run:     open,
}

func open(_ *cobra.Command, args []string) {
	url := fmt.Sprintf("%s/browse/%s", viper.Get("server"), args[0])

	if err := browser.OpenURL(url); err != nil {
		exitWithError(err)
	}
}

func init() {
	rootCmd.AddCommand(openCmd)
}
