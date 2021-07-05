package init

import (
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
		if e, ok := err.(*jira.ErrUnexpectedResponse); ok {
			cmdutil.Errorf("\n\033[0;31m✗\033[0m Received unexpected response '%s' from jira. Please try again.", e.Status)
		} else {
			switch err {
			case jiraConfig.ErrSkip:
				cmdutil.Success("Skipping config generation. Current config: %s", viper.ConfigFileUsed())
			case jiraConfig.ErrUnexpectedResponseFormat:
				cmdutil.Errorf("\n\033[0;31m✗\033[0m Got response in unexpected format when fetching metadata. Please try again.")
			default:
				cmdutil.Errorf("\n\033[0;31m✗\033[0m Unable to generate configuration: %s", err.Error())
			}
		}
		os.Exit(1)
	}

	cmdutil.Success("Configuration generated: %s", viper.ConfigFileUsed())
}
