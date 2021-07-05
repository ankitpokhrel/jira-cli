package move

import (
	"fmt"
	"os"
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

	optionCancel = "Cancel"
)

// NewCmdMove is a move command.
func NewCmdMove() *cobra.Command {
	cmd := cobra.Command{
		Use:     "move ISSUE-KEY STATE",
		Short:   "Transition an issue to a given state",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"transition", "mv"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1
STATE		State you want to transition the issue to`,
		},
		Run: move,
	}

	cmd.Flags().Bool("web", false, "Open issue in web browser after successful transition")

	return &cmd
}

func move(cmd *cobra.Command, args []string) {
	project := viper.GetString("project")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.Client(jira.Config{Debug: params.debug})
	mc := moveCmd{
		client:      client,
		transitions: nil,
		params:      params,
	}

	cmdutil.ExitIfError(mc.setIssueKey(project))
	cmdutil.ExitIfError(mc.setAvailableTransitions())
	cmdutil.ExitIfError(mc.setDesiredState())

	if mc.params.state == optionCancel {
		fmt.Print("\033[0;31m✗\033[0m Action aborted\n")
		os.Exit(0)
	}

	tr, err := mc.verifyTransition()
	if err != nil {
		cmdutil.Errorf(err.Error())
		return
	}

	func() {
		s := cmdutil.Info(fmt.Sprintf("Transitioning issue to \"%s\"...", tr.Name))
		defer s.Stop()

		_, err := client.Transition(mc.params.key, &jira.TransitionRequest{
			Transition: &jira.TransitionRequestData{ID: tr.ID.String(), Name: tr.Name},
		})
		cmdutil.ExitIfError(err)
	}()

	server := viper.GetString("server")

	cmdutil.Success("Issue transitioned to state \"%s\"", tr.Name)
	fmt.Printf("%s/browse/%s\n", server, mc.params.key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, mc.params.key)
		cmdutil.ExitIfError(err)
	}
}

type moveParams struct {
	key   string
	state string
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *moveParams {
	var key, state string

	nargs := len(args)
	if nargs >= 1 {
		key = cmdutil.GetJiraIssueKey(project, args[0])
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

type moveCmd struct {
	client      *jira.Client
	transitions []*jira.Transition
	params      *moveParams
}

func (mc *moveCmd) setIssueKey(project string) error {
	if mc.params.key != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "key",
		Prompt:   &survey.Input{Message: "Issue key"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	mc.params.key = cmdutil.GetJiraIssueKey(project, ans)

	return nil
}

func (mc *moveCmd) setDesiredState() error {
	if mc.params.state != "" {
		return nil
	}

	var (
		options []string
		ans     string
	)

	for _, t := range mc.transitions {
		if t.IsAvailable {
			options = append(options, t.Name)
		}
	}
	options = append(options, optionCancel)

	qs := &survey.Question{
		Name: "state",
		Prompt: &survey.Select{
			Message: "Desired state:",
			Options: options,
		},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	mc.params.state = ans

	return nil
}

func (mc *moveCmd) setAvailableTransitions() error {
	s := cmdutil.Info("Fetching available transitions. Please wait...")
	defer s.Stop()

	t, err := mc.client.Transitions(mc.params.key)
	if err != nil {
		return err
	}
	mc.transitions = t

	return nil
}

func (mc *moveCmd) verifyTransition() (*jira.Transition, error) {
	var tr *jira.Transition

	st := strings.ToLower(mc.params.state)
	all := make([]string, 0, len(mc.transitions))
	for _, t := range mc.transitions {
		if strings.ToLower(t.Name) == st {
			tr = t
		}
		all = append(all, fmt.Sprintf("'%s'", t.Name))
	}

	if tr == nil {
		return nil, fmt.Errorf(
			"\u001B[0;31m✗\u001B[0m Invalid transition state \"%s\"\nAvailable states for issue %s: %s",
			mc.params.state, mc.params.key, strings.Join(all, ", "),
		)
	}
	if !tr.IsAvailable {
		return nil, fmt.Errorf(
			"\u001B[0;31m✗\u001B[0m Transition state \"%s\" for issue \"%s\" is not available",
			mc.params.state, mc.params.key,
		)
	}
	return tr, nil
}
