package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/oauth"
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
	apiServer    = "https://api.atlassian.com/ex/jira"
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

// JiraCLIMTLSConfig is an authtype specific config.
type JiraCLIMTLSConfig struct {
	CaCert     string
	ClientCert string
	ClientKey  string
}

// JiraCLIConfig is a Jira CLI config.
type JiraCLIConfig struct {
	Installation string
	Server       string
	AuthType     string
	Login        string
	Project      string
	Board        string
	Force        bool
	Insecure     bool
	MTLS         JiraCLIMTLSConfig
}

// JiraCLIConfigGenerator is a Jira CLI config generator.
type JiraCLIConfigGenerator struct {
	usrCfg *JiraCLIConfig
	value  struct {
		installation string
		server       string
		// API server is the server URL for the Jira API. Should be the same as the server URL if not oAuth.
		apiServer string
		version   struct {
			major, minor, patch int
		}
		login        string
		authType     jira.AuthType
		project      *projectConf
		board        *jira.Board
		epic         *jira.Epic
		issueTypes   []*jira.IssueType
		customFields []*issueTypeFieldConf
		mtls         struct {
			caCert, clientCert, clientKey string
		}
		oauth struct {
			accessToken  string
			refreshToken string
			cloudId      string
		}
		timezone string
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
//
//nolint:gocyclo
func (c *JiraCLIConfigGenerator) Generate() (string, error) {
	var cfgFile string

	if cfgFile = viper.ConfigFileUsed(); cfgFile == "" {
		home, err := cmdutil.GetConfigHome()
		if err != nil {
			return "", err
		}
		cfgFile = fmt.Sprintf("%s/%s/%s.%s", home, Dir, FileName, FileType)
	} else {
		isExtValid := func() bool {
			cf := strings.ToLower(cfgFile)
			for _, ext := range []string{FileType, "yaml"} {
				if strings.HasSuffix(cf, fmt.Sprintf(".%s", ext)) {
					return true
				}
			}
			return false
		}
		// Enforce .yml extension.
		if !isExtValid() {
			cfgFile = fmt.Sprintf("%s.%s", cfgFile, FileType)
		}
	}

	cfgExists := func() bool {
		s := cmdutil.Info("Checking configuration...")
		defer s.Stop()

		return Exists(cfgFile)
	}()

	if !c.usrCfg.Force && cfgExists && !shallOverwrite() {
		return "", ErrSkip
	}
	if err := c.configureInstallationType(); err != nil {
		return "", err
	}

	if c.value.installation == jira.InstallationTypeLocal {
		if err := c.configureLocalAuthType(); err != nil {
			return "", err
		}
	}

	if c.value.installation == jira.InstallationTypeCloud {
		// This is to account for OAUTH setup
		if err := c.configureCloudAuthType(); err != nil {
			return "", err
		}
	}
	// Overrides the authType if the authType in the config has been set already
	if c.usrCfg.AuthType != "" {
		c.value.authType = jira.AuthType(c.usrCfg.AuthType)
	}

	if c.value.authType == jira.AuthTypeMTLS {
		if err := c.configureMTLS(); err != nil {
			return "", err
		}
	}

	if err := c.configureLoginDetails(); err != nil {
		return "", err

	}

	if c.value.authType == jira.AuthTypeOAuth {
		if err := c.configureOAuth(); err != nil {
			return "", err
		}
	}

	if err := c.configureServerDetails(); err != nil {
		return "", err
	}

	if c.value.installation == jira.InstallationTypeLocal {
		if err := c.configureServerMeta(); err != nil {
			return "", err
		}
	}
	if err := c.configureProjectAndBoardDetails(); err != nil {
		return "", err
	}
	if err := c.configureMetadata(); err != nil {
		return "", err
	}

	if err := func() error {
		s := cmdutil.Info("Creating new configuration...")
		defer s.Stop()

		return create(cfgFile)
	}(); err != nil {
		return "", err
	}
	return c.write(cfgFile)
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

func (c *JiraCLIConfigGenerator) configureLocalAuthType() error {
	authType := c.usrCfg.AuthType

	if c.usrCfg.AuthType == "" {
		qs := &survey.Select{
			Message: "Authentication type:",
			Help: `Authentication type coud be: basic (login), bearer (PAT) or mtls (client certs)
? If you are using your login credentials, the auth type is probably 'basic' (most common for local installation)
? If you are using a personal access token, the auth type is probably 'bearer'`,
			Options: []string{"basic", "bearer", "mtls"},
			Default: "basic",
		}
		if err := survey.AskOne(qs, &authType); err != nil {
			return err
		}
	}

	switch authType {
	case jira.AuthTypeBearer.String():
		c.value.authType = jira.AuthTypeBearer
	case jira.AuthTypeMTLS.String():
		c.value.authType = jira.AuthTypeMTLS
	default:
		c.value.authType = jira.AuthTypeBasic
	}

	return nil
}

func (c *JiraCLIConfigGenerator) configureCloudAuthType() error {
	authType := c.usrCfg.AuthType

	if c.usrCfg.AuthType == "" {
		qs := &survey.Select{
			Message: "Authentication type:",
			Help: `Authentication type could be: cloud or oauth 
? If you are using your login credentials, the auth type is probably 'cloud' (most common for cloud installation)
? If you are authenticating using oauth 3LO, the auth type is probably 'oauth'`,
			Options: []string{"cloud", "oauth"},
			Default: "cloud",
		}
		if err := survey.AskOne(qs, &authType); err != nil {
			return err
		}
	}

	switch authType {
	case jira.AuthTypeOAuth.String():
		c.value.authType = jira.AuthTypeOAuth
	case jira.AuthTypeCloud.String():
		c.value.authType = jira.AuthTypeCloud
	default:
		c.value.authType = jira.AuthTypeCloud
	}

	return nil
}

func (c *JiraCLIConfigGenerator) configureMTLS() error {
	var qs []*survey.Question

	c.value.mtls.caCert = c.usrCfg.MTLS.CaCert
	c.value.mtls.clientCert = c.usrCfg.MTLS.ClientCert
	c.value.mtls.clientKey = c.usrCfg.MTLS.ClientKey

	getIfEmpty := func(conf, name, msg, help string) {
		if conf != "" {
			return
		}
		qs = append(qs, &survey.Question{
			Name: name,
			Prompt: &survey.Input{
				Message: msg,
				Help:    help,
			},
		})
	}

	getIfEmpty(c.value.mtls.caCert, "cacert", "CA Certificate", "Local path to CA Certificate for your `server`")
	getIfEmpty(c.value.mtls.clientCert, "clientcert", "Client Certificate", "Local path to your client certificate")
	getIfEmpty(c.value.mtls.clientKey, "clientkey", "Client Key", "Local path to your client key")

	if len(qs) > 0 {
		ans := struct {
			CaCert     string
			ClientCert string
			ClientKey  string
		}{}

		if err := survey.Ask(qs, &ans); err != nil {
			return err
		}

		if ans.CaCert != "" {
			c.value.mtls.caCert = ans.CaCert
		}
		if ans.ClientCert != "" {
			c.value.mtls.clientCert = ans.ClientCert
		}
		if ans.ClientKey != "" {
			c.value.mtls.clientKey = ans.ClientKey
		}
	}

	return nil
}

func (c *JiraCLIConfigGenerator) configureOAuth() error {
	// Use the new OAuth package

	tokenResponse, err := oauth.Configure(c.value.login)
	if err != nil {
		return err
	}

	// Store the tokens and cloud ID
	c.value.oauth.accessToken = tokenResponse.AccessToken
	c.value.oauth.refreshToken = tokenResponse.RefreshToken
	c.value.oauth.cloudId = tokenResponse.CloudID

	return nil
}

//nolint:gocyclo
func (c *JiraCLIConfigGenerator) configureLoginDetails() error {
	var qs []*survey.Question

	c.value.login = c.usrCfg.Login
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
					if len(str) < 2 || len(str) > 254 {
						return errInvalidUser
					}

					return nil
				},
			})
		}
	}
	if len(qs) > 0 {
		ans := struct {
			Login string
		}{}

		if err := survey.Ask(qs, &ans); err != nil {
			return err
		}
		if ans.Login != "" {
			c.value.login = strings.TrimSpace(ans.Login)
		}

	}
	return nil
}

