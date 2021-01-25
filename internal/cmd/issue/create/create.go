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
	helpText = `Create an issue in a given project with minimal information.`
	examples = `$ jira issue create

# Create issue in the configured project
$ jira issue create -tBug -s"New Bug" -yHigh -lbug -lurgent -b"Bug description"

# Create issue in another project
$ jira issue create -pPRJ -tBug -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'`
)

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	return &cobra.Command{
		Use:     "create",
		Short:   "Create an issue in a project",
		Long:    helpText,
		Example: examples,
		Run:     create,
	}
}

// SetFlags sets flags supported by create command.
func SetFlags(cmd *cobra.Command) {
	cmdcommon.SetCreateFlags(cmd, "Issue")
}

func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	params := parseFlags(cmd.Flags())
	client := api.Client(jira.Config{Debug: params.debug})
	cc := createCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(cc.setIssueTypes())

	qs := cc.getQuestions()
	if len(qs) > 0 {
		ans := struct{ IssueType, Summary, Body string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.issueType == "" {
			params.issueType = ans.IssueType
		}
		if params.summary == "" {
			params.summary = ans.Summary
		}
		if params.body == "" {
			params.body = ans.Body
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
					params.priority = ans.Priority
				}
				if len(ans.Labels) > 0 {
					params.labels = strings.Split(ans.Labels, ",")
				}
				if len(ans.Components) > 0 {
					params.components = strings.Split(ans.Components, ",")
				}
			}
		}
	}

	key := func() string {
		s := cmdutil.Info("Creating an issue...")
		defer s.Stop()

		cr := jira.CreateRequest{
			Project:    project,
			IssueType:  params.issueType,
			Summary:    params.summary,
			Body:       params.body,
			Priority:   params.priority,
			Labels:     params.labels,
			Components: params.components,
		}

		resp, err := client.Create(&cr)
		cmdutil.ExitIfError(err)

		return resp.Key
	}()

	fmt.Printf("\033[0;32m✓\033[0m Issue created\n%s/browse/%s\n", server, key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

type createCmd struct {
	client     *jira.Client
	issueTypes []*jira.IssueType
	params     *createParams
}

func (cc *createCmd) setIssueTypes() error {
	issueTypes := make([]*jira.IssueType, 0)
	availableTypes, ok := viper.Get("issue.types").([]interface{})
	if !ok {
		return fmt.Errorf("invalid issue types in config")
	}
	for _, at := range availableTypes {
		tp := at.(map[interface{}]interface{})
		st := tp["subtask"].(bool)
		if st {
			continue
		}
		name := tp["name"].(string)
		if name == jira.IssueTypeEpic {
			continue
		}
		issueTypes = append(issueTypes, &jira.IssueType{
			ID:      tp["id"].(string),
			Name:    name,
			Subtask: st,
		})
	}
	cc.issueTypes = issueTypes

	return nil
}

func (cc *createCmd) getQuestions() []*survey.Question {
	var qs []*survey.Question

	if cc.params.issueType == "" {
		var options []string
		for _, t := range cc.issueTypes {
			options = append(options, t.Name)
		}

		qs = append(qs, &survey.Question{
			Name: "issueType",
			Prompt: &survey.Select{
				Message: "Issue type:",
				Options: options,
			},
			Validate: survey.Required,
		})
	}
	if cc.params.summary == "" {
		qs = append(qs, &survey.Question{
			Name:     "summary",
			Prompt:   &survey.Input{Message: "Summary"},
			Validate: survey.Required,
		})
	}

	if cc.params.noInput {
		return qs
	}

	if cc.params.body == "" {
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
	issueType  string
	summary    string
	body       string
	priority   string
	labels     []string
	components []string
	noInput    bool
	debug      bool
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

	components, err := flags.GetStringArray("component")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &createParams{
		issueType:  issueType,
		summary:    summary,
		body:       body,
		priority:   priority,
		labels:     labels,
		components: components,
		noInput:    noInput,
		debug:      debug,
	}
}
