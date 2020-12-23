package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	configDir  = ".config/jira"
	configFile = ".jira.yml"
)

var (
	// ErrSkip is returned when a user skips the config generation.
	ErrSkip = fmt.Errorf("skipping config generation")
	// ErrUnexpectedResponseFormat is returned if the response data is in unexpected format.
	ErrUnexpectedResponseFormat = fmt.Errorf("unexpected response format")
)

// JiraCLIConfig is a Jira CLI config.
type JiraCLIConfig struct {
	value struct {
		server  string
		login   string
		project string
		board   *jira.Board
		epic    *jira.Epic
	}
	jiraClient         *jira.Client
	projectSuggestions []string
	boardSuggestions   []string
	boardsMap          map[string]*jira.Board
}

// NewJiraCLIConfig creates a new Jira CLI config.
func NewJiraCLIConfig() *JiraCLIConfig {
	return &JiraCLIConfig{
		boardsMap: make(map[string]*jira.Board),
	}
}

// Generate generates the config file.
func (c *JiraCLIConfig) Generate() error {
	ce := func() bool {
		s := cmdutil.Info("Checking configuration...")
		defer s.Stop()

		return Exists(viper.ConfigFileUsed())
	}()

	if ce && !shallOverwrite() {
		return ErrSkip
	}
	if err := c.configureServerAndLoginDetails(); err != nil {
		return err
	}
	if err := c.configureProjectAndBoardDetails(); err != nil {
		return err
	}
	if err := c.configureMetadata(); err != nil {
		return err
	}

	if err := func() error {
		s := cmdutil.Info("Creating new configuration...")
		defer s.Stop()

		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		return create(fmt.Sprintf("%s/%s/", home, configDir), configFile)
	}(); err != nil {
		return err
	}

	return c.write()
}

func (c *JiraCLIConfig) configureServerAndLoginDetails() error {
	qs := []*survey.Question{
		{
			Name: "server",
			Prompt: &survey.Input{
				Message: "Link to Jira server:",
				Help:    "This is a link to your jira server, eg: https://company.atlassasian.net",
			},
			Validate: func(val interface{}) error {
				errInvalidURL := fmt.Errorf("not a valid URL")

				str, ok := val.(string)
				if !ok {
					return errInvalidURL
				}
				u, err := url.Parse(str)
				if err != nil || u.Scheme == "" || u.Host == "" {
					return errInvalidURL
				}
				if u.Scheme != "http" && u.Scheme != "https" {
					return errInvalidURL
				}

				return nil
			},
		},
		{
			Name: "login",
			Prompt: &survey.Input{
				Message: "Login email:",
				Help:    "This is the email you use to login to your jira account.",
			},
			Validate: func(val interface{}) error {
				var (
					emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9]" +
						"(?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

					errInvalidEmail = fmt.Errorf("not a valid email")
				)

				str, ok := val.(string)
				if !ok {
					return errInvalidEmail
				}
				if len(str) < 3 || len(str) > 254 {
					return errInvalidEmail
				}
				if !emailRegex.MatchString(str) {
					return errInvalidEmail
				}

				return nil
			},
		},
	}

	ans := struct {
		Server string
		Login  string
	}{}

	if err := survey.Ask(qs, &ans); err != nil {
		return err
	}

	return c.verifyLoginDetails(ans.Server, ans.Login)
}

func (c *JiraCLIConfig) verifyLoginDetails(server, login string) error {
	s := cmdutil.Info("Verifying login details...")
	defer s.Stop()

	c.jiraClient = api.Client(jira.Config{
		Server: server,
		Login:  login,
	})
	if _, err := c.jiraClient.Me(); err != nil {
		return err
	}

	c.value.server = server
	c.value.login = login

	return nil
}

func (c *JiraCLIConfig) configureProjectAndBoardDetails() error {
	var project, board string

	if err := c.getProjectSuggestions(); err != nil {
		return err
	}

	projectPrompt := survey.Select{
		Message: "Default project:",
		Help:    "This is your project key that you want to access by default when using the cli.",
		Options: c.projectSuggestions,
	}
	err := survey.AskOne(&projectPrompt, &project, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	if err := c.getBoardSuggestions(project); err != nil {
		return err
	}

	boardPrompt := survey.Select{
		Message: "Default board:",
		Help:    "This is your default project board that you want to access by default when using the cli.",
		Options: c.boardSuggestions,
	}

	err = survey.AskOne(&boardPrompt, &board, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	c.value.project = project
	c.value.board = c.boardsMap[board]

	return nil
}

func (c *JiraCLIConfig) configureMetadata() error {
	s := cmdutil.Info("Fetching metadata...")
	defer s.Stop()

	meta, err := c.jiraClient.GetCreateMeta(&jira.CreateMetaRequest{
		Projects:       c.value.project,
		IssueTypeNames: jira.IssueTypeEpic,
		Expand:         "projects.issuetypes.fields",
	})
	if err != nil {
		return err
	}
	if len(meta.Projects) == 0 || len(meta.Projects[0].IssueTypes) == 0 {
		return ErrUnexpectedResponseFormat
	}

	fields := meta.Projects[0].IssueTypes[0].Fields

	var key string

	for field, value := range fields {
		if !strings.Contains(field, "customfield") {
			continue
		}
		v := value.(map[string]interface{})
		if v["name"].(string) == "Epic Name" {
			key = v["key"].(string)
			break
		}
	}

	c.value.epic = &jira.Epic{Field: key}

	return nil
}

func (c *JiraCLIConfig) write() error {
	viper.Set("server", c.value.server)
	viper.Set("login", c.value.login)
	viper.Set("project", c.value.project)
	viper.Set("board.id", c.value.board.ID)
	viper.Set("board.name", c.value.board.Name)
	viper.Set("board.type", c.value.board.Type)
	viper.Set("epic.field", c.value.epic.Field)

	return viper.WriteConfig()
}

func (c *JiraCLIConfig) getProjectSuggestions() error {
	s := cmdutil.Info("Fetching projects...")
	defer s.Stop()

	projects, err := c.jiraClient.Project()
	if err != nil {
		return err
	}
	for _, project := range projects {
		c.projectSuggestions = append(c.projectSuggestions, project.Key)
	}

	return nil
}

func (c *JiraCLIConfig) getBoardSuggestions(project string) error {
	s := cmdutil.Info(fmt.Sprintf("Fetching boards for project '%s'...", project))
	defer s.Stop()

	resp, err := c.jiraClient.Boards(project, "")
	if err != nil {
		return err
	}
	for _, board := range resp.Boards {
		c.boardsMap[board.Name] = board
		c.boardSuggestions = append(c.boardSuggestions, board.Name)
	}

	return nil
}

// Exists checks if the file exist.
func Exists(file string) bool {
	if file == "" {
		return false
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func shallOverwrite() bool {
	var ans bool

	prompt := &survey.Confirm{
		Message: "Config already exist. Do you want to overwrite?",
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		return false
	}

	return ans
}

func create(path, name string) error {
	if !Exists(path) {
		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}
	}

	file := path + name

	if Exists(file) {
		if err := os.Rename(file, file+".bkp"); err != nil {
			return err
		}
	}

	_, err := os.Create(file)

	return err
}
