package close

import (
	"fmt"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/spf13/cobra"
)

const (
	helpText = `Close sprint.`
	examples = `$ jira sprint close SPRINT_ID`
)

// NewCmdClose is an add command.
func NewCmdClose() *cobra.Command {
	return &cobra.Command{
		Use:     "close SPRINT_ID",
		Short:   "Close sprint",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"complete"},
		Annotations: map[string]string{
			"help:args": "SPRINT_ID\t\tID of the sprint on which you want to assign issues to, eg: 123\n",
		},
		Run: closeSprint,
	}
}

func closeSprint(cmd *cobra.Command, args []string) {
	params := parseFlags(cmd.Flags(), args)
	client := api.DefaultClient(params.debug)

	qs := getQuestions(params)
	if len(qs) > 0 {
		ans := struct {
			SprintID string
		}{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.sprintID == "" {
			params.sprintID = ans.SprintID
		}
	}

	err := func() error {
		s := cmdutil.Info("Closing sprint...\n")
		defer s.Stop()

		sprintID, err := strconv.Atoi(params.sprintID)
		if err != nil {
			return err
		}

		return client.EndSprint(sprintID)
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success(fmt.Sprintf("Sprint %s has been closed.", params.sprintID))
}

func parseFlags(flags query.FlagParser, args []string) *addParams {
	var sprintID string

	nArgs := len(args)
	if nArgs > 0 {
		sprintID = args[0]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &addParams{
		sprintID: sprintID,
		debug:    debug,
	}
}

func getQuestions(params *addParams) []*survey.Question {
	var qs []*survey.Question

	if params.sprintID == "" {
		qs = append(qs, &survey.Question{
			Name:     "sprintID",
			Prompt:   &survey.Input{Message: "Sprint ID"},
			Validate: survey.Required,
		})
	}

	return qs
}

type addParams struct {
	sprintID string
	debug    bool
}
