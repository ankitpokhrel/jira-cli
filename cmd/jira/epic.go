package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/view"
)

var epicCmd = &cobra.Command{
	Use:   "epic",
	Short: "Epic lists all epics",
	Long:  `Sprint lists current unresolved sprints.`,
	Run:   epic,
}

func epic(*cobra.Command, []string) {
	jql := fmt.Sprintf("project=\"%s\" AND issuetype = \"Epic\" ORDER BY created DESC", viper.Get("project"))

	resp, err := jiraClient.Search(jql)
	if err != nil {
		exitWithError(err)
	}

	v := view.List{Data: resp.Issues}

	if err := v.Render(); err != nil {
		exitWithError(err)
	}
}

func init() {
	rootCmd.AddCommand(epicCmd)
}
