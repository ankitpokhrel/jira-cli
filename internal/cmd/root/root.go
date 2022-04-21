package root

import (
	"fmt"
	"os"

	"github.com/ankitpokhrel/jira-cli/pkg/netrc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/board"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/completion"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic"
	initCmd "github.com/ankitpokhrel/jira-cli/internal/cmd/init"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/man"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/me"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/open"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/project"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/sprint"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/version"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"

	"github.com/zalando/go-keyring"
)

const (
	jiraCLIHelpLink  = "https://github.com/ankitpokhrel/jira-cli#getting-started"
	jiraAPITokenLink = "https://id.atlassian.com/manage-profile/security/api-tokens"
)

var (
	config string
	debug  bool
)

func init() {
	cobra.OnInitialize(func() {
		if config != "" {
			viper.SetConfigFile(config)
		} else {
			home, err := cmdutil.GetConfigHome()
			if err != nil {
				cmdutil.Failed("Error: %s", err)
				return
			}

			viper.AddConfigPath(fmt.Sprintf("%s/%s", home, jiraConfig.Dir))
			viper.SetConfigName(jiraConfig.FileName)
			viper.SetConfigType(jiraConfig.FileType)
		}

		viper.AutomaticEnv()
		viper.SetEnvPrefix("jira")

		if err := viper.ReadInConfig(); err == nil && debug {
			fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
		}
	})
}

// NewCmdRoot is a root command.
func NewCmdRoot() *cobra.Command {
	cmd := cobra.Command{
		Use:   "jira",
		Short: "Interactive Jira CLI",
		Long:  "Interactive Jira CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			subCmd := cmd.Name()
			if !cmdRequireToken(subCmd) {
				return
			}

			checkForJiraToken(viper.GetString("server"), viper.GetString("login"))

			configFile := viper.ConfigFileUsed()
			if !jiraConfig.Exists(configFile) {
				cmdutil.Failed("Missing configuration file.\nRun 'jira init' to configure the tool.")
			}
		},
	}

	configHome, err := cmdutil.GetConfigHome()
	if err != nil {
		cmdutil.Failed("Error: %s", err)
	}

	cmd.PersistentFlags().StringVarP(
		&config, "config", "c", "",
		fmt.Sprintf("Config file (default is %s/%s/%s.yml)", configHome, jiraConfig.Dir, jiraConfig.FileName),
	)
	cmd.PersistentFlags().StringP(
		"project", "p", "",
		fmt.Sprintf(
			"Jira project to look into (defaults to %s/%s/%s.yml)",
			configHome, jiraConfig.Dir, jiraConfig.FileName,
		),
	)
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Turn on debug output")

	cmd.SetHelpFunc(helpFunc)

	_ = viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("project.key", cmd.PersistentFlags().Lookup("project"))
	_ = viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug"))

	addChildCommands(&cmd)

	return &cmd
}

func addChildCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		initCmd.NewCmdInit(),
		issue.NewCmdIssue(),
		epic.NewCmdEpic(),
		sprint.NewCmdSprint(),
		board.NewCmdBoard(),
		project.NewCmdProject(),
		open.NewCmdOpen(),
		me.NewCmdMe(),
		completion.NewCmdCompletion(),
		version.NewCmdVersion(),
		man.NewCmdMan(),
	)
}

func cmdRequireToken(cmd string) bool {
	allowList := []string{
		"init",
		"help",
		"jira",
		"version",
		"completion",
		"man",
	}

	for _, item := range allowList {
		if item == cmd {
			return false
		}
	}

	return true
}

func checkForJiraToken(server string, login string) {
	if os.Getenv("JIRA_API_TOKEN") != "" {
		return
	}

	netrcConfig, _ := netrc.Read(server, login)
	if netrcConfig != nil {
		return
	}

	secret, _ := keyring.Get("jira-cli", "token")
	if secret != "" {
		return
	}

	msg := fmt.Sprintf(`The tool needs a Jira API token to function.

For cloud server: you can generate the token using this link: %s
For local server: you can use the password you use to log in to Jira for basic auth or get a token from your Jira profile for PAT.

After generating the token, you can either:
  - Export API token to your shell as a JIRA_API_TOKEN env variable
  - Or, you can use a .netrc file to define required machine details

Once you are done with the above steps, run 'jira init' to generate the config if you haven't already.

For more details, see: %s
`, jiraAPITokenLink, jiraCLIHelpLink)

	cmdutil.Warn(msg)
	os.Exit(1)
}