//nolint:gocyclo
func (c *JiraCLIConfigGenerator) configureServerDetails() error {
	var qs []*survey.Question

	c.value.server = c.usrCfg.Server

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

	if len(qs) > 0 {
		ans := struct {
			Server string
		}{}

		if err := survey.Ask(qs, &ans); err != nil {
			return err
		}

		if ans.Server != "" {
			c.value.server = strings.TrimSpace(ans.Server)
		}

		if c.value.authType == jira.AuthTypeOAuth {
			// Set server URL using the cloud ID from OAuth configuration
			c.value.apiServer = fmt.Sprintf("%s/%s", apiServer, c.value.oauth.cloudId)
		} else {
			c.value.apiServer = c.value.server
		}
	}
	// Trim trailing slash from server URL
	c.value.server = strings.TrimRight(c.value.server, "/")
	return c.verifyLoginDetails()
}

func (c *JiraCLIConfigGenerator) generateJiraConfig() jira.Config {
	config := jira.Config{
		Server:   c.value.apiServer,
		Login:    c.value.login,
		Insecure: &c.usrCfg.Insecure,
		AuthType: &c.value.authType,
		Debug:    viper.GetBool("debug"),
	}

	switch c.value.authType {
	case jira.AuthTypeOAuth:
		config.APIToken = c.value.oauth.accessToken
	case jira.AuthTypeMTLS:
		config.MTLSConfig = jira.MTLSConfig{
			CaCert:     c.value.mtls.caCert,
			ClientCert: c.value.mtls.clientCert,
			ClientKey:  c.value.mtls.clientKey,
		}
	}
	return config
}

