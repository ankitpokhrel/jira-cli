package jira

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/internal/view"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in a project",
	Long:  `List lists all issues in a given project.`,
	Run:   list,
}

func list(*cobra.Command, []string) {
	jql := fmt.Sprintf("project=\"%s\" ORDER BY created DESC", viper.Get("project"))

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
	rootCmd.AddCommand(listCmd)
}
