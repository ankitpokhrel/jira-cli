package create

import (
	"fmt"
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

	params := parseFlags(cmd.Flags())
	client := api.Client(jira.Config{Debug: params.debug})
	cc := createCmd{
		client: client,
		params: params,
	}

	if cc.isNonInteractive() {
		cc.params.noInput = true

		if cc.isMandatoryParamsMissing() {
			cmdutil.Errorf(
				"\u001B[0;31m✗\u001B[0m Mandatory params `--summary` and `--name` is required when using non-interactive mode",
			)
		}
	}

	qs := cc.getQuestions()
	if len(qs) > 0 {
		ans := struct{ Name, Summary, Body, Action string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.name == "" {
			params.name = ans.Name
		}
		if params.summary == "" {
			params.summary = ans.Summary
		}
		if params.body == "" {
			params.body = ans.Body
		}
	}

	// TODO: Remove duplicates with issue/create.
	if !params.noInput {
		answer := struct{ Action string }{}
		for answer.Action != cmdcommon.ActionSubmit {
			err := survey.Ask([]*survey.Question{cmdcommon.GetNextAction()}, &answer)
			cmdutil.ExitIfError(err)

			switch answer.Action {
			case cmdcommon.ActionCancel:
				cmdutil.Errorf("\033[0;31m✗\033[0m Action aborted")
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
	}

	key := func() string {
		s := cmdutil.Info("Creating an epic...")
		defer s.Stop()

		resp, err := client.Create(&jira.CreateRequest{
			Project:       project,
			IssueType:     jira.IssueTypeEpic,
			Name:          params.name,
			Summary:       params.summary,
			Body:          params.body,
			Priority:      params.priority,
			Labels:        params.labels,
			Components:    params.components,
			EpicFieldName: viper.GetString("epic.field"),
		})
		cmdutil.ExitIfError(err)

		return resp.Key
	}()

	fmt.Printf("\033[0;32m✓\033[0m Epic created\n%s/browse/%s\n", server, key)

	if params.assignee != "" {
		user, err := client.UserSearch(&jira.UserSearchOptions{
			Query: params.assignee,
		})
		if err != nil || len(user) == 0 {
			cmdutil.Errorf("\033[0;31m✗\033[0m Unable to find assignee")
		}
		if err = client.AssignIssue(key, user[0].AccountID); err != nil {
			cmdutil.Errorf("\033[0;31m✗\033[0m Unable to set assignee: %s", err.Error())
		}
	}

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

func (cc *createCmd) getQuestions() []*survey.Question {
	var qs []*survey.Question

	if cc.params.name == "" {
		qs = append(qs, &survey.Question{
			Name:     "name",
			Prompt:   &survey.Input{Message: "Epic name"},
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

	var defaultBody string

	if cc.params.template != "" {
		b, err := cmdutil.ReadFile(cc.params.template)
		if err != nil {
			cmdutil.Errorf(fmt.Sprintf("\u001B[0;31m✗\u001B[0m Error: %s", err))
		}
		defaultBody = string(b)
	}

	if cc.params.noInput {
		cc.params.body = defaultBody
		return qs
	}

	if cc.params.body == "" {
		qs = append(qs, &survey.Question{
			Name: "body",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Description",
					Default:       defaultBody,
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: true,
			},
		})
	}

	return qs
}

type createCmd struct {
	client     *jira.Client
	issueTypes []*jira.IssueType
	params     *createParams
}

func (cc *createCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData() || cc.params.template == "-"
}

func (cc *createCmd) isMandatoryParamsMissing() bool {
	return cc.params.summary == "" || cc.params.name == ""
}

type createParams struct {
	name       string
	summary    string
	body       string
	priority   string
	assignee   string
	labels     []string
	components []string
	template   string
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

	assignee, err := flags.GetString("assignee")
	cmdutil.ExitIfError(err)

	labels, err := flags.GetStringArray("label")
	cmdutil.ExitIfError(err)

	components, err := flags.GetStringArray("component")
	cmdutil.ExitIfError(err)

	template, err := flags.GetString("template")
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
		assignee:   assignee,
		labels:     labels,
		components: components,
		template:   template,
		noInput:    noInput,
		debug:      debug,
	}
}
