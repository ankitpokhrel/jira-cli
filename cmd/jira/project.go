package jira

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/view"
)

var projectCmd = &cobra.Command{
	Use:     "project",
	Short:   "All accessible jira projects",
	Long:    `Project lists all jira projects that a user has access to.`,
	Aliases: []string{"projects"},
	Run:     projects,
}

func projects(*cobra.Command, []string) {
	resp, err := jiraClient.Project()
	exitIfError(err)

	v := view.NewProject(resp)

	exitIfError(v.Render())
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
