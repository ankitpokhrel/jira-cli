package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Create a new sprint.`
	examples = `$ jira sprint create MySprint

# Add a start and end date to the sprint:
$ jira sprint create --start 2025-08-25 --end 2025-08-31 MySprint

# Also add a goal for the sprint:
$ jira sprint create --start 2025-08-25 --end 2025-08-31 --goal "Fix all bugs" MySprint

# Omit some parameters on purpose:
$ jira sprint create --no-input MySprint

# Get JSON output:
$ jira sprint create --raw MySprint
`
)

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	cmd := cobra.Command{
		Use:     "create SPRINT-NAME",
		Short:   "Create new sprint",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "SPRINT_NAME\t\tThe name of the sprint to be created",
		},
		Run: create,
	}

	cmd.Flags().Bool("raw", false, "Print output in JSON format")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")
	cmd.Flags().StringP("start", "s", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringP("end", "e", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringP("goal", "g", "", "Goal of the sprint")

	return &cmd
}

func create(cmd *cobra.Command, args []string) {
	params := parseFlags(cmd.Flags(), args)
	client := api.DefaultClient(params.Debug)

	qs := getQuestions(params)
	if len(qs) > 0 {
		ans := struct {
			SprintName string
			StartDate  string
			EndDate    string
			Goal       string
		}{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.SprintName == "" {
			params.SprintName = ans.SprintName
		}
		if params.StartDate == "" {
			params.StartDate = ans.StartDate
		}
		if params.EndDate == "" {
			params.EndDate = ans.EndDate
		}
		if params.Goal == "" {
			params.Goal = ans.Goal
		}
	}

	if (params.StartDate != "" && params.EndDate == "") || (params.StartDate == "" && params.EndDate != "") {
		cmdutil.Failed("Either both start and end dates must be supplied, or none of them")
	}
	cr := jira.SprintCreateRequest{
		Name:          params.SprintName,
		StartDate:     params.StartDate,
		EndDate:       params.EndDate,
		Goal:          params.Goal,
		OriginBoardID: viper.GetInt("board.id"),
	}
	sprint, err := client.CreateSprint(&cr)
	cmdutil.ExitIfError(err)

	if params.Raw {
		jsonData, err := json.Marshal(sprint)
		cmdutil.ExitIfError(err)
		fmt.Println(string(jsonData))
		return
	}

	cmdutil.Success("Sprint '%s' with id '%d' created\n", sprint.Name, sprint.ID)
}

func parseFlags(flags query.FlagParser, args []string) *SprintCreateParams {
	var sprintName string

	if len(args) > 0 {
		sprintName = args[0]
	}

	start, err := flags.GetString("start")
	cmdutil.ExitIfError(err)
	if start != "" {
		if err = validateDate(start); err != nil {
			cmdutil.Failed("Invalid start date. Should be in YYYY-MM-DD format")
		}
	}

	end, err := flags.GetString("end")
	cmdutil.ExitIfError(err)
	if end != "" {
		if err = validateDate(end); err != nil {
			cmdutil.Failed("Invalid end date. Should be in YYYY-MM-DD format")
		}
	}

	goal, err := flags.GetString("goal")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	raw, err := flags.GetBool("raw")
	cmdutil.ExitIfError(err)

	return &SprintCreateParams{
		SprintName: sprintName,
		StartDate:  start,
		EndDate:    end,
		Goal:       goal,
		Debug:      debug,
		NoInput:    noInput,
		Raw:        raw,
	}
}

func getQuestions(params *SprintCreateParams) []*survey.Question {
	var qs []*survey.Question

	if params.SprintName == "" {
		qs = append(qs, &survey.Question{
			Name:     "SprintName",
			Prompt:   &survey.Input{Message: "Sprint Name"},
			Validate: survey.Required,
		})
	}

	if params.NoInput {
		return qs
	}

	if params.StartDate == "" {
		qs = append(qs, &survey.Question{
			Name:     "StartDate",
			Prompt:   &survey.Input{Message: "Start date (YYYY-MM-DDD)"},
			Validate: validateDate,
		})
	}

	if params.EndDate == "" {
		qs = append(qs, &survey.Question{
			Name:     "EndDate",
			Prompt:   &survey.Input{Message: "End date (YYYY-MM-DDD)"},
			Validate: validateDate,
		})
	}

	if params.Goal == "" {
		qs = append(qs, &survey.Question{
			Name:   "Goal",
			Prompt: &survey.Input{Message: "Goal"},
		})
	}

	return qs
}

type SprintCreateParams struct {
	SprintName string
	StartDate  string
	EndDate    string
	Goal       string
	Debug      bool
	NoInput    bool
	Raw        bool
}

// Returns an error if the date is not in the form YYYY-MM-DD. Technically, the
// JIRA API accepts other formats, but YYYY-MM-DD should be enough for most
// users.
func validateDate(val interface{}) error {
	// We allow empty dates for the start and end of the sprint
	if val.(string) == "" {
		return nil
	}

	_, err := time.Parse(time.DateOnly, val.(string))
	if err != nil {
		return errors.New("Invalid date")
	}
	return nil
}
