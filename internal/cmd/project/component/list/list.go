package list

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/view"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const tabWidth = 8

// NewCmdList is a list command.
func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List lists Jira project components",
		Long:    "List lists Jira project components for the given Jira project.",
		Aliases: []string{"lists", "ls"},
		Run:     List,
	}

	cmd.Flags().Bool("plain", false, "Display output in plain mode")
	cmd.Flags().Bool("raw", false, "Print raw JSON output")

	return cmd
}

// List displays a list view.
func List(cmd *cobra.Command, _ []string) {
	project := viper.GetString("project.key")
	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	components, total, err := func() ([]*jira.ProjectComponent, int, error) {
		s := cmdutil.Info("Fetching project components...")
		defer s.Stop()

		components, err := api.ProxyProjectComponents(api.DefaultClient(debug), project)
		if err != nil {
			return nil, 0, err
		}
		return components, len(components), nil
	}()
	cmdutil.ExitIfError(err)

	if total == 0 {
		cmdutil.Failed("No components found.")
		return
	}

	raw, err := cmd.Flags().GetBool("raw")
	cmdutil.ExitIfError(err)

	if raw {
		outputRawJSON(components)
		return
	}

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	if plain {
		outputPlain(components)
		return
	}

	v := view.NewComponent(components)

	cmdutil.ExitIfError(v.Render())
}

func outputRawJSON(components []*jira.ProjectComponent) {
	data, err := json.MarshalIndent(components, "", "  ")
	if err != nil {
		cmdutil.Failed("Failed to marshal components to JSON: %s", err)
		return
	}
	fmt.Println(string(data))
}

func outputPlain(components []*jira.ProjectComponent) {
	w := tabwriter.NewWriter(os.Stdout, 0, tabWidth, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")

	for _, c := range components {
		desc := ""
		if c.Description != nil {
			desc = fmt.Sprint(c.Description)
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, c.Name, desc)
	}

	_ = w.Flush()
}
