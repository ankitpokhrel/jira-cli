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
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Add issues to an epic.`
	examples = `$ jira epic add EPIC-KEY ISSUE-1 ISSUE-2`
)

// NewCmdAdd is an add command.
func NewCmdAdd() *cobra.Command {
	return &cobra.Command{
		Use:     "add EPIC-KEY ISSUE-1 [...ISSUE-N]",
		Short:   "Add issues to an epic",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"assign"},
		Annotations: map[string]string{
			"help:args": "EPIC-KEY\t\tEpic to which you want to assign issues to, eg: EPIC-1\n" +
				"ISSUE-1 [...ISSUE-N]\tKey of the issues to add to an epic (max 50 issues at once)",
		},
		Run: add,
	}
}

func add(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")
	projectType := viper.GetString("project.type")
	params := parseFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)

	qs := getQuestions(params)
	if len(qs) > 0 {
		ans := struct {
			EpicKey string
			Issues  string
		}{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.epicKey == "" {
			params.epicKey = cmdutil.GetJiraIssueKey(project, ans.EpicKey)
		}

		if len(params.issues) == 0 {
			issues := strings.Split(ans.Issues, ",")
			for i, iss := range issues {
				issues[i] = cmdutil.GetJiraIssueKey(project, strings.TrimSpace(iss))
			}
			params.issues = issues
		}
	}

	var (
		failed strings.Builder
		passed bool
	)

	err := func() error {
		s := cmdutil.Info("Adding issues to the epic...")
		defer s.Stop()

		if projectType != jira.ProjectTypeNextGen {
			return client.EpicIssuesAdd(params.epicKey, params.issues...)
		}

		// If the project is of the next-gen type, we need to set the parent property for each issue.
		// There is no way to send bulk update requests as of now, so we need to send these requests
		// in a loop. We will print failed requests with exit code 1 at the end if there are any.
		for _, iss := range params.issues {
			if err := client.Edit(iss, &jira.EditRequest{ParentIssueKey: params.epicKey}); err != nil {
				msg := fmt.Sprintf("\n  - %s: %s", iss, cmdutil.NormalizeJiraError(err.Error()))
				failed.WriteString(msg)
			} else {
				// We will show success message if at-least one request reports success.
				passed = true
			}
		}

		if failed.Len() > 0 {
			return &jira.ErrMultipleFailed{Msg: failed.String()}
		}
		return nil
	}()

	msg := fmt.Sprintf("Issues added to the epic %s\n%s/browse/%s", params.epicKey, server, params.epicKey)

	if projectType != jira.ProjectTypeNextGen {
		cmdutil.ExitIfError(err)
		cmdutil.Success(msg)
	} else {
		if passed {
			cmdutil.Success(msg)
		}
		cmdutil.ExitIfError(err)
	}
}

func parseFlags(flags query.FlagParser, args []string, project string) *addParams {
	var (
		epicKey string
		issues  []string
	)

	nArgs := len(args)
	if nArgs > 0 {
		epicKey = cmdutil.GetJiraIssueKey(project, args[0])
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
		epicKey: epicKey,
		issues:  issues,
		debug:   debug,
	}
}

func getQuestions(params *addParams) []*survey.Question {
	var qs []*survey.Question

	if params.epicKey == "" {
		qs = append(qs, &survey.Question{
			Name:     "epicKey",
			Prompt:   &survey.Input{Message: "Epic Key"},
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
	epicKey string
	issues  []string
	debug   bool
}
