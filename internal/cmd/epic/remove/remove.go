package remove

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Remove/unassign epic from issues.`
	examples = `$ jira epic remove ISSUE_1 ISSUE_2`
)

// NewCmdRemove is a remove command.
func NewCmdRemove() *cobra.Command {
	return &cobra.Command{
		Use:     "remove ISSUE_1 [...ISSUE_N]",
		Short:   "Remove/unassign epic from issues",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"rm", "unassign"},
		Annotations: map[string]string{
			"help:args": "ISSUE_1 [...ISSUE_N]\tKey of the issues to remove assigned epic (max 50 issues at once)",
		},
		Run: remove,
	}
}

func remove(cmd *cobra.Command, args []string) {
	params := parseFlags(cmd.Flags(), args)
	client := api.Client(jira.Config{Debug: params.debug})

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
				issues[i] = strings.TrimSpace(iss)
			}
			params.issues = issues
		}
	}

	err := func() error {
		s := cmdutil.Info("Removing assigned epic from issues...")
		defer s.Stop()

		return client.EpicIssuesRemove(params.issues...)
	}()
	cmdutil.ExitIfError(err)

	fmt.Printf("\033[0;32m✓\033[0m Epic unassigned from given issues\n")
}

func parseFlags(flags query.FlagParser, args []string) *removeParams {
	issues := args[0:]

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
				Help:    "Comma separated list of issues key to remove. eg: ISSUE_1, ISSUE_2",
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
