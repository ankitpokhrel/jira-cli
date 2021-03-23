package epic

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic/add"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic/create"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic/list"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic/remove"
)

const helpText = `Epic manage epics in a given project. See available commands below.`

// NewCmdEpic is an epic command.
func NewCmdEpic() *cobra.Command {
	cmd := cobra.Command{
		Use:         "epic",
		Short:       "Epic manage epics in a project",
		Long:        helpText,
		Aliases:     []string{"epics"},
		Annotations: map[string]string{"cmd:main": "true"},
		RunE:        epic,
	}

	lc := list.NewCmdList()
	cc := create.NewCmdCreate()
	ac := add.NewCmdAdd()
	rc := remove.NewCmdRemove()

	cmd.AddCommand(lc, cc, ac, rc)

	list.SetFlags(lc)
	create.SetFlags(cc)

	return &cmd
}

func epic(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
