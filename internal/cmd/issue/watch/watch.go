package watch

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Adds user to issue watchers.`
	examples = `$ jira issue watch ISSUE-1 jon@domain.tld

# Watcher name or email needs to be an exact match
$ jira issue watch ISSUE-1 "Jon Doe"

# Add self to watchers
$ jira issue watch ISSUE-1 $(jira me)`

	maxResults = 100
	lineBreak  = "----------"

	optionSearch = "[Search...]"
	optionCancel = "Cancel"
)

// NewCmdWatch is an watch command.
func NewCmdWatch() *cobra.Command {
	return &cobra.Command{
		Use:     "watch ISSUE-KEY WATCHER",
		Short:   "Add user to issue watchers",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"wat"},
		Annotations: map[string]string{
			"help:args": `ISSUE-KEY	Issue key, eg: ISSUE-1
WATCHER	Email or display name of the user to add to issue watchers`,
		},
		Run: watch,
	}
}

func watch(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.DefaultClient(params.debug)
	ac := watchCmd{
		client: client,
		users:  nil,
		params: params,
	}

	cmdutil.ExitIfError(ac.setIssueKey(project))

	cmdutil.ExitIfError(ac.setAvailableUsers(project))
	cmdutil.ExitIfError(ac.setWatcher(project))

	u, err := ac.verifyWatcher()
	if err != nil {
		cmdutil.Failed("Error: %s", err.Error())
		return
	}

	uname := getQueryableName(u.DisplayName, u.Name)

	err = func() error {
		s := cmdutil.Info(fmt.Sprintf("Adding user %q as watcher of issue %q...", uname, ac.params.key))
		defer s.Stop()

		return api.ProxyWatchIssue(client, ac.params.key, u)
	}()
	cmdutil.ExitIfError(err)

	cmdutil.Success("User %q added as watcher of issue %q", uname, ac.params.key)
	fmt.Printf("%s\n", cmdutil.GenerateServerURL(viper.GetString("server"), ac.params.key))
}

type watchParams struct {
	key   string
	user  string
	debug bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *watchParams {
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

	return &watchParams{
		key:   key,
		user:  user,
		debug: debug,
	}
}

type watchCmd struct {
	client *jira.Client
	users  []*jira.User
	params *watchParams
}

func (ac *watchCmd) setIssueKey(project string) error {
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

func (ac *watchCmd) setWatcher(project string) error {
	if ac.params.user != "" && len(ac.users) == 1 {
		ac.params.user = getQueryableName(ac.users[0].Name, ac.users[0].DisplayName)
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
				Message: "User to add to watchers:",
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
		if err := ac.searchAndSetUser(project); err != nil {
			return err
		}
		last = true
	}
	ac.params.user = ans

	return nil
}

func (ac *watchCmd) getOptions(last bool) []string {
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
	options := []string{optionSearch}

	if last {
		options = append(options, validUsers...)
		options = append(options, lineBreak)
	} else {
		options = append(options, lineBreak)
		options = append(options, validUsers...)
	}

	return options
}

func (ac *watchCmd) getSearchKeyword() error {
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

func (ac *watchCmd) searchAndSetUser(project string) error {
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

func (ac *watchCmd) setAvailableUsers(project string) error {
	s := cmdutil.Info("Fetching available users. Please wait...")
	defer s.Stop()

	return ac.searchAndSetUser(project)
}

func (ac *watchCmd) verifyWatcher() (*jira.User, error) {
	watcher := strings.ToLower(ac.params.user)

	var user *jira.User

	for _, u := range ac.users {
		name := strings.ToLower(getQueryableName(u.Name, u.DisplayName))
		if name == watcher || strings.ToLower(u.Email) == watcher {
			user = u
		}
		if strings.ToLower(fmt.Sprintf("%s (%s)", u.DisplayName, u.Name)) == watcher {
			user = u
		}
		if user != nil {
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid watcher %q", ac.params.user)
	}
	if !user.Active {
		return nil, fmt.Errorf("user %q is not active", getQueryableName(user.Name, user.DisplayName))
	}
	return user, nil
}

func getQueryableName(name, displayName string) string {
	if name != "" {
		return name
	}
	return displayName
}
