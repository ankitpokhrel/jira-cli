package move

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const helpText = `Move transitions an issue from one state to another.`

// NewCmdMove is a move command.
func NewCmdMove() *cobra.Command {
	return &cobra.Command{
		Use:     "move ISSUE_KEY STATE",
		Short:   "Transition an issue to a given state",
		Long:    helpText,
		Args:    cobra.MinimumNArgs(2), //nolint:gomnd
		Aliases: []string{"transition"},
		Run:     move,
	}
}

func move(cmd *cobra.Command, args []string) {
	key, state := args[0], args[1]

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.Client(jira.Config{Debug: debug})

	transitions := func() []*jira.Transition {
		s := cmdutil.Info("Verifying transition...")
		defer s.Stop()

		transitions, err := client.Transitions(key)
		cmdutil.ExitIfError(err)

		return transitions
	}()

	tr := func() *jira.Transition {
		st := strings.ToLower(state)
		for _, t := range transitions {
			if strings.ToLower(t.Name) == st {
				return t
			}
		}
		return nil
	}()
	if tr == nil {
		cmdutil.PrintErrF("\u001B[0;31m✗\u001B[0m Unable to transition issue to state \"%s\"", state)
		return
	}
	if !tr.IsAvailable {
		cmdutil.PrintErrF("\u001B[0;31m✗\u001B[0m Transition state \"%s\" for issue \"%s\" is not available", tr.Name, key)
		return
	}

	func() {
		s := cmdutil.Info(fmt.Sprintf("Transitioning issue to \"%s\"...", tr.Name))
		defer s.Stop()

		_, err := client.Transition(key, &jira.TransitionRequest{
			Transition: &jira.TransitionRequestData{ID: tr.ID.String(), Name: tr.Name},
		})
		cmdutil.ExitIfError(err)
	}()

	fmt.Printf("\u001B[0;32m✓\u001B[0m Issue transitioned to state \"%s\"\n", tr.Name)
}
