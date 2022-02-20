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
	cmd := cobra.Command{
		Use:     "init",
		Short:   "Init initializes jira config",
		Long:    "Init initializes jira configuration required for the tool to work properly.",
		Aliases: []string{"initialize", "configure", "config", "setup"},
		Run:     initialize,
	}

	cmd.Flags().Bool("insecure", false, `If set, the tool will skip TLS certificate verification.
This can be useful if your server is using self-signed certificates.`)

	return &cmd
}

func initialize(cmd *cobra.Command, _ []string) {
	insecure, err := cmd.Flags().GetBool("insecure")
	cmdutil.ExitIfError(err)

	c := jiraConfig.NewJiraCLIConfig(jiraConfig.WithInsecureTLS(insecure))

	if insecure {
		cmdutil.Warn(`You are using --insecure option. In this mode, the client will NOT verify
server's certificate chain and host name in requests to the jira server.`)
		fmt.Println()
	}

	file, err := c.Generate()
	if err != nil {
		if e, ok := err.(*jira.ErrUnexpectedResponse); ok {
			fmt.Println()
			cmdutil.Failed("Received unexpected response '%s' from jira. Please try again.", e.Status)
		} else {
			switch err {
			case jiraConfig.ErrSkip:
				cmdutil.Success("Skipping config generation. Current config: %s", viper.ConfigFileUsed())
			case jiraConfig.ErrUnexpectedResponseFormat:
				fmt.Println()
				cmdutil.Failed("Got response in unexpected format when fetching metadata. Please try again.")
			default:
				fmt.Println()
				cmdutil.Failed("Unable to generate configuration: %s", err.Error())
			}
		}
		os.Exit(1)
	}

	cmdutil.Success("Configuration generated: %s", file)
}
