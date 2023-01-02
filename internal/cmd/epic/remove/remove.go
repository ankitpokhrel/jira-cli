package remove

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
	helpText = `Remove/unassign epic from issues.`
	examples = `$ jira epic remove ISSUE-1 ISSUE-2`
)

// NewCmdRemove is a remove command.
func NewCmdRemove() *cobra.Command {
	return &cobra.Command{
		Use:     "remove ISSUE-1 [...ISSUE-N]",
		Short:   "Remove/unassign epic from issues",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"rm", "unassign"},
		Annotations: map[string]string{
			"help:args": "ISSUE-1 [...ISSUE-N]\tKey of the issues to remove assigned epic (max 50 issues at once)",
		},
		Run: remove,
	}
}

func remove(cmd *cobra.Command, args []string) {
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
		s := cmdutil.Info("Removing assigned epic from issues...")
		defer s.Stop()

		if projectType != jira.ProjectTypeNextGen {
			return client.EpicIssuesRemove(params.issues...)
		}

		for _, iss := range params.issues {
			if err := client.Edit(iss, &jira.EditRequest{ParentIssueKey: jira.AssigneeNone}); err != nil {
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

	msg := "Epic unassigned from given issues"

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

func parseFlags(flags query.FlagParser, args []string, project string) *removeParams {
	tickets := args[0:]
	issues := make([]string, 0, len(tickets))
	for _, iss := range tickets {
		issues = append(issues, cmdutil.GetJiraIssueKey(project, iss))
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &removeParams{
		issues: issues,
		debug:  debug,
	}
}

func getQuestions(params *removeParams) []*survey.Question {
	var qs []*survey.Question

	if len(params.issues) == 0 {
		qs = append(qs, &survey.Question{
			Name: "issues",
			Prompt: &survey.Input{
				Message: "Issues",
				Help:    "Comma separated list of issues key to remove. eg: ISSUE-1, ISSUE-2",
			},
			Validate: survey.Required,
		})
	}

	return qs
}

type removeParams struct {
	issues []string
	debug  bool
}
