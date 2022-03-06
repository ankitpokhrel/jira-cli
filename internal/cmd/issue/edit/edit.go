package edit

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
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const (
	helpText = `Edit an issue in a given project with minimal information.`
	examples = `$ jira issue edit ISSUE-1

# Edit issue in the configured project
$ jira issue edit ISSUE-1 -s"New Bug" -yHigh -lbug -lurgent -CBackend -b"Bug description"

# Use --no-input option to disable interactive prompt
$ jira issue edit ISSUE-1 -s"New updated summary" --no-input`
)

// NewCmdEdit is an edit command.
func NewCmdEdit() *cobra.Command {
	cmd := cobra.Command{
		Use:     "edit ISSUE-KEY",
		Short:   "Edit an issue in a project",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"update", "modify"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1`,
		},
		Args: cobra.MinimumNArgs(1),
		Run:  edit,
	}

	setFlags(&cmd)

	return &cmd
}

func edit(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")

	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.Client(jira.Config{Debug: params.debug})
	ec := editCmd{
		client: client,
		params: params,
	}

	issue, err := func() (*jira.Issue, error) {
		s := cmdutil.Info(fmt.Sprintf("Fetching issue %s...", params.issueKey))
		defer s.Stop()

		issue, err := api.ProxyGetIssue(client, params.issueKey)
		if err != nil {
			return nil, err
		}

		return issue, nil
	}()
	cmdutil.ExitIfError(err)

	var (
		isADF        bool
		originalBody string
	)

	if issue.Fields.Description != nil {
		if adfBody, ok := issue.Fields.Description.(*adf.ADF); ok {
			isADF = true
			originalBody = adf.NewTranslator(adfBody, adf.NewJiraMarkdownTranslator()).Translate()
		} else {
			originalBody = issue.Fields.Description.(string)
		}
	}

	cmdutil.ExitIfError(ec.askQuestions(issue, originalBody))

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
					qs := getMetadataQuestions(ans.Metadata, issue)
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

	if params.isEmpty() {
		fmt.Println()
		cmdutil.Failed("Nothing to update")
	}

	// Keep body as is if there were no changes.
	if params.body != "" && params.body == originalBody {
		params.body = ""
	}
	labels := params.labels
	labels = append(labels, issue.Fields.Labels...)

	err = func() error {
		s := cmdutil.Info("Updating an issue...")
		defer s.Stop()

		body := params.body
		if isADF {
			body = md.ToJiraMD(body)
		}

		edr := jira.EditRequest{
			Summary:    params.summary,
			Body:       body,
			Priority:   params.priority,
			Labels:     labels,
			Components: params.components,
		}

		return client.Edit(params.issueKey, &edr)
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("Issue updated\n%s/browse/%s", server, params.issueKey)

	handleUserAssign(project, params.issueKey, params.assignee, client)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, params.issueKey)
		cmdutil.ExitIfError(err)
	}
}

func handleUserAssign(project, key, assignee string, client *jira.Client) {
	if assignee == "" {
		return
	}
	if assignee == "x" {
		if err := api.ProxyAssignIssue(client, key, nil, jira.AssigneeNone); err != nil {
			cmdutil.Failed("Unable to unassign user: %s", err.Error())
		}
		return
	}
	user, err := api.ProxyUserSearch(client, &jira.UserSearchOptions{
		Query:   assignee,
		Project: project,
	})
	if err != nil || len(user) == 0 {
		cmdutil.Failed("Unable to find assignee")
	}
	if err = api.ProxyAssignIssue(client, key, user[0], assignee); err != nil {
		cmdutil.Failed("Unable to set assignee: %s", err.Error())
	}
}

type editCmd struct {
	client *jira.Client
	params *editParams
}

func (ec *editCmd) askQuestions(issue *jira.Issue, originalBody string) error {
	if ec.params.noInput {
		return nil
	}

	var qs []*survey.Question

	if ec.params.summary == "" {
		qs = append(qs, &survey.Question{
			Name: "summary",
			Prompt: &survey.Input{
				Message: "Summary",
				Default: issue.Fields.Summary,
			},
			Validate: survey.Required,
		})
	}

	if ec.params.body == "" {
		qs = append(qs, &survey.Question{
			Name: "body",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Description",
					Default:       originalBody,
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: true,
			},
		})
	}

	ans := struct{ Summary, Body string }{}
	err := survey.Ask(qs, &ans)
	if err != nil {
		return err
	}

	if ec.params.summary == "" {
		ec.params.summary = ans.Summary
	}
	if ec.params.body == "" {
		ec.params.body = ans.Body
	}

	return nil
}

type editParams struct {
	issueKey   string
	summary    string
	body       string
	priority   string
	assignee   string
	labels     []string
	components []string
	noInput    bool
	debug      bool
}

func (ep editParams) isEmpty() bool {
	return ep.summary == "" && ep.body == "" && ep.priority == "" &&
		ep.assignee == "" && len(ep.labels) == 0 && len(ep.components) == 0
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *editParams {
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

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &editParams{
		issueKey:   cmdutil.GetJiraIssueKey(project, args[0]),
		summary:    summary,
		body:       body,
		priority:   priority,
		assignee:   assignee,
		labels:     labels,
		components: components,
		noInput:    noInput,
		debug:      debug,
	}
}

func getMetadataQuestions(meta []string, issue *jira.Issue) []*survey.Question {
	var qs []*survey.Question

	for _, m := range meta {
		switch m {
		case "Priority":
			qs = append(qs, &survey.Question{
				Name:   "priority",
				Prompt: &survey.Input{Message: "Priority", Default: issue.Fields.Priority.Name},
			})
		case "Components":
			qs = append(qs, &survey.Question{
				Name: "components",
				Prompt: &survey.Input{
					Message: "Components",
					Help:    "Comma separated list of valid components. For eg: BE,FE",
				},
			})
		case "Labels":
			qs = append(qs, &survey.Question{
				Name: "labels",
				Prompt: &survey.Input{
					Message: "Labels",
					Help:    "Comma separated list of labels. For eg: backend,urgent",
					Default: strings.Join(issue.Fields.Labels, ","),
				},
			})
		}
	}

	return qs
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("summary", "s", "", "Edit summary or title")
	cmd.Flags().StringP("body", "b", "", "Edit description")
	cmd.Flags().StringP("priority", "y", "", "Edit priority")
	cmd.Flags().StringP("assignee", "a", "", "Edit assignee (email or display name)")
	cmd.Flags().StringArrayP("label", "l", []string{}, "Append labels")
	cmd.Flags().StringArrayP("component", "C", []string{}, "Replace components")
	cmd.Flags().Bool("web", false, "Open in web browser after successful update")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")
}
