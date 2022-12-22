// Package version prints the version information of the tool.
package version

import (
	"fmt"

	"github.com/spf13/cobra"

	v "github.com/ankitpokhrel/jira-cli/internal/version"
)

// NewCmdVersion is a version command.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the app version information",
		Long:  "Print the app version and build information",
		Run:   version,
	}
}

func version(*cobra.Command, []string) {
	fmt.Println(v.Info())
}
