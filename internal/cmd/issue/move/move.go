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

	cmd.Flags().SortFlags = false

	cmd.Flags().String("comment", "", "Add comment to the issue")
	cmd.Flags().StringP("assignee", "a", "", "Assign issue to a user")
	cmd.Flags().StringP("resolution", "R", "", "Set resolution")
	cmd.Flags().Bool("web", false, "Open issue in web browser after successful transition")

	return &cmd
}

func move(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	installation := viper.GetString("installation")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)
	mc := moveCmd{
		client:      client,
		transitions: nil,
		params:      params,
	}

	cmdutil.ExitIfError(mc.setIssueKey(project))
	cmdutil.ExitIfError(mc.setAvailableTransitions())
	cmdutil.ExitIfError(mc.setDesiredState(installation))

	if mc.params.state == optionCancel {
		cmdutil.Fail("Action aborted")
		os.Exit(0)
	}

	tr, err := mc.verifyTransition(installation)
	if err != nil {
		fmt.Println()
		cmdutil.Failed("Error: %s", err.Error())
		return
	}

	err = func() error {
		s := cmdutil.Info(fmt.Sprintf("Transitioning issue to %q...", tr.Name))
		defer s.Stop()

		trFieldsReq := jira.TransitionRequestFields{}
		trUpdateReq := jira.TransitionRequestUpdate{}

		if mc.params.assignee != "" {
			trFieldsReq.Assignee = &struct {
				Name string `json:"name"`
			}{Name: mc.params.assignee}
		}
		if mc.params.resolution != "" {
			trFieldsReq.Resolution = &struct {
				Name string `json:"name"`
			}{Name: mc.params.resolution}
		}
		if mc.params.comment != "" {
			trUpdateReq.Comment = []struct {
				Add struct {
					Body string `json:"body"`
				} `json:"add"`
			}{
				{Add: struct {
					Body string `json:"body"`
				}{Body: mc.params.comment}},
			}
		}

		_, err := client.Transition(mc.params.key, &jira.TransitionRequest{
			Fields: &trFieldsReq,
			Update: &trUpdateReq,
			Transition: &jira.TransitionRequestData{
				ID:   tr.ID.String(),
				Name: tr.Name,
			},
		})
		return err
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Issue transitioned to state %q", tr.Name)
	fmt.Printf("%s/browse/%s\n", server, mc.params.key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, mc.params.key)
		cmdutil.ExitIfError(err)
	}
}

type moveParams struct {
	key        string
	state      string
	comment    string
	assignee   string
	resolution string
	debug      bool
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

	comment, err := flags.GetString("comment")
	cmdutil.ExitIfError(err)

	assignee, err := flags.GetString("assignee")
	cmdutil.ExitIfError(err)

	resolution, err := flags.GetString("resolution")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &moveParams{
		key:        key,
		state:      state,
		comment:    comment,
		assignee:   assignee,
		resolution: resolution,
		debug:      debug,
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

func (mc *moveCmd) setDesiredState(it string) error {
	if mc.params.state != "" {
		return nil
	}

	var (
		options = make([]string, 0, len(mc.transitions))
		ans     string
	)

	for _, t := range mc.transitions {
		if it == jira.InstallationTypeCloud && !t.IsAvailable {
			continue
		}
		options = append(options, t.Name)
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

	t, err := api.ProxyTransitions(mc.client, mc.params.key)
	if err != nil {
		return err
	}
	mc.transitions = t

	return nil
}

func (mc *moveCmd) verifyTransition(it string) (*jira.Transition, error) {
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
			"invalid transition state %q\nAvailable states for issue %s: %s",
			mc.params.state, mc.params.key, strings.Join(all, ", "),
		)
	}

	// Jira API v2 doesn't seem to return "isAvailable" field even if the documentation says it does.
	// So, we will only verify if the transition is available for the cloud installation.
	if it == jira.InstallationTypeCloud && !tr.IsAvailable {
		return nil, fmt.Errorf(
			"transition state %q for issue %q is not available",
			mc.params.state, mc.params.key,
		)
	}
	return tr, nil
}
