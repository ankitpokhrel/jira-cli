package create

import (
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
$ jira epic create -pPRJ -n"Amazing epic" -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'

# Create epic and set custom fields
# See https://github.com/ankitpokhrel/jira-cli/discussions/346
$ jira epic create -n"Epic with custom fields" --custom story-points=3`
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
	project := viper.GetString("project.key")
	projectType := viper.GetString("project.type")
	installation := viper.GetString("installation")

	params := parseFlags(cmd.Flags())
	client := api.DefaultClient(params.Debug)
	cc := createCmd{
		client: client,
		params: params,
	}

	if cc.isNonInteractive() {
		cc.params.NoInput = true

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

		if params.Name == "" {
			params.Name = ans.Name
		}
		if params.Summary == "" {
			params.Summary = ans.Summary
		}
		if params.Body == "" {
			params.Body = ans.Body
		}
	}

	if !params.NoInput {
		err := cmdcommon.HandleNoInput(params)
		cmdutil.ExitIfError(err)
	}

	params.Reporter = cmdcommon.GetRelevantUser(client, project, params.Reporter)
	params.Assignee = cmdcommon.GetRelevantUser(client, project, params.Assignee)

	key, err := func() (string, error) {
		s := cmdutil.Info("Creating an epic...")
		defer s.Stop()

		cr := jira.CreateRequest{
			Project:      project,
			IssueType:    jira.IssueTypeEpic,
			Summary:      params.Summary,
			Body:         params.Body,
			Reporter:     params.Reporter,
			Assignee:     params.Assignee,
			Priority:     params.Priority,
			Labels:       params.Labels,
			Components:   params.Components,
			FixVersions:  params.FixVersions,
			CustomFields: params.CustomFields,
			EpicField:    viper.GetString("epic.name"),
		}
		if projectType != jira.ProjectTypeNextGen {
			cr.Name = params.Name
		}
		cr.ForProjectType(projectType)
		cr.ForInstallationType(installation)
		if configuredCustomFields, err := cmdcommon.GetConfiguredCustomFields(); err == nil {
			cmdcommon.ValidateCustomFields(cr.CustomFields, configuredCustomFields)
			cr.WithCustomFields(configuredCustomFields)
		}

		resp, err := client.CreateV2(&cr)
		if err != nil {
			return "", err
		}
		return resp.Key, nil
	}()

	cmdutil.ExitIfError(err)
	cmdutil.Success("Epic created\n%s/browse/%s", server, key)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

func (cc *createCmd) getQuestions(projectType string) []*survey.Question {
	var qs []*survey.Question

	if cc.params.Name == "" && projectType != jira.ProjectTypeNextGen {
		qs = append(qs, &survey.Question{
			Name:     "name",
			Prompt:   &survey.Input{Message: "Epic name"},
			Validate: survey.Required,
		})
	}
	if cc.params.Summary == "" {
		qs = append(qs, &survey.Question{
			Name:     "summary",
			Prompt:   &survey.Input{Message: "Summary"},
			Validate: survey.Required,
		})
	}

	var defaultBody string

	if cc.params.Template != "" || cmdutil.StdinHasData() {
		b, err := cmdutil.ReadFile(cc.params.Template)
		if err != nil {
			cmdutil.Failed("Error: %s", err)
		}
		defaultBody = string(b)
	}

	if cc.params.NoInput {
		cc.params.Body = defaultBody
		return qs
	}

	if cc.params.Body == "" {
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
	client *jira.Client
	params *cmdcommon.CreateParams
}

func (cc *createCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData() || cc.params.Template == "-"
}

func (cc *createCmd) isMandatoryParamsMissing() bool {
	return cc.params.Summary == "" || cc.params.Name == ""
}

func parseFlags(flags query.FlagParser) *cmdcommon.CreateParams {
	name, err := flags.GetString("name")
	cmdutil.ExitIfError(err)

	summary, err := flags.GetString("summary")
	cmdutil.ExitIfError(err)

	body, err := flags.GetString("body")
	cmdutil.ExitIfError(err)

	priority, err := flags.GetString("priority")
	cmdutil.ExitIfError(err)

	reporter, err := flags.GetString("reporter")
	cmdutil.ExitIfError(err)

	assignee, err := flags.GetString("assignee")
	cmdutil.ExitIfError(err)

	labels, err := flags.GetStringArray("label")
	cmdutil.ExitIfError(err)

	components, err := flags.GetStringArray("component")
	cmdutil.ExitIfError(err)

	fixVersions, err := flags.GetStringArray("fix-version")
	cmdutil.ExitIfError(err)

	custom, err := flags.GetStringToString("custom")
	cmdutil.ExitIfError(err)

	template, err := flags.GetString("template")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &cmdcommon.CreateParams{
		Name:         name,
		Summary:      summary,
		Body:         body,
		Priority:     priority,
		Reporter:     reporter,
		Assignee:     assignee,
		Labels:       labels,
		Components:   components,
		FixVersions:  fixVersions,
		CustomFields: custom,
		Template:     template,
		NoInput:      noInput,
		Debug:        debug,
	}
}
