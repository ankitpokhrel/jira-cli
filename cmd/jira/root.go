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
	debug      bool
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
	checkForJiraToken()

	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initJiraClient)

	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Config file (default is $HOME/.jira.yml)")
	rootCmd.PersistentFlags().StringVarP(&project, "project", "p", "", "Jira project to look into (defaults to value from $HOME/.jira.yml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Turn on debug output")

	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
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

	if err := viper.ReadInConfig(); err == nil && debug {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}
}

func initJiraClient() {
	config := jira.Config{
		Server:   viper.GetString("server"),
		Login:    viper.GetString("login"),
		APIToken: viper.GetString("api_token"),
		Debug:    debug,
	}

	jiraClient = jira.NewClient(config, jira.WithTimeout(clientTimeout))
}

func exitIfError(err error) {
	if err != nil {
		var msg string

		switch err {
		case jira.ErrUnexpectedStatusCode:
			msg = "Received unexpected response code from jira. Please check the parameters you supplied and try again."
		case jira.ErrEmptyResponse:
			msg = "Received empty response from jira. Please try again."
		default:
			msg = err.Error()
		}

		_, _ = fmt.Fprintf(os.Stderr, "\nError: %s\n", msg)
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

	if !debug {
		s.Start()
	}

	return s
}

func checkForJiraToken() {
	const jiraAPITokenLink = "https://id.atlassian.com/manage-profile/security/api-tokens"

	if os.Getenv("JIRA_API_TOKEN") != "" {
		return
	}

	msg := fmt.Sprintf(`You need to define JIRA_API_TOKEN env for the tool to work. 

You can generate a token using this link: %s

After generating the token, export it to your shell and run 'jira init' if you haven't already.`, jiraAPITokenLink)

	fmt.Println(msg)
	os.Exit(1)
}
