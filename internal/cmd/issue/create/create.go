package create

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const helpText = `Create an issue in a given project with minimal information.

EG:
	# Create issue in configured project
	jira issue create -tBug -s"New Bug" -yHigh -lbug -lurgent -b"Bug description"

	# Create issue in another project
	jira issue create -pPRJ -tBug -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'
`

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create an issue in a project",
		Long:  helpText,
		Run:   create,
	}
}

func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	issueType, err := cmd.Flags().GetString("type")
	cmdutil.ExitIfError(err)

	summary, err := cmd.Flags().GetString("summary")
	cmdutil.ExitIfError(err)

	body, err := cmd.Flags().GetString("body")
	cmdutil.ExitIfError(err)

	priority, err := cmd.Flags().GetString("priority")
	cmdutil.ExitIfError(err)

	labels, err := cmd.Flags().GetStringArray("label")
	cmdutil.ExitIfError(err)

	key := func() string {
		s := cmdutil.Info("Creating an issue...")
		defer s.Stop()

		resp, err := api.Client(jira.Config{Debug: debug}).Create(&jira.CreateRequest{
			Project:   project,
			IssueType: issueType,
			Summary:   summary,
			Body:      body,
			Priority:  priority,
			Labels:    labels,
		})
		cmdutil.ExitIfError(err)

		return resp.Key
	}()

	fmt.Printf("\u001B[0;32m✓\u001B[0m Issue created: %s/browse/%s\n", server, key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

// SetFlags sets flags supported by create command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("type", "t", "", "Issue type")
	cmd.Flags().StringP("summary", "s", "", "Issue summary or title")
	cmd.Flags().StringP("body", "b", "", "Issue description")
	cmd.Flags().StringP("priority", "y", "", "Issue priority")
	cmd.Flags().StringArrayP("label", "l", []string{}, "Issue labels")
	cmd.Flags().Bool("web", false, "Open issue in web browser after successful creation")

	cmdutil.ExitIfError(cmd.MarkFlagRequired("type"))
	cmdutil.ExitIfError(cmd.MarkFlagRequired("summary"))
}
