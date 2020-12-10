package jira

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Init initializes jira config",
	Long:    "Init initializes jira configuration required for the tool to work properly.",
	Aliases: []string{"initialize", "configure", "config", "setup"},
	Run:     initialize,
}

func initialize(*cobra.Command, []string) {
	checkForJiraToken()

	c := jiraConfig.NewJiraCLIConfig()

	if err := c.Generate(viper.ConfigFileUsed()); err != nil {
		fmt.Println(err.Error())
		fmt.Printf("\n\033[0;31m✗\033[0m Unable to generate configuration: %s\n", viper.ConfigFileUsed())

		os.Exit(1)
	}

	fmt.Printf("\n\033[0;32m✓\033[0m Configuration generated: %s\n", viper.ConfigFileUsed())
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func checkForJiraToken() {
	const jiraAPITokenLink = "https://id.atlassian.com/manage-profile/security/api-tokens"

	if os.Getenv("JIRA_API_TOKEN") != "" {
		return
	}

	msg := fmt.Sprintf(`
You first need to define JIRA_API_TOKEN env to continue with the initializion process. 

You can generate token from this link: %s

After generating the token, export it to your terminal and run 'jira init' again.
`, jiraAPITokenLink)

	fmt.Println(msg)
	os.Exit(1)
}
