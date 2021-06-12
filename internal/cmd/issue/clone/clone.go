package clone

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Clone duplicates an issue and also allow you to override some of the metadata when doing so.`
	examples = `$ jira issue clone ISSUE-1

# Clone issue and modify the summary, priority and assignee
$ jira issue clone ISSUE-1 -s"Modified summary" -yHigh -a$(jira me)

# Clone issue and replace text from summary and description
$ jira issue clone ISSUE-1 -H"find me:replace with me"`
)

// NewCmdClone is a clone command.
func NewCmdClone() *cobra.Command {
	cmd := cobra.Command{
		Use:     "clone ISSUE_KEY",
		Short:   "Clone duplicates an issue",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": "ISSUE_KEY\tKey of the issue to clone, eg: ISSUE-1",
		},
		Args: cobra.MinimumNArgs(1),
		Run:  clone,
	}

	setFlags(&cmd)

	return &cmd
}

func clone(cmd *cobra.Command, args []string) {
	server := viper.GetString("server")
	project := viper.GetString("project")

	params := parseFlags(cmd.Flags())
	client := api.Client(jira.Config{Debug: params.debug})
	cc := cloneCmd{
		client: client,
		params: params,
	}

	key := cmdutil.GetJiraIssueKey(project, args[0])
	issue := func() *jira.Issue {
		s := cmdutil.Info("Fetching issue details...")
		defer s.Stop()

		issue, err := api.Client(jira.Config{Debug: cc.params.debug}).GetIssueV2(key)
		cmdutil.ExitIfError(err)

		return issue
	}()

	cp := cc.getActualCreateParams(issue)

	clonedIssueKey := func() string {
		s := cmdutil.Info(fmt.Sprintf("Cloning %s...", key))
		defer s.Stop()

		cr := jira.CreateRequest{
			Project:    project,
			IssueType:  issue.Fields.IssueType.Name,
			Summary:    cp.summary,
			Body:       cp.body,
			Priority:   cp.priority,
			Labels:     cp.labels,
			Components: cp.components,
		}

		resp, err := client.Create(&cr)
		cmdutil.ExitIfError(err)

		return resp.Key
	}()

	fmt.Printf("\033[0;32m✓\033[0m Issue cloned\n%s/browse/%s\n", server, clonedIssueKey)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		if err := client.LinkIssue(key, clonedIssueKey, "Cloners"); err != nil {
			cmdutil.Errorf("\n\033[0;31m✗\033[0m Unable to link cloned issue")
		}
	}()

	if cp.assignee != "" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			user, err := client.UserSearch(&jira.UserSearchOptions{
				Query: cp.assignee,
			})
			if err != nil || len(user) == 0 {
				cmdutil.Errorf("\n\033[0;31m✗\033[0m Unable to find assignee")
			}
			if err = client.AssignIssue(clonedIssueKey, user[0].AccountID); err != nil {
				cmdutil.Errorf("\n\033[0;31m✗\033[0m Unable to set assignee: %s", err.Error())
			}
		}()
	}

	s := cmdutil.Info("Updating metadata...")
	defer s.Stop()

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, clonedIssueKey)
		cmdutil.ExitIfError(err)
	}

	wg.Wait()
}

type createParams struct {
	summary    string
	body       string
	priority   string
	assignee   string
	labels     []string
	components []string
	replace    string
}

type cloneCmd struct {
	client  *jira.Client
	params  *cloneParams
	cParams *createParams
}

func (cc *cloneCmd) getActualCreateParams(issue *jira.Issue) *createParams {
	cp := createParams{}

	cp.summary = issue.Fields.Summary
	if cc.params.summary != "" {
		cp.summary = cc.params.summary
	}

	cp.priority = issue.Fields.Priority.Name
	if cc.params.priority != "" {
		cp.priority = cc.params.priority
	}

	cp.assignee = issue.Fields.Assignee.Name
	if cc.params.assignee != "" {
		cp.assignee = cc.params.assignee
	}

	cp.labels = issue.Fields.Labels
	if len(cc.params.labels) > 0 {
		cp.labels = cc.params.labels
	}

	components := make([]string, 0, len(issue.Fields.Components))
	for _, v := range issue.Fields.Components {
		components = append(components, v.Name)
	}
	cp.components = components
	if len(cc.params.components) > 0 {
		cp.components = cc.params.components
	}

	var ok bool
	if cp.body, ok = issue.Fields.Description.(string); !ok {
		cp.body = ""
	}

	if cc.params.replace != "" {
		pieces := strings.Split(cc.params.replace, ":")
		if len(pieces) != 2 {
			fmt.Fprintf(
				os.Stderr,
				"\u001B[0;31m✗\u001B[0m Invalid replace string, must be in format <find>:<replace>. Skipping replacement...",
			)
		} else {
			from, to := pieces[0], pieces[1]

			cp.summary = strings.ReplaceAll(cp.summary, from, to)
			cp.body = strings.ReplaceAll(cp.body, from, to)
		}
	}

	return &cp
}

type cloneParams struct {
	summary    string
	priority   string
	assignee   string
	labels     []string
	components []string
	replace    string
	debug      bool
}

func parseFlags(flags query.FlagParser) *cloneParams {
	summary, err := flags.GetString("summary")
	cmdutil.ExitIfError(err)

	priority, err := flags.GetString("priority")
	cmdutil.ExitIfError(err)

	assignee, err := flags.GetString("assignee")
	cmdutil.ExitIfError(err)

	labels, err := flags.GetStringArray("label")
	cmdutil.ExitIfError(err)

	components, err := flags.GetStringArray("component")
	cmdutil.ExitIfError(err)

	replace, err := flags.GetString("replace")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &cloneParams{
		summary:    summary,
		priority:   priority,
		assignee:   assignee,
		labels:     labels,
		components: components,
		replace:    replace,
		debug:      debug,
	}
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("summary", "s", "", "Issue summary or title")
	cmd.Flags().StringP("priority", "y", "", "Issue priority")
	cmd.Flags().StringP("assignee", "a", "", "Issue assignee (email or display name)")
	cmd.Flags().StringArrayP("label", "l", []string{}, "Issue labels")
	cmd.Flags().StringArrayP("component", "C", []string{}, "Issue components")
	cmd.Flags().StringP("replace", "H", "", "Replace strings in summary and body. Format <search>:<replace>, eg: \"find me:replace with me\"")
	cmd.Flags().Bool("web", false, "Open in web browser after successful cloning")
}
