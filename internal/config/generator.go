package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	configFile    = ".jira.yml"
	clientTimeout = 15 * time.Second
	refreshRate   = 100 * time.Millisecond
)

// JiraCLIConfig is a Jira CLI config.
type JiraCLIConfig struct {
	value struct {
		server  string
		login   string
		project string
		board   *jira.Board
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
		s := info("Checking configuration...")
		defer s.Stop()

		return exists(viper.ConfigFileUsed())
	}()

	if ce && !shallOverwrite() {
		return nil
	}

	if err := func() error {
		s := info("Creating new configuration...")
		defer s.Stop()

		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		return create(home + "/" + configFile)
	}(); err != nil {
		return err
	}

	if err := c.configureServerAndLoginDetails(); err != nil {
		return err
	}

	if err := c.configureProjectAndBoardDetails(); err != nil {
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
	s := info("Verifying login details...")
	defer s.Stop()

	config := jira.Config{
		Server:   server,
		Login:    login,
		APIToken: viper.GetString("api_token"),
	}

	c.jiraClient = jira.NewClient(config, jira.WithTimeout(clientTimeout))

	if _, err := c.jiraClient.Me(); err != nil {
		fmt.Printf("\nUnable to reach jira server with the given details. Please try again.\n")
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

func (c *JiraCLIConfig) write() error {
	viper.Set("server", c.value.server)
	viper.Set("login", c.value.login)
	viper.Set("project", c.value.project)
	viper.Set("board.id", c.value.board.ID)
	viper.Set("board.name", c.value.board.Name)
	viper.Set("board.type", c.value.board.Type)

	return viper.WriteConfig()
}

func (c *JiraCLIConfig) getProjectSuggestions() error {
	s := info("Fetching projects...")
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
	s := info(fmt.Sprintf("Fetching boards for project '%s'...", project))
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

func exists(file string) bool {
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

func create(path string) error {
	if exists(path) {
		if err := os.Rename(path, path+".bkp"); err != nil {
			return err
		}
	}

	_, err := os.Create(path)

	return err
}

func info(msg string) *spinner.Spinner {
	s := spinner.New(
		spinner.CharSets[14],
		refreshRate,
		spinner.WithSuffix(" "+msg),
		spinner.WithHiddenCursor(true),
	)

	s.Start()

	return s
}
