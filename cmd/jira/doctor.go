package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Checks access to jira server",
	Long:  `Check verifies if access to the Jira server can be established.`,
	Run:   doctor,
}

func doctor(*cobra.Command, []string) {
	me, err := jiraClient.Me()
	if err != nil {
		fmt.Printf(
			"Unauthorized. Make sure the token for user %s is correct: %s",
			viper.Get("login"), "https://id.atlassian.com/manage-profile/security/api-tokens",
		)
	} else {
		fmt.Printf("Logged in as %s\n", me.Email)
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
