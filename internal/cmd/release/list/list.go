package list

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// NewCmdList is a list command.
func NewCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List lists Jira projects versions",
		Long:    "List lists Jira projects versions that a user has access to.",
		Aliases: []string{"lists", "ls"},
		Run:     List,
	}
}

// List displays a list view.
func List(cmd *cobra.Command, _ []string) {
	project := viper.GetString("project.key")
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	releases, total, err := func() ([]*jira.ProjectVersion, int, error) {
		s := cmdutil.Info("Fetching project versions...")
		defer s.Stop()

		releases, err := api.DefaultClient(debug).Release(project)
		if err != nil {
			return nil, 0, err
		}
		return releases, len(releases), nil
	}()
	cmdutil.ExitIfError(err)

	if total == 0 {
		cmdutil.Failed("No releases found.")
		return
	}

	v := view.NewRelease(releases)

	cmdutil.ExitIfError(v.Render())
}
