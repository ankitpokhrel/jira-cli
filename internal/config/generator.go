package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
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

	optionSearch = "[Search...]"
	optionBack   = "Go-back"
	optionNone   = "None"
	lineBreak    = "----------"
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

// issueTypeFieldConf is a trimmed down version of jira.IssueTypeField.
type issueTypeFieldConf struct {
	Name   string `yaml:"name"`
	Key    string `yaml:"key"`
	Schema struct {
		DataType string `yaml:"datatype"`
		Items    string `yaml:"items,omitempty"`
	}
}

// JiraCLIConfig is a Jira CLI config.
type JiraCLIConfig struct {
	Installation string
	Server       string
	Login        string
	Project      string
	Board        string
	Force        bool
	Insecure     bool
}

// JiraCLIConfigGenerator is a Jira CLI config generator.
type JiraCLIConfigGenerator struct {
	usrCfg *JiraCLIConfig
	value  struct {
		installation string
		server       string
		login        string
		authType     jira.AuthType
		project      *projectConf
		board        *jira.Board
		epic         *jira.Epic
		issueTypes   []*jira.IssueType
		customFields []*issueTypeFieldConf
	}
	jiraClient         *jira.Client
	projectSuggestions []string
	boardSuggestions   []string
	projectsMap        map[string]*projectConf
	boardsMap          map[string]*jira.Board
}

// NewJiraCLIConfigGenerator creates a new Jira CLI config.
func NewJiraCLIConfigGenerator(cfg *JiraCLIConfig) *JiraCLIConfigGenerator {
	gen := JiraCLIConfigGenerator{
		usrCfg:      cfg,
		projectsMap: make(map[string]*projectConf),
		boardsMap:   make(map[string]*jira.Board),
	}

	return &gen
}

