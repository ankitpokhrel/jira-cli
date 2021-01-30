package assign

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Assign issue to a user.`
	examples = `$ jira issue assign ISSUE-1 jon@domain.tld

# Assignee name or email needs to be an exact match
$ jira issue assign ISSUE-1 "Jon Doe"

# Assign to self
$ jira issue assign ISSUE-1 $(jira me)

# Assign to default assignee
$ jira issue assign ISSUE-1 default

# Unassign
$ jira issue assign ISSUE-1 x`

	maxSearchResults = 100
	assigneeDefault  = "Default"
	assigneeNone     = "No-one (Unassign)"
)

// NewCmdAssign is an assign command.
func NewCmdAssign() *cobra.Command {
	return &cobra.Command{
		Use:     "assign ISSUE_KEY ASSIGNEE",
		Short:   "Assign issue to a user",
		Long:    helpText,
		Example: examples,
		Annotations: map[string]string{
			"help:args": `ISSUE_KEY	Issue key, eg: ISSUE-1
ASSIGNEE	Exact email or display name of the user to assign the issue to`,
		},
		Run: assign,
	}
}

func assign(cmd *cobra.Command, args []string) {
	params := parseArgsAndFlags(args, cmd.Flags())
	client := api.Client(jira.Config{Debug: params.debug})
	ac := assignCmd{
		client: client,
		users:  nil,
		params: params,
	}

	cmdutil.ExitIfError(ac.setIssueKey())
	cmdutil.ExitIfError(ac.setAvailableUsers())
	cmdutil.ExitIfError(ac.setAssignee())

	u, err := ac.verifyAssignee()
	if err != nil {
		cmdutil.Errorf(err.Error())
		return
	}

	var assignee, uname string

	lu := strings.ToLower(ac.params.user)

	switch {
	case u != nil:
		assignee = u.AccountID
		uname = u.Name
	case lu == strings.ToLower(assigneeNone) || lu == "x":
		assignee = jira.AssigneeNone
		uname = "unassigned"
	case lu == strings.ToLower(assigneeDefault):
		assignee = jira.AssigneeDefault
		uname = assignee
	}

	func() {
		var s *spinner.Spinner
		if uname == "unassigned" {
			s = cmdutil.Info(fmt.Sprintf("Unassigning user from issue \"%s\"...", ac.params.key))
		} else {
			s = cmdutil.Info(fmt.Sprintf("Assigning issue \"%s\" to user \"%s\"...", ac.params.key, uname))
		}
		defer s.Stop()

		err := client.AssignIssue(ac.params.key, assignee)
		cmdutil.ExitIfError(err)
	}()

	if uname == "unassigned" {
		fmt.Printf("\u001B[0;32m✓\u001B[0m User unassigned from the issue \"%s\"\n", ac.params.key)
	} else {
		fmt.Printf("\u001B[0;32m✓\u001B[0m User \"%s\" assigned to issue \"%s\"\n", uname, ac.params.key)
	}
	fmt.Printf("%s/browse/%s\n", viper.GetString("server"), ac.params.key)
}

type assignParams struct {
	key   string
	user  string
	debug bool
}

func parseArgsAndFlags(args []string, flags query.FlagParser) *assignParams {
	var key, user string

	nargs := len(args)
	if nargs >= 1 {
		key = strings.ToUpper(args[0])
	}
	if nargs >= 2 {
		user = args[1]
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &assignParams{
		key:   key,
		user:  user,
		debug: debug,
	}
}

type assignCmd struct {
	client *jira.Client
	users  []*jira.User
	params *assignParams
}

func (ac *assignCmd) setIssueKey() error {
	if ac.params.key != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "key",
		Prompt:   &survey.Input{Message: "Issue key"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	ac.params.key = ans

	return nil
}

func (ac *assignCmd) setAssignee() error {
	if ac.params.user != "" {
		return nil
	}

	options := []string{assigneeDefault, assigneeNone}
	for _, t := range ac.users {
		if t.Active {
			options = append(options, t.Name)
		}
	}

	var ans string

	qs := &survey.Question{
		Name: "user",
		Prompt: &survey.Select{
			Message: "Assign to user:",
			Options: options,
		},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	ac.params.user = ans

	return nil
}

func (ac *assignCmd) setAvailableUsers() error {
	s := cmdutil.Info("Fetching available users. Please wait...")
	defer s.Stop()

	t, err := ac.client.UserSearch(&jira.UserSearchOptions{
		Query:      ac.params.key,
		MaxResults: maxSearchResults,
	})
	if err != nil {
		return err
	}
	ac.users = t

	return nil
}

func (ac *assignCmd) verifyAssignee() (*jira.User, error) {
	u, d, n := strings.ToLower(ac.params.user), strings.ToLower(assigneeDefault), strings.ToLower(assigneeNone)
	if u == d || u == n || u == "x" {
		return nil, nil
	}

	var user *jira.User

	st := strings.ToLower(ac.params.user)
	all := make([]string, 0, len(ac.users))
	for _, u := range ac.users {
		if strings.ToLower(u.Name) == st || strings.ToLower(u.Email) == st {
			user = u
		}
		all = append(all, fmt.Sprintf("'%s'", u.Name))
	}

	if user == nil {
		return nil, fmt.Errorf("\u001B[0;31m✗\u001B[0m Invalid assignee \"%s\"", ac.params.user)
	}
	if !user.Active {
		return nil, fmt.Errorf("\u001B[0;31m✗\u001B[0m User \"%s\" is not active", user.Name)
	}
	return user, nil
}
