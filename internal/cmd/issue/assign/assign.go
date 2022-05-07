package assign

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
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

	maxResults = 100
	lineBreak  = "----------"

	optionSearch  = "[Search...]"
	optionDefault = "Default"
	optionNone    = "No-one (Unassign)"
	optionCancel  = "Cancel"
)

// NewCmdAssign is an assign command.
func NewCmdAssign() *cobra.Command {
	return &cobra.Command{
		Use:     "assign ISSUE-KEY ASSIGNEE",
		Short:   "Assign issue to a user",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"asg"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1
ASSIGNEE	Email or display name of the user to assign the issue to`,
		},
		Run: assign,
	}
}

func assign(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.Client(jira.Config{Debug: params.debug})
	ac := assignCmd{
		client: client,
		users:  nil,
		params: params,
	}
	lu := strings.ToLower(ac.params.user)

	cmdutil.ExitIfError(ac.setIssueKey(project))

	if lu != strings.ToLower(optionNone) && lu != "x" && lu != jira.AssigneeDefault {
		cmdutil.ExitIfError(ac.setAvailableUsers(project))
		cmdutil.ExitIfError(ac.setAssignee(project))
	}

	u, err := ac.verifyAssignee()
	if err != nil {
		cmdutil.Failed("Error: %s", err.Error())
		return
	}

	var assignee, uname string

	switch {
	case u != nil:
		uname = getQueryableName(u.DisplayName, u.Name)
	case lu == strings.ToLower(optionNone) || lu == "x":
		assignee = jira.AssigneeNone
		uname = "unassigned"
	case lu == strings.ToLower(optionDefault):
		assignee = jira.AssigneeDefault
		uname = assignee
	}

	err = func() error {
		var s *spinner.Spinner
		if uname == "unassigned" {
			s = cmdutil.Info(fmt.Sprintf("Unassigning user from issue \"%s\"...", ac.params.key))
		} else {
			s = cmdutil.Info(fmt.Sprintf("Assigning issue \"%s\" to user \"%s\"...", ac.params.key, uname))
		}
		defer s.Stop()

		return api.ProxyAssignIssue(client, ac.params.key, u, assignee)
	}()
	cmdutil.ExitIfError(err)

	if uname == "unassigned" {
		cmdutil.Success("User unassigned from the issue \"%s\"", ac.params.key)
	} else {
		cmdutil.Success("User \"%s\" assigned to issue \"%s\"", uname, ac.params.key)
	}
	fmt.Printf("%s/browse/%s\n", viper.GetString("server"), ac.params.key)
}

type assignParams struct {
	key   string
	user  string
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *assignParams {
	var key, user string

	nargs := len(args)
	if nargs >= 1 {
		key = cmdutil.GetJiraIssueKey(project, args[0])
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

func (ac *assignCmd) setIssueKey(project string) error {
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
	ac.params.key = cmdutil.GetJiraIssueKey(project, ans)

	return nil
}

func (ac *assignCmd) setAssignee(project string) error {
	if ac.params.user != "" && len(ac.users) == 1 {
		ac.params.user = getQueryableName(ac.users[0].Name, ac.users[0].DisplayName)
		return nil
	}

	lu := strings.ToLower(ac.params.user)
	if lu == strings.ToLower(optionNone) || lu == strings.ToLower(optionDefault) || lu == "x" {
		return nil
	}

	var (
		ans  string
		last bool
	)
	if ac.params.user != "" && len(ac.users) > 0 {
		last = true
	}

	for {
		qs := &survey.Question{
			Name: "user",
			Prompt: &survey.Select{
				Message: "Assign to user:",
				Help:    "Can't find the user? Select search and look for a keyword or cancel to abort",
				Options: ac.getOptions(last),
			},
			Validate: func(val interface{}) error {
				errInvalidSelection := fmt.Errorf("invalid selection")

				ans, ok := val.(core.OptionAnswer)
				if !ok {
					return errInvalidSelection
				}
				if ans.Value == "" || ans.Value == lineBreak {
					return errInvalidSelection
				}

				return nil
			},
		}

		if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
			return err
		}
		if ans == optionCancel {
			cmdutil.Fail("Action aborted")
			os.Exit(0)
		}
		if ans != optionSearch {
			break
		}
		if err := ac.getSearchKeyword(); err != nil {
			return err
		}
		if err := ac.searchAndAssignUser(project); err != nil {
			return err
		}
		last = true
	}
	ac.params.user = ans

	return nil
}

func (ac *assignCmd) getOptions(last bool) []string {
	var validUsers []string

	for _, t := range ac.users {
		if t.Active {
			name := t.DisplayName
			if t.Name != "" {
				name += fmt.Sprintf(" (%s)", t.Name)
			}
			validUsers = append(validUsers, name)
		}
	}
	always := []string{optionDefault, optionNone, optionCancel}
	options := []string{optionSearch}

	if last {
		options = append(options, validUsers...)
		options = append(options, lineBreak)
		options = append(options, always...)
	} else {
		options = append(options, always...)
		options = append(options, lineBreak)
		options = append(options, validUsers...)
	}

	return options
}

func (ac *assignCmd) getSearchKeyword() error {
	qs := &survey.Question{
		Name: "user",
		Prompt: &survey.Input{
			Message: "Search user:",
			Help:    "Type user email or display name to search for a user",
		},
		Validate: func(val interface{}) error {
			errInvalidKeyword := fmt.Errorf("enter atleast 3 characters to search")

			str, ok := val.(string)
			if !ok {
				return errInvalidKeyword
			}
			if len(str) < 3 {
				return errInvalidKeyword
			}

			return nil
		},
	}
	return survey.Ask([]*survey.Question{qs}, &ac.params.user)
}

func (ac *assignCmd) searchAndAssignUser(project string) error {
	u, err := api.ProxyUserSearch(ac.client, &jira.UserSearchOptions{
		Query:      ac.params.user,
		Project:    project,
		MaxResults: maxResults,
	})
	if err != nil {
		return err
	}
	ac.users = u
	return nil
}

func (ac *assignCmd) setAvailableUsers(project string) error {
	s := cmdutil.Info("Fetching available users. Please wait...")
	defer s.Stop()

	return ac.searchAndAssignUser(project)
}

func (ac *assignCmd) verifyAssignee() (*jira.User, error) {
	assignee := strings.ToLower(ac.params.user)
	if assignee == strings.ToLower(optionDefault) || assignee == strings.ToLower(optionNone) || assignee == "x" {
		return nil, nil
	}

	var user *jira.User

	for _, u := range ac.users {
		name := strings.ToLower(getQueryableName(u.Name, u.DisplayName))
		if name == assignee || strings.ToLower(u.Email) == assignee {
			user = u
		}
		if strings.ToLower(fmt.Sprintf("%s (%s)", u.DisplayName, u.Name)) == assignee {
			user = u
		}
		if user != nil {
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid assignee \"%s\"", ac.params.user)
	}
	if !user.Active {
		return nil, fmt.Errorf("user \"%s\" is not active", getQueryableName(user.Name, user.DisplayName))
	}
	return user, nil
}

func getQueryableName(name, displayName string) string {
	if name != "" {
		return name
	}
	return displayName
}
