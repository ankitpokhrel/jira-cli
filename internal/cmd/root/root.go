package root

import (
	"fmt"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/board"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/completion"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic"
	initCmd "github.com/ankitpokhrel/jira-cli/internal/cmd/init"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/issue"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/me"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/open"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/project"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/sprint"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/version"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"
)

const (
	configDir  = ".config/jira"
	configName = ".jira"
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
			home, err := homedir.Dir()
			if err != nil {
				cmdutil.ExitWithErrMessage(fmt.Sprintf("Error: %s", err))
			}

			viper.AddConfigPath(fmt.Sprintf("%s/%s", home, configDir))
			viper.SetConfigName(configName)
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
			if subCmd == "init" || subCmd == "help" || subCmd == "jira" {
				return
			}

			configFile := viper.ConfigFileUsed()
			if !jiraConfig.Exists(configFile) {
				cmdutil.ExitWithErrMessage("Missing configuration file.\nRun 'jira init' to configure the tool.")
			}
		},
	}

	cmd.PersistentFlags().StringVarP(
		&config, "config", "c", "",
		fmt.Sprintf("Config file (default is $HOME/%s/%s.yml)", configDir, configName),
	)
	cmd.PersistentFlags().StringP(
		"project", "p", "",
		fmt.Sprintf("Jira project to look into (defaults to value from $HOME/%s/%s.yml)", configDir, configName),
	)
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Turn on debug output")

	cmd.SetHelpFunc(helpFunc)

	_ = viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("project", cmd.PersistentFlags().Lookup("project"))

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
	)
}
