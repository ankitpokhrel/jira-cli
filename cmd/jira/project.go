package jira

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/view"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "All accessible jira projects",
	Long:  `Project lists all jira projects that a user has access to.`,
	Run:   projects,
}

func projects(*cobra.Command, []string) {
	resp, err := jiraClient.Project()
	if err != nil {
		exitWithError(err)
	}

	v := view.Project{Data: resp}

	if err := v.Render(); err != nil {
		exitWithError(err)
	}
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
