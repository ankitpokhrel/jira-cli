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
	helpText = `Create an issue in a given project with minimal information.`
	examples = `$ jira issue create

# Create issue in the configured project
$ jira issue create -tBug -s"New Bug" -yHigh -lbug -lurgent -b"Bug description"

# Create issue in another project
$ jira issue create -pPRJ -tBug -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'

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
				"Params `--summary` and `--type` is mandatory when using a non-interactive mode",
			)
		}
	}

	cmdutil.ExitIfError(cc.setIssueTypes())
	cmdutil.ExitIfError(cc.askQuestions())

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
					qs := cmdcommon.GetMetadataQuestions(ans.Metadata)
					ans := struct {
						Priority    string
						Labels      string
						Components  string
						FixVersions string
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
					if len(ans.FixVersions) > 0 {
						params.fixVersions = strings.Split(ans.FixVersions, ",")
					}
				}
			}
		}
	}

	key, err := func() (string, error) {
		s := cmdutil.Info("Creating an issue...")
		defer s.Stop()

		cr := jira.CreateRequest{
			Project:        project,
			IssueType:      params.issueType,
			ParentIssueKey: params.parentIssueKey,
			Summary:        params.summary,
			Body:           params.body,
			Priority:       params.priority,
			Labels:         params.labels,
			Components:     params.components,
			FixVersions:    params.fixVersions,
			CustomFields:   params.customFields,
			EpicField:      viper.GetString("epic.link"),
		}
		cr.ForProjectType(projectType)

		if strings.EqualFold(params.issueType, jira.IssueTypeSubTask) {
			cr.SubtaskField = cmdutil.GetSubtaskHandle(cc.issueTypes)
		}

		resp, err := client.CreateV2(&cr)
		if err != nil {
			return "", err
		}
		return resp.Key, nil
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("Issue created\n%s/browse/%s", server, key)

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

	if cc.params.issueType == "" {
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

		if cc.params.issueType == "" {
			for _, t := range cc.issueTypes {
				if t.Handle != "" && fmt.Sprintf("%s (%s)", t.Name, t.Handle) == ans.IssueType {
					cc.params.issueType = t.Handle
				} else if t.Name == ans.IssueType {
					cc.params.issueType = t.Name
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

	if cc.params.parentIssueKey == "" {
		cc.params.parentIssueKey = cmdutil.GetJiraIssueKey(project, ans.ParentIssueKey)
	} else {
		cc.params.parentIssueKey = cmdutil.GetJiraIssueKey(project, cc.params.parentIssueKey)
	}

	if cc.params.summary == "" {
		cc.params.summary = ans.Summary
	}
	if cc.params.body == "" {
		cc.params.body = ans.Body
	}

	return nil
}

func (cc *createCmd) getRemainingQuestions() []*survey.Question {
	var qs []*survey.Question

	if cc.params.parentIssueKey == "" {
		for _, t := range cc.issueTypes {
			if t.Subtask && (t.Name == cc.params.issueType || (t.Handle != "" && t.Handle == cc.params.issueType)) {
				qs = append(qs, &survey.Question{
					Name:     "parentIssueKey",
					Prompt:   &survey.Input{Message: "Parent issue key"},
					Validate: survey.Required,
				})
			}
		}
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
		if cc.params.body == "" {
			cc.params.body = defaultBody
		}
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

func (cc *createCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData() || cc.params.template == "-"
}

func (cc *createCmd) isMandatoryParamsMissing() bool {
	return cc.params.summary == "" || cc.params.issueType == ""
}

type createParams struct {
	issueType      string
	parentIssueKey string
	summary        string
	body           string
	priority       string
	assignee       string
	labels         []string
	components     []string
	fixVersions    []string
	customFields   map[string]string
	template       string
	noInput        bool
	debug          bool
}

func parseFlags(flags query.FlagParser) *createParams {
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

	return &createParams{
		issueType:      issueType,
		parentIssueKey: parentIssueKey,
		summary:        summary,
		body:           body,
		priority:       priority,
		assignee:       assignee,
		labels:         labels,
		components:     components,
		fixVersions:    fixVersions,
		customFields:   custom,
		template:       template,
		noInput:        noInput,
		debug:          debug,
	}
}
