package list

import (
	"fmt"

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
		Short:   "List lists boards in a project",
		Long:    "List lists boards in a project.",
		Aliases: []string{"lists", "ls"},
		Run:     List,
	}
}

// List displays a list view.
func List(cmd *cobra.Command, _ []string) {
	project := viper.GetString("project.key")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	boards, total, err := func() ([]*jira.Board, int, error) {
		s := cmdutil.Info(fmt.Sprintf("Fetching boards in project %s...", project))
		defer s.Stop()

		resp, err := api.DefaultClient(debug).Boards(project, jira.BoardTypeAll)
		if err != nil {
			return nil, 0, err
		}
		return resp.Boards, resp.Total, nil
	}()
	cmdutil.ExitIfError(err)

	// Total results in jira API response may not be present in older versions.
	if total == 0 {
		total = len(boards)
	}

	if total == 0 {
		fmt.Println()
		cmdutil.Failed("No boards found in project %q", project)
		return
	}

	v := view.NewBoard(boards)

	cmdutil.ExitIfError(v.Render())
}
