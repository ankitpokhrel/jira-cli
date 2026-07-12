package list

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rethab/jira-cli/api"
	"github.com/rethab/jira-cli/internal/cmdutil"
	"github.com/rethab/jira-cli/internal/view"
	"github.com/rethab/jira-cli/pkg/jira"
)

const (
	flagRaw = "raw"

	helpText = `List lists Jira projects that a user has access to.`
	examples = `$ jira project list

# Get the raw Jira API response
$ jira project list --raw`
)

// NewCmdList is a list command.
func NewCmdList() *cobra.Command {
	cmd := cobra.Command{
		Use:     "list",
		Short:   "List lists Jira projects",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"lists", "ls"},
		Run:     List,
	}
	cmd.Flags().Bool(flagRaw, false, "Print raw Jira API response")

	return &cmd
}

// List displays a list view.
func List(cmd *cobra.Command, _ []string) {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	raw, err := cmd.Flags().GetBool(flagRaw)
	cmdutil.ExitIfError(err)

	if raw {
		listRaw(debug)
		return
	}

	projects, total, err := func() ([]*jira.Project, int, error) {
		s := cmdutil.Info("Fetching projects...")
		defer s.Stop()

		projects, err := api.DefaultClient(debug).Project()
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

func listRaw(debug bool) {
	apiResp, err := func() (string, error) {
		s := cmdutil.Info("Fetching projects...")
		defer s.Stop()

		return api.DefaultClient(debug).ProjectRaw()
	}()
	cmdutil.ExitIfError(err)

	fmt.Println(apiResp)
}