func (c *JiraCLIConfigGenerator) verifyLoginDetails() error {
	s := cmdutil.Info("Verifying login details...")
	defer s.Stop()
	// Configure JIRA client based on auth type
	config := c.generateJiraConfig()
	c.jiraClient = api.Client(config)

	ret, err := c.jiraClient.Me()
	if err != nil {
		return err
	}
	if c.value.authType == jira.AuthTypeBearer {
		c.value.login = ret.Login
	}

	c.value.timezone = ret.Timezone

	return nil
}

func (c *JiraCLIConfigGenerator) configureServerMeta() error {
	s := cmdutil.Info("Fetching server details...")
	defer s.Stop()

	if c.jiraClient != nil {
		config := c.generateJiraConfig()
		c.jiraClient = api.Client(config)
	}
	info, err := c.jiraClient.ServerInfo()
	if err != nil {
		return err
	}

	if len(info.VersionNumbers) == 3 {
		c.value.version.major = info.VersionNumbers[0]
		c.value.version.minor = info.VersionNumbers[1]
		c.value.version.patch = info.VersionNumbers[2]
	}

	return nil
}

//nolint:gocyclo
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
		var suggest string
		if len(defaultBoardSuggestions) > 2 {
			suggest = strings.Join(defaultBoardSuggestions[2:], ", ")
		} else {
			suggest = strings.Join(defaultBoardSuggestions, ", ")
		}
		return fmt.Errorf(
			"board not found\n  Boards available for the project '%s' are '%s'",
			c.value.project.Key,
			suggest,
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
	var err error

	//nolint:mnd
	isV9Compatible := c.value.version.major >= 9 || (c.value.version.major == 8 && c.value.version.minor > 4)
	if c.value.installation == jira.InstallationTypeLocal && isV9Compatible {
		err = c.configureIssueTypesForJiraServerV9()
	} else {
		err = c.configureIssueTypes()
	}
	if err != nil {
		return err
	}

	return c.configureFields()
}

func (c *JiraCLIConfigGenerator) configureIssueTypes() error {
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

	issueTypes := make([]*jira.IssueType, 0, len(meta.Projects[0].IssueTypes))

	for _, it := range meta.Projects[0].IssueTypes {
		issueType := jira.IssueType{
			ID:      it.ID,
			Name:    it.Name,
			Handle:  it.Handle,
			Subtask: it.Subtask,
		}
		issueTypes = append(issueTypes, &issueType)
	}

	c.value.issueTypes = issueTypes

	return nil
}

