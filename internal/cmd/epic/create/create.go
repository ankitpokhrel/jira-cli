package create

import (
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

//nolint:gocyclo
// TODO: Reduce cyclomatic complexity.
func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")
	projectType := viper.GetString("project.type")

	params := parseFlags(cmd.Flags())
	client := api.Client(jira.Config{Debug: params.debug})
	cc := createCmd{
		client: client,
		params: params,
	}

	if cc.isNonInteractive() {
		cc.params.noInput = true

		if cc.isMandatoryParamsMissing() {
			cmdutil.Failed(
				"Params `--summary` and `--name` is mandatory when using a non-interactive mode",
			)
		}
	}

	qs := cc.getQuestions(projectType)
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
				cmdutil.Failed("Action aborted")
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

	key, err := func() (string, error) {
		s := cmdutil.Info("Creating an epic...")
		defer s.Stop()

		cr := jira.CreateRequest{
			Project:    project,
			IssueType:  jira.IssueTypeEpic,
			Summary:    params.summary,
			Body:       params.body,
			Priority:   params.priority,
			Labels:     params.labels,
			Components: params.components,
			EpicField:  viper.GetString("epic.name"),
		}
		if projectType != jira.ProjectTypeNextGen {
			cr.Name = params.name
		}

		resp, err := client.CreateV2(&cr)
		if err != nil {
			return "", err
		}
		return resp.Key, nil
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("Epic created\n%s/browse/%s", server, key)

	if params.assignee != "" {
		user, err := api.ProxyUserSearch(client, &jira.UserSearchOptions{
			Query:   params.assignee,
			Project: project,
		})
		if err != nil || len(user) == 0 {
			cmdutil.Failed("Unable to find assignee")
		}
		if err = api.ProxyAssignIssue(client, key, user[0], jira.AssigneeDefault); err != nil {
			cmdutil.Failed("Unable to set assignee: %s", err.Error())
		}
	}

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

func (cc *createCmd) getQuestions(projectType string) []*survey.Question {
	var qs []*survey.Question

	if cc.params.name == "" && projectType != jira.ProjectTypeNextGen {
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

	if cc.params.template != "" || cmdutil.StdinHasData() {
		b, err := cmdutil.ReadFile(cc.params.template)
		if err != nil {
			cmdutil.Failed("Error: %s", err)
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
