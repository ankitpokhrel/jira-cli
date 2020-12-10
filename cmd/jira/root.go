package jira

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	clientTimeout = 15 * time.Second
	refreshRate   = 100 * time.Millisecond
)

var (
	config     string
	project    string
	jiraClient *jira.Client

	rootCmd = &cobra.Command{
		Use:   "jira",
		Short: "Jira cli designed for developers",
		Long:  `A jira command line designed for developers to help with frequent jira chores.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initJiraClient)

	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Config file (default is $HOME/.jira.yml)")
	rootCmd.PersistentFlags().StringVarP(&project, "project", "p", "", "Jira project to look into (defaults to value from $HOME/.jira.yml)")

	_ = viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
}

func initConfig() {
	if config != "" {
		viper.SetConfigFile(config)
	} else {
		home, err := homedir.Dir()
		exitIfError(err)

		viper.AddConfigPath(home)
		viper.SetConfigName(".jira")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("jira")

	if err := viper.ReadInConfig(); err == nil {
		// TODO: Only display this debug mode
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func initJiraClient() {
	config := jira.Config{
		Server:   viper.GetString("server"),
		Login:    viper.GetString("login"),
		APIToken: viper.GetString("api_token"),
	}

	jiraClient = jira.NewClient(config, jira.WithTimeout(clientTimeout))
}

func exitIfError(err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func info(msg string) *spinner.Spinner {
	s := spinner.New(
		spinner.CharSets[14],
		refreshRate,
		spinner.WithSuffix(" "+msg),
		spinner.WithHiddenCursor(true),
	)

	s.Start()

	return s
}
