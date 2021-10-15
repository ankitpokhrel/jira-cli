package project

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// NewCmdProject is a project command.
func NewCmdProject() *cobra.Command {
	return &cobra.Command{
		Use:         "project",
		Short:       "All accessible jira projects",
		Long:        "Project lists all jira projects that a user has access to.",
		Aliases:     []string{"projects"},
		Annotations: map[string]string{"cmd:main": "true"},
		Run:         projects,
	}
}

func projects(cmd *cobra.Command, _ []string) {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	projects, total, err := func() ([]*jira.Project, int, error) {
		s := cmdutil.Info("Fetching projects...")
		defer s.Stop()

		projects, err := api.Client(jira.Config{Debug: debug}).Project()
		if err != nil {
			return nil, 0, err
		}
		return projects, len(projects), nil
	}()
	cmdutil.ExitIfError(err)

	if total == 0 {
		cmdutil.Failed("No projects found.")
		return
	}

	v := view.NewProject(projects)

	cmdutil.ExitIfError(v.Render())
}
