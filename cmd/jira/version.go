package jira

import (
	"fmt"

	"github.com/spf13/cobra"

	v "github.com/ankitpokhrel/jira-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the app version information",
	Long:  `Print the app version and build information`,
	Run:   version,
}

func version(*cobra.Command, []string) {
	fmt.Println(v.Info())
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