// Generate generates the config file.
func (c *JiraCLIConfigGenerator) Generate() (string, error) {
	ce := func() bool {
		s := cmdutil.Info("Checking configuration...")
		defer s.Stop()

		return Exists(viper.ConfigFileUsed())
	}()

	if !c.usrCfg.Force && ce && !shallOverwrite() {
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

func (c *JiraCLIConfigGenerator) configureInstallationType() error {
	switch c.usrCfg.Installation {
	case strings.ToLower(jira.InstallationTypeCloud):
		c.value.installation = jira.InstallationTypeCloud
	case strings.ToLower(jira.InstallationTypeLocal):
		c.value.installation = jira.InstallationTypeLocal
	default:
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
	}

	return nil
}

func (c *JiraCLIConfigGenerator) configureServerAndLoginDetails() error {
	var qs []*survey.Question

	c.value.server = c.usrCfg.Server
	c.value.login = c.usrCfg.Login

	if c.usrCfg.Server == "" {
		qs = append(qs, &survey.Question{
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
		})
	}

	if c.usrCfg.Login == "" {
		switch c.value.installation {
		case jira.InstallationTypeCloud:
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
		case jira.InstallationTypeLocal:
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
	}

	if len(qs) > 0 {
		ans := struct {
			Server string
			Login  string
		}{}

		if err := survey.Ask(qs, &ans); err != nil {
			return err
		}

		c.value.server = ans.Server
		c.value.login = ans.Login
	}

	return c.verifyLoginDetails(c.value.server, c.value.login)
}

func (c *JiraCLIConfigGenerator) verifyLoginDetails(server, login string) error {
	s := cmdutil.Info("Verifying login details...")
	defer s.Stop()

	server = strings.TrimRight(server, "/")

	c.jiraClient = api.Client(jira.Config{
		Server:   server,
		Login:    login,
		Insecure: c.usrCfg.Insecure,
		AuthType: c.value.authType,
		Debug:    viper.GetBool("debug"),
	})
	if ret, err := c.jiraClient.Me(); err != nil {
		return err
	} else if c.value.authType == jira.AuthTypeBearer {
		login = ret.Login
	}

	c.value.server = server
	c.value.login = login

	return nil
}

func (c *JiraCLIConfigGenerator) configureProjectAndBoardDetails() error {
	project := c.usrCfg.Project
	board := c.usrCfg.Board

	if err := c.getProjectSuggestions(); err != nil {
		return err
	}

	if c.usrCfg.Project == "" {
		projectPrompt := survey.Select{
			Message: "Default project:",
			Help:    "This is your project key that you want to access by default when using the cli.",
			Options: c.projectSuggestions,
		}
		if err := survey.AskOne(&projectPrompt, &project, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}
	c.value.project = c.projectsMap[strings.ToLower(project)]

	if c.value.project == nil {
		return fmt.Errorf("project not found\n  Please check the project key and try again")
	}

	if err := c.getBoardSuggestions(project); err != nil {
		return err
	}
	defaultBoardSuggestions := c.boardSuggestions

	if c.usrCfg.Board == "" {
		for {
			boardPrompt := &survey.Question{
				Name: "",
				Prompt: &survey.Select{
					Message: "Default board:",
					Help:    "This is your default project board that you want to access by default when using the cli.",
					Options: c.boardSuggestions,
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

			if err := survey.Ask([]*survey.Question{boardPrompt}, &board, survey.WithValidator(survey.Required)); err != nil {
				return err
			}
			if board != optionBack && board != optionSearch {
				break
			}
			if board == optionBack {
				c.boardSuggestions = defaultBoardSuggestions
			}
			if board == optionSearch {
				kw, err := c.getSearchKeyword()
				if err != nil {
					return err
				}
				if err := c.searchAndAssignBoard(project, kw); err != nil {
					return err
				}
			}
		}
	}
	c.value.board = c.boardsMap[strings.ToLower(board)]

	if c.value.board == nil && !strings.EqualFold(board, optionNone) {
		return fmt.Errorf(
			"board not found\n  Boards available for the project '%s' are '%s'",
			c.value.project.Key,
			strings.Join(defaultBoardSuggestions[2:], ", "),
		)
	}
	return nil
}

func (*JiraCLIConfigGenerator) getSearchKeyword() (string, error) {
	var ans string

	qs := &survey.Question{
		Name: "board",
		Prompt: &survey.Input{
			Message: "Search board:",
			Help:    "Type board name to search",
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
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return "", err
	}
	return ans, nil
}

func (c *JiraCLIConfigGenerator) searchAndAssignBoard(project, keyword string) error {
	resp, err := c.jiraClient.BoardSearch(project, keyword)
	if err != nil {
		return err
	}

	c.boardSuggestions = []string{}
	for _, board := range resp.Boards {
		c.boardsMap[strings.ToLower(board.Name)] = board
		c.boardSuggestions = append(c.boardSuggestions, board.Name)
	}
	c.boardSuggestions = append(c.boardSuggestions, lineBreak, optionSearch, optionBack)

	return nil
}

func (c *JiraCLIConfigGenerator) configureMetadata() error {
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
		epicMeta     map[string]jira.IssueTypeField
		issueTypes   = make([]*jira.IssueType, 0, len(meta.Projects[0].IssueTypes))
		customFields = make([]*issueTypeFieldConf, 0)
		fieldSeen    = make(map[string]struct{})
	)

	for _, it := range meta.Projects[0].IssueTypes {
		if it.Handle == jira.IssueTypeEpic || it.Name == jira.IssueTypeEpic {
			epicMeta = it.Fields
		}
		issueType := jira.IssueType{
			ID:      it.ID,
			Name:    it.Name,
			Handle:  it.Handle,
			Subtask: it.Subtask,
		}
		for key, field := range it.Fields {
			if strings.HasPrefix(key, "customfield_") {
				if _, ok := fieldSeen[key]; ok {
					continue
				}
				fieldSeen[key] = struct{}{}

				fieldKey := field.Key
				if field.FieldID != "" {
					fieldKey = field.FieldID
				}
				customFields = append(customFields, &issueTypeFieldConf{
					Name: field.Name,
					Key:  fieldKey,
					Schema: struct {
						DataType string `yaml:"datatype"`
						Items    string `yaml:"items,omitempty"`
					}{
						DataType: field.Schema.DataType,
						Items:    field.Schema.Items,
					},
				})
			}
		}
		issueTypes = append(issueTypes, &issueType)
	}

	c.value.issueTypes = issueTypes

	epicName, epicLink := c.decipherEpicMeta(epicMeta)
	c.value.epic = &jira.Epic{Name: epicName, Link: epicLink}
	c.value.customFields = customFields

	return nil
}

func (c *JiraCLIConfigGenerator) decipherEpicMeta(epicMeta map[string]jira.IssueTypeField) (string, string) {
	var (
		temp     string
		epicName string
		epicLink string
	)

	for field, meta := range epicMeta {
		if !strings.Contains(field, "customfield") {
			continue
		}

		if meta.Name == jira.EpicFieldName || meta.Name == jira.EpicFieldLink {
			switch c.value.installation {
			case jira.InstallationTypeCloud:
				temp = meta.Key
			case jira.InstallationTypeLocal:
				if meta.FieldID != "" {
					temp = meta.FieldID
				} else {
					temp = field
				}
			}

			if meta.Name == jira.EpicFieldName {
				epicName = temp
			}
			if meta.Name == jira.EpicFieldLink {
				epicLink = temp
			}
		}
	}

	return epicName, epicLink
}

func (c *JiraCLIConfigGenerator) write(path string) (string, error) {
	config := viper.New()
	config.AddConfigPath(path)
	config.SetConfigName(FileName)
	config.SetConfigType(FileType)

	if c.usrCfg.Insecure {
		config.Set("insecure", c.usrCfg.Insecure)
	}

	config.Set("installation", c.value.installation)
	config.Set("server", c.value.server)
	config.Set("login", c.value.login)
	config.Set("project", c.value.project)
	config.Set("epic", c.value.epic)
	config.Set("issue.types", c.value.issueTypes)
	config.Set("issue.fields.custom", c.value.customFields)

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

func (c *JiraCLIConfigGenerator) getProjectSuggestions() error {
	s := cmdutil.Info("Fetching projects...")
	defer s.Stop()

	projects, err := c.jiraClient.Project()
	if err != nil {
		return err
	}
	for _, project := range projects {
		c.projectsMap[strings.ToLower(project.Key)] = &projectConf{
			Key:  project.Key,
			Type: project.Type,
		}
		c.projectSuggestions = append(c.projectSuggestions, project.Key)
	}

	return nil
}

func (c *JiraCLIConfigGenerator) getBoardSuggestions(project string) error {
	s := cmdutil.Info(fmt.Sprintf("Fetching boards for project '%s'...", project))
	defer s.Stop()

	resp, err := c.jiraClient.Boards(project, "")
	if err != nil {
		return err
	}
	c.boardSuggestions = append(c.boardSuggestions, optionSearch, lineBreak)
	for _, board := range resp.Boards {
		c.boardsMap[strings.ToLower(board.Name)] = board
		c.boardSuggestions = append(c.boardSuggestions, board.Name)
	}
	c.boardSuggestions = append(c.boardSuggestions, optionNone)

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
