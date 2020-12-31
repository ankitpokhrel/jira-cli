package move

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Move transitions an issue from one state to another.`
	examples = `$ jira issue move ISSUE-1 "In Progress"
$ jira issue move ISSUE-1 Done`
)

// NewCmdMove is a move command.
func NewCmdMove() *cobra.Command {
	cmd := cobra.Command{
		Use:     "move ISSUE_KEY STATE",
		Short:   "Transition an issue to a given state",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"transition"},
		Annotations: map[string]string{
			"help:args": `ISSUE_KEY	Issue key, eg: ISSUE-1
STATE		State you want to transition the issue to`,
		},
		Run: move,
	}

	cmd.Flags().Bool("web", false, "Open issue in web browser after successful transistion")

	return &cmd
}

func move(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())

	qs := getQuestions(params)
	if len(qs) > 0 {
		ans := struct {
			Key   string
			State string
		}{}

		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.key == "" {
			params.key = strings.ToUpper(ans.Key)
		}
		if params.state == "" {
			params.state = ans.State
		}
	}

	client := api.Client(jira.Config{Debug: params.debug})

	transitions := func() []*jira.Transition {
		s := cmdutil.Info("Verifying transition...")
		defer s.Stop()

		transitions, err := client.Transitions(params.key)
		cmdutil.ExitIfError(err)

		return transitions
	}()

	tr := func() *jira.Transition {
		st := strings.ToLower(params.state)
		for _, t := range transitions {
			if strings.ToLower(t.Name) == st {
				return t
			}
		}
		return nil
	}()
	if tr == nil {
		cmdutil.Errorf("\u001B[0;31m✗\u001B[0m Unable to transition issue to state \"%s\"", params.state)
		return
	}
	if !tr.IsAvailable {
		cmdutil.Errorf(
			"\u001B[0;31m✗\u001B[0m Transition state \"%s\" for issue \"%s\" is not available",
			tr.Name, params.key,
		)
		return
	}

	func() {
		s := cmdutil.Info(fmt.Sprintf("Transitioning issue to \"%s\"...", tr.Name))
		defer s.Stop()

		_, err := client.Transition(params.key, &jira.TransitionRequest{
			Transition: &jira.TransitionRequestData{ID: tr.ID.String(), Name: tr.Name},
		})
		cmdutil.ExitIfError(err)
	}()

	server := viper.GetString("server")

	fmt.Printf("\u001B[0;32m✓\u001B[0m Issue transitioned to state \"%s\"\n", tr.Name)
	fmt.Printf("%s/browse/%s\n", server, params.key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, params.key)
		cmdutil.ExitIfError(err)
	}
}

func getQuestions(params *moveParams) []*survey.Question {
	var qs []*survey.Question

	if params.key == "" {
		qs = append(qs, &survey.Question{
			Name:     "key",
			Prompt:   &survey.Input{Message: "Issue key"},
			Validate: survey.Required,
		})
	}

	if params.state == "" {
		qs = append(qs, &survey.Question{
			Name:     "state",
			Prompt:   &survey.Input{Message: "Desired state"},
			Validate: survey.Required,
		})
	}

	return qs
}

type moveParams struct {
	key   string
	state string
	debug bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *moveParams {
	var key, state string

	nargs := len(args)
	if nargs >= 1 {
		key = strings.ToUpper(args[0])
	}
	if nargs >= 2 {
		state = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &moveParams{
		key:   key,
		state: state,
		debug: debug,
	}
}
