package create

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdcommon"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const (
	helpText = `Create an epic in a given project with minimal information.`
	examples = `$ jira epic create

# Create epic in the configured project
$ jira epic create -n"Epic epic" -s"Everything" -yHigh -lbug -lurgent -b"Bug description"

# Create epic in another project
$ jira epic create -pPRJ -n"Amazing epic" -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'`
)

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	return &cobra.Command{
		Use:     "create",
		Short:   "Create an epic in a project",
		Long:    helpText,
		Example: examples,
		Run:     create,
	}
}

// SetFlags sets flags supported by create command.
func SetFlags(cmd *cobra.Command) {
	cmdcommon.SetCreateFlags(cmd, "Epic")
}

func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	flags := parseFlags(cmd.Flags())
	qs := getQuestions(flags)

	if len(qs) > 0 {
		ans := struct{ Name, Summary, Body, Action string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if flags.name == "" {
			flags.name = ans.Name
		}
		if flags.summary == "" {
			flags.summary = ans.Summary
		}
		if flags.body == "" {
			flags.body = ans.Body
		}
	}

	answer := struct{ Action string }{}
	for answer.Action != cmdcommon.ActionSubmit {
		err := survey.Ask([]*survey.Question{cmdcommon.GetNextAction()}, &answer)
		cmdutil.ExitIfError(err)

		switch answer.Action {
		case cmdcommon.ActionCancel:
			fmt.Print("\033[0;31m✗\033[0m Action aborted\n")
			os.Exit(0)
		case cmdcommon.ActionMetadata:
			ans := struct{ Metadata []string }{}
			err := survey.Ask(cmdcommon.GetMetadata(), &ans)
			cmdutil.ExitIfError(err)

			if len(ans.Metadata) > 0 {
				qs = cmdcommon.GetMetadataQuestions(ans.Metadata)
				ans := struct {
					Priority   string
					Labels     string
					Components string
				}{}
				err := survey.Ask(qs, &ans)
				cmdutil.ExitIfError(err)

				if ans.Priority != "" {
					flags.priority = ans.Priority
				}
				if len(ans.Labels) > 0 {
					flags.labels = strings.Split(ans.Labels, ",")
				}
				if len(ans.Components) > 0 {
					flags.components = strings.Split(ans.Components, ",")
				}
			}
		}
	}

	key := func() string {
		s := cmdutil.Info("Creating an epic...")
		defer s.Stop()

		resp, err := api.Client(jira.Config{Debug: flags.debug}).Create(&jira.CreateRequest{
			Project:       project,
			IssueType:     jira.IssueTypeEpic,
			Name:          flags.name,
			Summary:       flags.summary,
			Body:          flags.body,
			Priority:      flags.priority,
			Labels:        flags.labels,
			Components:    flags.components,
			EpicFieldName: viper.GetString("epic.field"),
		})
		cmdutil.ExitIfError(err)

		return resp.Key
	}()

	fmt.Printf("\033[0;32m✓\033[0m Epic created\n%s/browse/%s\n", server, key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

func getQuestions(params *createParams) []*survey.Question {
	var qs []*survey.Question

	if params.name == "" {
		qs = append(qs, &survey.Question{
			Name:     "name",
			Prompt:   &survey.Input{Message: "Epic name"},
			Validate: survey.Required,
		})
	}
	if params.summary == "" {
		qs = append(qs, &survey.Question{
			Name:     "summary",
			Prompt:   &survey.Input{Message: "Summary"},
			Validate: survey.Required,
		})
	}

	if params.noInput {
		return qs
	}

	if params.body == "" {
		qs = append(qs, &survey.Question{
			Name: "body",
			Prompt: &surveyext.JiraEditor{
				Editor:       &survey.Editor{Message: "Description", HideDefault: true},
				BlankAllowed: true,
			},
		})
	}

	return qs
}

type createParams struct {
	name       string
	summary    string
	body       string
	priority   string
	labels     []string
	components []string
	noInput    bool
	debug      bool
}

func parseFlags(flags query.FlagParser) *createParams {
	name, err := flags.GetString("name")
	cmdutil.ExitIfError(err)

	summary, err := flags.GetString("summary")
	cmdutil.ExitIfError(err)

	body, err := flags.GetString("body")
	cmdutil.ExitIfError(err)

	priority, err := flags.GetString("priority")
	cmdutil.ExitIfError(err)

	labels, err := flags.GetStringArray("label")
	cmdutil.ExitIfError(err)

	components, err := flags.GetStringArray("component")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &createParams{
		name:       name,
		summary:    summary,
		body:       body,
		priority:   priority,
		labels:     labels,
		components: components,
		noInput:    noInput,
		debug:      debug,
	}
}
