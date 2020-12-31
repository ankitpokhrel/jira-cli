package init

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// NewCmdInit is an init command.
func NewCmdInit() *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Short:   "Init initializes jira config",
		Long:    "Init initializes jira configuration required for the tool to work properly.",
		Aliases: []string{"initialize", "configure", "config", "setup"},
		Run:     initialize,
	}
}

func initialize(*cobra.Command, []string) {
	c := jiraConfig.NewJiraCLIConfig()

	if err := c.Generate(); err != nil {
		switch err {
		case jiraConfig.ErrSkip:
			fmt.Printf("\033[0;32m✓\033[0m Skipping config generation. Current config: %s\n", viper.ConfigFileUsed())
		case jira.ErrUnexpectedStatusCode:
			cmdutil.Errorf("\n\033[0;31m✗\033[0m Received unexpected status code from jira. Please try again.")
		case jiraConfig.ErrUnexpectedResponseFormat:
			cmdutil.Errorf("\n\033[0;31m✗\033[0m Got response in unexpected format when fetching metadata. Please try again.")
		default:
			cmdutil.Errorf("\n\033[0;31m✗\033[0m Unable to generate configuration: %s", err.Error())
		}
		os.Exit(1)
	}

	fmt.Printf("\n\033[0;32m✓\033[0m Configuration generated: %s\n", viper.ConfigFileUsed())
}
