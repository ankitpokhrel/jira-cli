package init

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

type initParams struct {
	installation string
	server       string
	login        string
	authType     string
	project      string
	board        string
	force        bool
	insecure     bool
}

// NewCmdInit is an init command.
func NewCmdInit() *cobra.Command {
	cmd := cobra.Command{
		Use:     "init",
		Short:   "Init initializes jira config",
		Long:    "Init initializes jira configuration required for the tool to work properly.",
		Aliases: []string{"initialize", "configure", "config", "setup"},
		Run:     initialize,
	}

	cmd.Flags().SortFlags = false

	cmd.Flags().String("installation", "", "Is this a 'cloud' or 'local' jira installation?")
	cmd.Flags().String("server", "", "Link to your jira server")
	cmd.Flags().String("login", "", "Jira login username or email based on your setup")
	cmd.Flags().String("auth-type", "", "Authentication type can be basic, bearer, mtls, or cf-access")
	cmd.Flags().String("project", "", "Your default project key")
	cmd.Flags().String("board", "", "Name of your default board in the project")
	cmd.Flags().Bool("force", false, "Forcefully override existing config if it exists")
	cmd.Flags().Bool("insecure", false, `If set, the tool will skip TLS certificate verification.
This can be useful if your server is using self-signed certificates.`)

	return &cmd
}

func parseFlags(flags query.FlagParser) *initParams {
	installation, err := flags.GetString("installation")
	cmdutil.ExitIfError(err)

	server, err := flags.GetString("server")
	cmdutil.ExitIfError(err)

	login, err := flags.GetString("login")
	cmdutil.ExitIfError(err)

	authType, err := flags.GetString("auth-type")
	cmdutil.ExitIfError(err)

	// If auth type is not provided, check if it's set in env.
	if authType == "" {
		authType = os.Getenv("JIRA_AUTH_TYPE")
	}
	authType = strings.ToLower(authType)

	project, err := flags.GetString("project")
	cmdutil.ExitIfError(err)

	board, err := flags.GetString("board")
	cmdutil.ExitIfError(err)

	force, err := flags.GetBool("force")
	cmdutil.ExitIfError(err)

	insecure, err := flags.GetBool("insecure")
	cmdutil.ExitIfError(err)

	return &initParams{
		installation: installation,
		server:       server,
		login:        login,
		authType:     authType,
		project:      project,
		board:        board,
		force:        force,
		insecure:     insecure,
	}
}

func initialize(cmd *cobra.Command, _ []string) {
	params := parseFlags(cmd.Flags())

	c := jiraConfig.NewJiraCLIConfigGenerator(
		&jiraConfig.JiraCLIConfig{
			Installation: strings.ToLower(params.installation),
			Server:       params.server,
			Login:        params.login,
			AuthType:     params.authType,
			Project:      params.project,
			Board:        params.board,
			Force:        params.force,
			Insecure:     params.insecure,
		},
	)

	if params.insecure {
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
