package create

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const helpText = `Create an issue in a given project with minimal information.

EG:
	# Create issue in configured project
	jira issue create -tBug -s"New Bug" -yHigh -lbug -lurgent -b"Bug description"

	# Create issue in another project
	jira issue create -pPRJ -tBug -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'
`

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create an issue in a project",
		Long:  helpText,
		Run:   create,
	}
}

func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	flags := parseFlags(cmd.Flags())
	qs := getQuestions(flags)

	if len(qs) > 0 {
		ans := struct {
			IssueType string
			Summary   string
			Body      string
		}{}

		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if flags.issueType == "" {
			flags.issueType = ans.IssueType
		}
		if flags.summary == "" {
			flags.summary = ans.Summary
		}
		if flags.body == "" {
			flags.body = ans.Body
		}
	}

	key := func() string {
		s := cmdutil.Info("Creating an issue...")
		defer s.Stop()

		resp, err := api.Client(jira.Config{Debug: flags.debug}).Create(&jira.CreateRequest{
			Project:   project,
			IssueType: flags.issueType,
			Summary:   flags.summary,
			Body:      flags.body,
			Priority:  flags.priority,
			Labels:    flags.labels,
		})
		cmdutil.ExitIfError(err)

		return resp.Key
	}()

	fmt.Printf("\u001B[0;32m✓\u001B[0m Issue created: %s/browse/%s\n", server, key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

// SetFlags sets flags supported by create command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("type", "t", "", "Issue type")
	cmd.Flags().StringP("summary", "s", "", "Issue summary or title")
	cmd.Flags().StringP("body", "b", "", "Issue description")
	cmd.Flags().StringP("priority", "y", "", "Issue priority")
	cmd.Flags().StringArrayP("label", "l", []string{}, "Issue labels")
	cmd.Flags().Bool("web", false, "Open issue in web browser after successful creation")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")
}

func getQuestions(params *createParams) []*survey.Question {
	var qs []*survey.Question

	if params.issueType == "" {
		qs = append(qs, &survey.Question{
			Name:     "issueType",
			Prompt:   &survey.Input{Message: "Issue type"},
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
	if !params.noInput && params.body == "" {
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
	issueType string
	summary   string
	body      string
	priority  string
	labels    []string
	noInput   bool
	debug     bool
}

func parseFlags(flags query.FlagParser) *createParams {
	issueType, err := flags.GetString("type")
	cmdutil.ExitIfError(err)

	summary, err := flags.GetString("summary")
	cmdutil.ExitIfError(err)

	body, err := flags.GetString("body")
	cmdutil.ExitIfError(err)

	priority, err := flags.GetString("priority")
	cmdutil.ExitIfError(err)

	labels, err := flags.GetStringArray("label")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &createParams{
		issueType: issueType,
		summary:   summary,
		body:      body,
		priority:  priority,
		labels:    labels,
		noInput:   noInput,
		debug:     debug,
	}
}
