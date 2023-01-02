package serverinfo

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// NewCmdServerInfo is a server info command.
func NewCmdServerInfo() *cobra.Command {
	return &cobra.Command{
		Use:     "serverinfo",
		Short:   "Displays information about the Jira instance",
		Long:    "Displays information about the Jira instance.",
		Aliases: []string{"systeminfo"},
		Run:     serverInfo,
	}
}

func serverInfo(cmd *cobra.Command, _ []string) {
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	info, err := func() (*jira.ServerInfo, error) {
		s := cmdutil.Info("Fetching server info...")
		defer s.Stop()

		info, err := api.DefaultClient(debug).ServerInfo()
		if err != nil {
			return nil, err
		}
		return info, nil
	}()
	cmdutil.ExitIfError(err)

	v := view.NewServerInfo(info)

	cmdutil.ExitIfError(v.Render())
}
