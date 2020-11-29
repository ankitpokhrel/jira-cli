package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Displays configured jira user",
	Long:  `Displays configured jira user.`,
	Run:   me,
}

func me(*cobra.Command, []string) {
	fmt.Println(viper.GetString("login"))
}

func init() {
	rootCmd.AddCommand(meCmd)
}
