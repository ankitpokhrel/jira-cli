package create

import (
	"fmt"

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
$ jira issue create -pPRJ -tBug -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'

# Create issue and set custom fields
# See https://github.com/ankitpokhrel/jira-cli/discussions/346
$ jira issue create -tStory -s"Issue with custom fields" --custom story-points=3

# Load description from template file
$ jira issue create --template /path/to/template.tmpl

# Get description from standard input
$ jira issue create --template -

# Or, use pipe to read input directly from standard input
$ echo "Description from stdin" | jira issue create -s"Summary" -tTask

# For issue description, the flag --body/-b takes precedence over the --template flag
# The example below will add "Body from flag" as an issue description
$ jira issue create -tTask -sSummary -b"Body from flag" --template /path/to/template.tpl`
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
				"Params `--summary` and `--type` is mandatory when using a non-interactive mode",
			)
		}
	}

	cmdutil.ExitIfError(cc.setIssueTypes())
	cmdutil.ExitIfError(cc.askQuestions())

	if !params.NoInput {
		err := cmdcommon.HandleNoInput(params)
		cmdutil.ExitIfError(err)
	}

	params.Reporter = cmdcommon.GetRelevantUser(client, project, params.Reporter)
	params.Assignee = cmdcommon.GetRelevantUser(client, project, params.Assignee)

	key, err := func() (string, error) {
		s := cmdutil.Info("Creating an issue...")
		defer s.Stop()

		cr := jira.CreateRequest{
			Project:        project,
			IssueType:      params.IssueType,
			ParentIssueKey: params.ParentIssueKey,
			Summary:        params.Summary,
			Body:           params.Body,
			Reporter:       params.Reporter,
			Assignee:       params.Assignee,
			Priority:       params.Priority,
			Labels:         params.Labels,
			Components:     params.Components,
			FixVersions:    params.FixVersions,
			CustomFields:   params.CustomFields,
			EpicField:      viper.GetString("epic.link"),
		}
		cr.ForProjectType(projectType)
		cr.ForInstallationType(installation)
		if configuredCustomFields, err := cmdcommon.GetConfiguredCustomFields(); err == nil {
			cmdcommon.ValidateCustomFields(cr.CustomFields, configuredCustomFields)
			cr.WithCustomFields(configuredCustomFields)
		}

		if handle := cmdutil.GetSubtaskHandle(params.IssueType, cc.issueTypes); handle != "" {
			cr.SubtaskField = handle
		}

		resp, err := client.CreateV2(&cr)
		if err != nil {
			return "", err
		}
		return resp.Key, nil
	}()

	cmdutil.ExitIfError(err)
	cmdutil.Success("Issue created\n%s", cmdutil.GenerateServerBrowseURL(server, key))

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, key)
		cmdutil.ExitIfError(err)
	}
}

type createCmd struct {
	client     *jira.Client
	issueTypes []*jira.IssueType
	params     *cmdcommon.CreateParams
}

func (cc *createCmd) setIssueTypes() error {
	issueTypes := make([]*jira.IssueType, 0)
	availableTypes, ok := viper.Get("issue.types").([]interface{})
	if !ok {
		return fmt.Errorf("invalid issue types in config")
	}
	for _, at := range availableTypes {
		tp := at.(map[string]interface{})
		name := tp["name"].(string)
		handle, _ := tp["handle"].(string)
		if handle == jira.IssueTypeEpic || name == jira.IssueTypeEpic {
			continue
		}
		issueTypes = append(issueTypes, &jira.IssueType{
			ID:      tp["id"].(string),
			Name:    name,
			Handle:  handle,
			Subtask: tp["subtask"].(bool),
		})
	}
	cc.issueTypes = issueTypes

	return nil
}

func (cc *createCmd) getIssueType() *survey.Question {
	var qs *survey.Question

	if cc.params.IssueType == "" {
		var options []string
		for _, t := range cc.issueTypes {
			if t.Handle != "" && t.Handle != t.Name {
				options = append(options, fmt.Sprintf("%s (%s)", t.Name, t.Handle))
			} else {
				options = append(options, t.Name)
			}
		}

		qs = &survey.Question{
			Name: "issueType",
			Prompt: &survey.Select{
				Message: "Issue type",
				Options: options,
			},
			Validate: survey.Required,
		}
	}

	return qs
}

func (cc *createCmd) askQuestions() error {
	it := cc.getIssueType()
	if it != nil {
		ans := struct{ IssueType string }{}
		err := survey.Ask([]*survey.Question{it}, &ans)
		if err != nil {
			return err
		}

		if cc.params.IssueType == "" {
			for _, t := range cc.issueTypes {
				if t.Handle != "" && fmt.Sprintf("%s (%s)", t.Name, t.Handle) == ans.IssueType {
					cc.params.IssueType = t.Handle
				} else if t.Name == ans.IssueType {
					cc.params.IssueType = t.Name
				}
			}
		}
	}

	qs := cc.getRemainingQuestions()
	if len(qs) == 0 {
		return nil
	}

	ans := struct{ ParentIssueKey, Summary, Body string }{}
	err := survey.Ask(qs, &ans)
	if err != nil {
		return err
	}

	project := viper.GetString("project.key")

	if cc.params.ParentIssueKey == "" {
		cc.params.ParentIssueKey = cmdutil.GetJiraIssueKey(project, ans.ParentIssueKey)
	} else {
		cc.params.ParentIssueKey = cmdutil.GetJiraIssueKey(project, cc.params.ParentIssueKey)
	}

	if cc.params.Summary == "" {
		cc.params.Summary = ans.Summary
	}
	if cc.params.Body == "" {
		cc.params.Body = ans.Body
	}

	return nil
}

func (cc *createCmd) getRemainingQuestions() []*survey.Question {
	var qs []*survey.Question

	if cc.params.ParentIssueKey == "" {
		for _, t := range cc.issueTypes {
			if t.Subtask && (t.Name == cc.params.IssueType || (t.Handle != "" && t.Handle == cc.params.IssueType)) {
				qs = append(qs, &survey.Question{
					Name:     "parentIssueKey",
					Prompt:   &survey.Input{Message: "Parent issue key"},
					Validate: survey.Required,
				})
			}
		}
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
		if cc.params.Body == "" {
			cc.params.Body = defaultBody
		}
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

func (cc *createCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData() || cc.params.Template == "-"
}

func (cc *createCmd) isMandatoryParamsMissing() bool {
	return cc.params.Summary == "" || cc.params.IssueType == ""
}

func parseFlags(flags query.FlagParser) *cmdcommon.CreateParams {
	issueType, err := flags.GetString("type")
	cmdutil.ExitIfError(err)

	parentIssueKey, err := flags.GetString("parent")
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
		IssueType:      issueType,
		ParentIssueKey: parentIssueKey,
		Summary:        summary,
		Body:           body,
		Priority:       priority,
		Assignee:       assignee,
		Labels:         labels,
		Reporter:       reporter,
		Components:     components,
		FixVersions:    fixVersions,
		CustomFields:   custom,
		Template:       template,
		NoInput:        noInput,
		Debug:          debug,
	}
}