func (c *JiraCLIConfigGenerator) configureIssueTypesForJiraServerV9() error {
	s := cmdutil.Info("Configuring metadata. Please wait...")
	defer s.Stop()

	meta, err := c.jiraClient.GetCreateMetaForJiraServerV9(&jira.CreateMetaRequest{
		Projects: c.value.project.Key,
		Expand:   "projects.issuetypes.fields",
	})
	if err != nil {
		return err
	}
	if len(meta.Values) == 0 {
		return ErrUnexpectedResponseFormat
	}

	issueTypes := make([]*jira.IssueType, 0, len(meta.Values))

	for _, it := range meta.Values {
		issueType := jira.IssueType{
			ID:      it.ID,
			Name:    it.Name,
			Subtask: it.Subtask,
		}
		issueTypes = append(issueTypes, &issueType)
	}

	c.value.issueTypes = issueTypes

	return nil
}

func (c *JiraCLIConfigGenerator) configureFields() error {
	customFields := make([]*issueTypeFieldConf, 0)

	fields, err := c.jiraClient.GetField()
	if err != nil {
		return err
	}
	var epic jira.Epic

	for _, field := range fields {
		if !field.Custom {
			continue
		}
		if field.Name == jira.EpicFieldName {
			epic.Name = field.ID
			continue
		}
		if field.Name == jira.EpicFieldLink {
			epic.Link = field.ID
			continue
		}
		customFields = append(customFields, &issueTypeFieldConf{
			Name: field.Name,
			Key:  field.ID,
			Schema: struct {
				DataType string `yaml:"datatype"`
				Items    string `yaml:"items,omitempty"`
			}{
				DataType: field.Schema.DataType,
				Items:    field.Schema.Items,
			},
		})
	}

	c.value.epic = &epic
	c.value.customFields = customFields

	return nil
}

func (c *JiraCLIConfigGenerator) write(path string) (string, error) {
	name := func() string {
		ext := filepath.Ext(path)
		if ext == "" {
			return path
		}
		return strings.TrimSuffix(filepath.Base(path), ext)
	}

	config := viper.New()
	config.AddConfigPath(filepath.Dir(path))
	config.SetConfigName(name())
	config.SetConfigType(FileType)

	if c.usrCfg.Insecure {
		config.Set("insecure", c.usrCfg.Insecure)
	}

	config.Set("installation", c.value.installation)
	config.Set("server", c.value.server)
	config.Set("api_server", c.value.apiServer)
	config.Set("login", c.value.login)
	config.Set("project", c.value.project)
	config.Set("epic", c.value.epic)
	config.Set("issue.types", c.value.issueTypes)
	config.Set("issue.fields.custom", c.value.customFields)
	config.Set("auth_type", c.value.authType.String())
	config.Set("timezone", c.value.timezone)

	// MTLS.
	if c.value.mtls.caCert != "" {
		config.Set("mtls.ca_cert", c.value.mtls.caCert)
		config.Set("mtls.client_cert", c.value.mtls.clientCert)
		config.Set("mtls.client_key", c.value.mtls.clientKey)
	}

	// Jira version.
	if c.value.version.major > 0 {
		config.Set("version.major", c.value.version.major)
		config.Set("version.minor", c.value.version.minor)
		config.Set("version.patch", c.value.version.patch)
	}

	if c.value.authType == jira.AuthTypeOAuth {
		config.Set("oauth.cloud_id", c.value.oauth.cloudId)
	}

	if c.value.board != nil {
		config.Set("board", c.value.board)
	} else {
		config.Set("board", "")
	}

	if err := config.WriteConfig(); err != nil {
		return "", err
	}
	return path, nil
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
		if c.value.installation == jira.InstallationTypeCloud {
			return err
		}
		// We don't care about the error in the local instance since board API may not exist if agile-addon is not installed.
		// The only option available for board selection, in this case, is "None" if not passed directly from the flag.
		c.boardSuggestions = append(c.boardSuggestions, optionNone)
		return nil
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

func create(file string) error {
	const perm = 0o700

	path := filepath.Dir(file)
	if !Exists(path) {
		if err := os.MkdirAll(path, perm); err != nil {
			return err
		}
	}

	if Exists(file) {
		if err := os.Rename(file, file+".bkp"); err != nil {
			return err
		}
	}
	f, err := os.Create(file)
	defer func() { _ = f.Close() }()

	return err
}
