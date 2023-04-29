package add

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
)

const (
	helpText = `Add issues to sprint.`
	examples = `$ jira sprint add SPRINT_ID ISSUE-1 ISSUE-2`
)

// NewCmdAdd is an add command.
func NewCmdAdd() *cobra.Command {
	return &cobra.Command{
		Use:     "add SPRINT_ID ISSUE-1 [...ISSUE-N]",
		Short:   "Add issues to sprint",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"assign"},
		Annotations: map[string]string{
			"help:args": "SPRINT_ID\t\tID of the sprint on which you want to assign issues to, eg: 123\n" +
				"ISSUE-1 [...ISSUE-N]\tKey of the issues to add to the sprint (max 50 issues at once)",
		},
		Run: add,
	}
}

func add(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")
	params := parseFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)

	qs := getQuestions(params)
	if len(qs) > 0 {
		ans := struct {
			SprintID string
			Issues   string
		}{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.sprintID == "" {
			params.sprintID = ans.SprintID
		}

		if len(params.issues) == 0 {
			issues := strings.Split(ans.Issues, ",")
			for i, iss := range issues {
				issues[i] = cmdutil.GetJiraIssueKey(project, strings.TrimSpace(iss))
			}
			params.issues = issues
		}
	}

	err := func() error {
		s := cmdutil.Info("Adding issues to the sprint...")
		defer s.Stop()

		return client.SprintIssuesAdd(params.sprintID, params.issues...)
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success(fmt.Sprintf("Issues added to the sprint %s\n%s", params.sprintID, cmdutil.GenerateServerURL(server, project)))
}

func parseFlags(flags query.FlagParser, args []string, project string) *addParams {
	var (
		sprintID string
		issues   []string
	)

	nArgs := len(args)
	if nArgs > 0 {
		sprintID = args[0]
	}
	if nArgs > 1 {
		tickets := args[1:]
		issues = make([]string, 0, len(tickets))
		for _, iss := range tickets {
			issues = append(issues, cmdutil.GetJiraIssueKey(project, iss))
		}
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &addParams{
		sprintID: sprintID,
		issues:   issues,
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
	if len(params.issues) == 0 {
		qs = append(qs, &survey.Question{
			Name: "issues",
			Prompt: &survey.Input{
				Message: "Issues",
				Help:    "Comma separated list of issues key to add. eg: ISSUE-1, ISSUE-2",
			},
			Validate: survey.Required,
		})
	}

	return qs
}

type addParams struct {
	sprintID string
	issues   []string
	debug    bool
}
