package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	// Dir is a jira-cli config directory.
	Dir = ".jira"
	// FileName is a jira-cli config file name.
	FileName = ".config"
	// FileType is a jira-cli config file extension.
	FileType = "yml"
)

var (
	// ErrSkip is returned when a user skips the config generation.
	ErrSkip = fmt.Errorf("skipping config generation")
	// ErrUnexpectedResponseFormat is returned if the response data is in unexpected format.
	ErrUnexpectedResponseFormat = fmt.Errorf("unexpected response format")
)

// projectConf is a trimmed down version of jira.Project.
type projectConf struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

// JiraCLIConfig is a Jira CLI config.
type JiraCLIConfig struct {
	value struct {
		installation string
		server       string
		login        string
		project      *projectConf
		board        *jira.Board
		epic         *jira.Epic
		issueTypes   []*jira.IssueType
	}
	insecure           bool
	jiraClient         *jira.Client
	projectSuggestions []string
	boardSuggestions   []string
	projectsMap        map[string]*projectConf
	boardsMap          map[string]*jira.Board
}

// JiraCLIConfigFunc decorates option for JiraCLIConfig.
type JiraCLIConfigFunc func(*JiraCLIConfig)

// NewJiraCLIConfig creates a new Jira CLI config.
func NewJiraCLIConfig(opts ...JiraCLIConfigFunc) *JiraCLIConfig {
	cfg := JiraCLIConfig{
		projectsMap: make(map[string]*projectConf),
		boardsMap:   make(map[string]*jira.Board),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &cfg
}

// WithInsecureTLS is a functional opt to set TLS certificate verfication option.
func WithInsecureTLS(ins bool) JiraCLIConfigFunc {
	return func(c *JiraCLIConfig) {
		c.insecure = ins
	}
}

// Generate generates the config file.
func (c *JiraCLIConfig) Generate() (string, error) {
	ce := func() bool {
		s := cmdutil.Info("Checking configuration...")
		defer s.Stop()

		return Exists(viper.ConfigFileUsed())
	}()

	if ce && !shallOverwrite() {
		return "", ErrSkip
	}
	if err := c.configureInstallationType(); err != nil {
		return "", err
	}
	if err := c.configureServerAndLoginDetails(); err != nil {
		return "", err
	}
	if err := c.configureProjectAndBoardDetails(); err != nil {
		return "", err
	}
	if err := c.configureMetadata(); err != nil {
		return "", err
	}

	home, err := cmdutil.GetConfigHome()
	if err != nil {
		return "", err
	}
	cfgDir := fmt.Sprintf("%s/%s", home, Dir)

	if err := func() error {
		s := cmdutil.Info("Creating new configuration...")
		defer s.Stop()

		return create(cfgDir, fmt.Sprintf("%s.%s", FileName, FileType))
	}(); err != nil {
		return "", err
	}

	return c.write(cfgDir)
}

func (c *JiraCLIConfig) configureInstallationType() error {
	qs := &survey.Select{
		Message: "Installation type:",
		Help:    "Is this a cloud installation or an on-premise (local) installation.",
		Options: []string{"Cloud", "Local"},
		Default: "Cloud",
	}

	var installation string
	if err := survey.AskOne(qs, &installation); err != nil {
		return err
	}

	c.value.installation = installation

	return nil
}

func (c *JiraCLIConfig) configureServerAndLoginDetails() error {
	qs := []*survey.Question{
		{
			Name: "server",
			Prompt: &survey.Input{
				Message: "Link to Jira server:",
				Help:    "This is a link to your jira server, eg: https://company.atlassian.net",
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
	}

	if c.value.installation == jira.InstallationTypeCloud {
		qs = append(qs, &survey.Question{
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
		})
	} else if c.value.installation == jira.InstallationTypeLocal {
		qs = append(qs, &survey.Question{
			Name: "login",
			Prompt: &survey.Input{
				Message: "Login username:",
				Help:    "This is the username you use to login to your jira account.",
			},
			Validate: func(val interface{}) error {
				errInvalidUser := fmt.Errorf("not a valid user")

				str, ok := val.(string)
				if !ok {
					return errInvalidUser
				}
				if len(str) < 3 || len(str) > 254 {
					return errInvalidUser
				}

				return nil
			},
		})
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

	server = strings.TrimRight(server, "/")

	c.jiraClient = api.Client(jira.Config{
		Server:   server,
		Login:    login,
		Insecure: c.insecure,
		Debug:    viper.GetBool("debug"),
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
	if err := survey.AskOne(&projectPrompt, &project, survey.WithValidator(survey.Required)); err != nil {
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
	if err := survey.AskOne(&boardPrompt, &board, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	c.value.project = c.projectsMap[project]
	c.value.board = c.boardsMap[board]

	return nil
}

func (c *JiraCLIConfig) configureMetadata() error {
	s := cmdutil.Info("Configuring metadata. Please wait...")
	defer s.Stop()

	meta, err := c.jiraClient.GetCreateMeta(&jira.CreateMetaRequest{
		Projects: c.value.project.Key,
		Expand:   "projects.issuetypes.fields",
	})
	if err != nil {
		return err
	}
	if len(meta.Projects) == 0 || len(meta.Projects[0].IssueTypes) == 0 {
		return ErrUnexpectedResponseFormat
	}

	var (
		epicMeta   map[string]interface{}
		issueTypes = make([]*jira.IssueType, 0, len(meta.Projects[0].IssueTypes))
	)

	for _, it := range meta.Projects[0].IssueTypes {
		if it.Handle == jira.IssueTypeEpic || it.Name == jira.IssueTypeEpic {
			epicMeta = it.Fields
		}
		issueTypes = append(issueTypes, &jira.IssueType{
			ID:      it.ID,
			Name:    it.Name,
			Handle:  it.Handle,
			Subtask: it.Subtask,
		})
	}

	c.value.issueTypes = issueTypes

	epicName, epicLink := c.decipherEpicMeta(epicMeta)
	c.value.epic = &jira.Epic{Name: epicName, Link: epicLink}

	return nil
}

func (c *JiraCLIConfig) decipherEpicMeta(epicMeta map[string]interface{}) (string, string) {
	var (
		temp     string
		epicName string
		epicLink string
	)

	for field, value := range epicMeta {
		if !strings.Contains(field, "customfield") {
			continue
		}
		v := value.(map[string]interface{})

		f := v["name"].(string)
		if f == jira.EpicFieldName || f == jira.EpicFieldLink {
			switch c.value.installation {
			case jira.InstallationTypeCloud:
				temp = v["key"].(string)
			case jira.InstallationTypeLocal:
				if _, ok := v["fieldId"]; ok {
					temp = v["fieldId"].(string)
				} else {
					temp = field
				}
			}

			if f == jira.EpicFieldName {
				epicName = temp
			}
			if f == jira.EpicFieldLink {
				epicLink = temp
			}
		}
	}

	return epicName, epicLink
}

func (c *JiraCLIConfig) write(path string) (string, error) {
	config := viper.New()
	config.AddConfigPath(path)
	config.SetConfigName(FileName)
	config.SetConfigType(FileType)

	if c.insecure {
		config.Set("insecure", c.insecure)
	}

	config.Set("installation", c.value.installation)
	config.Set("server", c.value.server)
	config.Set("login", c.value.login)
	config.Set("project", c.value.project)
	config.Set("epic", c.value.epic)
	config.Set("issue.types", c.value.issueTypes)

	if c.value.board != nil {
		config.Set("board", c.value.board)
	} else {
		config.Set("board", "")
	}

	if err := config.WriteConfig(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s.%s", path, FileName, FileType), nil
}

func (c *JiraCLIConfig) getProjectSuggestions() error {
	s := cmdutil.Info("Fetching projects...")
	defer s.Stop()

	projects, err := c.jiraClient.Project()
	if err != nil {
		return err
	}
	for _, project := range projects {
		c.projectsMap[project.Key] = &projectConf{
			Key:  project.Key,
			Type: project.Type,
		}
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
	c.boardSuggestions = append(c.boardSuggestions, "None")

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
	const perm = 0o700

	if !Exists(path) {
		if err := os.MkdirAll(path, perm); err != nil {
			return err
		}
	}

	file := fmt.Sprintf("%s/%s", path, name)
	if Exists(file) {
		if err := os.Rename(file, file+".bkp"); err != nil {
			return err
		}
	}
	_, err := os.Create(file)

	return err
}
