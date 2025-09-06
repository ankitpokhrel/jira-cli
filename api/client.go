package api

import (
	"time"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter"
	"github.com/ankitpokhrel/jira-cli/pkg/netrc"
	"github.com/ankitpokhrel/jira-cli/pkg/oauth"
)

const clientTimeout = 15 * time.Second

var jiraClient *jira.Client

// getAPIToken retrieves the API token from various sources in order of priority:
// 1. Viper configuration
// 2. OAuth access token (if available and valid)
// 3. Netrc file
// 4. Keyring
func getAPIToken(config *jira.Config) string {
	if config.APIToken != "" {
		return config.APIToken
	}

	// Try viper config first
	if token := viper.GetString("api_token"); token != "" {
		return token
	}

	// Try OAuth access token if available and valid
	if oauthToken := oauth.GetValidAccessToken(); oauthToken != "" {
		return oauthToken
	}

	// Try netrc file
	if netrcConfig, _ := netrc.Read(config.Server, config.Login); netrcConfig != nil {
		if netrcConfig.Password != "" {
			return netrcConfig.Password
		}
	}

	// Try keyring
	if secret, _ := keyring.Get("jira-cli", config.Login); secret != "" {
		return secret
	}

	return ""
}

// Client initializes and returns jira client.
func Client(config jira.Config) *jira.Client {
	if jiraClient != nil {
		return jiraClient
	}

	if config.Server == "" {
		config.Server = viper.GetString("server")
	}
	if config.Login == "" {
		config.Login = viper.GetString("login")
	}
	//TODO: Load auth token here
	if config.AuthType == nil {
		authType := jira.AuthType(viper.GetString("auth_type"))
		config.AuthType = &authType
	}
	if config.Insecure == nil {
		insecure := viper.GetBool("insecure")
		config.Insecure = &insecure
	}

	// Get API token from various sources
	config.APIToken = getAPIToken(&config)

	// If we have an OAuth token, set auth type to OAuth
	if oauthToken := oauth.GetValidAccessToken(); oauthToken != "" && config.APIToken == oauthToken {
		oauthAuthType := jira.AuthTypeOAuth
		config.AuthType = &oauthAuthType
	}

	// MTLS

	if config.MTLSConfig.CaCert == "" {
		config.MTLSConfig.CaCert = viper.GetString("mtls.ca_cert")
	}
	if config.MTLSConfig.ClientCert == "" {
		config.MTLSConfig.ClientCert = viper.GetString("mtls.client_cert")
	}
	if config.MTLSConfig.ClientKey == "" {
		config.MTLSConfig.ClientKey = viper.GetString("mtls.client_key")
	}

	jiraClient = jira.NewClient(
		config,
		jira.WithTimeout(clientTimeout),
		jira.WithInsecureTLS(*config.Insecure),
	)

	return jiraClient
}

// DefaultClient returns default jira client.
func DefaultClient(debug bool) *jira.Client {
	return Client(jira.Config{Debug: debug})
}

// ProxyCreate uses either a v2 or v3 version of the Jira POST /issue
// endpoint to create an issue based on configured installation type.
// Defaults to v3 if installation type is not defined in the config.
func ProxyCreate(c *jira.Client, cr *jira.CreateRequest) (*jira.CreateResponse, error) {
	var (
		resp *jira.CreateResponse
		err  error
	)

	it := viper.GetString("installation")

	if it == jira.InstallationTypeLocal {
		resp, err = c.CreateV2(cr)
	} else {
		resp, err = c.Create(cr)
	}

	return resp, err
}

// ProxyGetIssueRaw executes the same request as ProxyGetIssue but returns raw API response body string.
func ProxyGetIssueRaw(c *jira.Client, key string) (string, error) {
	it := viper.GetString("installation")
	if it == jira.InstallationTypeLocal {
		return c.GetIssueV2Raw(key)
	}
	return c.GetIssueRaw(key)
}

// ProxyGetIssue uses either a v2 or v3 version of the Jira GET /issue/{key}
// endpoint to fetch the issue details based on configured installation type.
// Defaults to v3 if installation type is not defined in the config.
func ProxyGetIssue(c *jira.Client, key string, opts ...filter.Filter) (*jira.Issue, error) {
	var (
		iss *jira.Issue
		err error
	)

	it := viper.GetString("installation")

	if it == jira.InstallationTypeLocal {
		iss, err = c.GetIssueV2(key, opts...)
	} else {
		iss, err = c.GetIssue(key, opts...)
	}

	return iss, err
}

// ProxySearch uses either a v2 or v3 version of the Jira GET /search endpoint
// to search for the relevant issues based on configured installation type.
// Defaults to v3 if installation type is not defined in the config.
func ProxySearch(c *jira.Client, jql string, from, limit uint) (*jira.SearchResult, error) {
	var (
		issues *jira.SearchResult
		err    error
	)

	it := viper.GetString("installation")

	if it == jira.InstallationTypeLocal {
		issues, err = c.SearchV2(jql, from, limit)
	} else {
		issues, err = c.Search(jql, limit)
	}

	return issues, err
}

// ProxyAssignIssue uses either a v2 or v3 version of the PUT /issue/{key}/assignee
// endpoint to assign an issue to the user.
// Defaults to v3 if installation type is not defined in the config.
func ProxyAssignIssue(c *jira.Client, key string, user *jira.User, def string) error {
	it := viper.GetString("installation")
	assignee := def

	if user != nil {
		switch it {
		case jira.InstallationTypeLocal:
			assignee = user.Name
		default:
			assignee = user.AccountID
		}
	}

	if it == jira.InstallationTypeLocal {
		return c.AssignIssueV2(key, assignee)
	}
	return c.AssignIssue(key, assignee)
}

// ProxyUserSearch uses either v2 or v3 version of the GET /user/assignable/search
// endpoint to search for the users assignable to the given issue.
// Defaults to v3 if installation type is not defined in the config.
func ProxyUserSearch(c *jira.Client, opts *jira.UserSearchOptions) ([]*jira.User, error) {
	var (
		users []*jira.User
		err   error
	)

	it := viper.GetString("installation")

	if it == jira.InstallationTypeLocal {
		users, err = c.UserSearchV2(opts)
	} else {
		users, err = c.UserSearch(opts)
	}

	return users, err
}

// ProxyTransitions uses either v2 or v3 version of the GET /issue/{key}/transitions
// endpoint to fetch valid transitions for an issue.
// Defaults to v3 if installation type is not defined in the config.
func ProxyTransitions(c *jira.Client, key string) ([]*jira.Transition, error) {
	var (
		transitions []*jira.Transition
		err         error
	)

	it := viper.GetString("installation")

	if it == jira.InstallationTypeLocal {
		transitions, err = c.TransitionsV2(key)
	} else {
		transitions, err = c.Transitions(key)
	}

	return transitions, err
}

// ProxyWatchIssue uses either a v2 or v3 version of the PUT /issue/{key}/watchers
// endpoint to assign an issue to the user. Defaults to v3 if installation type is
// not defined in the config.
func ProxyWatchIssue(c *jira.Client, key string, user *jira.User) error {
	it := viper.GetString("installation")

	var assignee string

	if user != nil {
		switch it {
		case jira.InstallationTypeLocal:
			assignee = user.Name
		default:
			assignee = user.AccountID
		}
	}

	if it == jira.InstallationTypeLocal {
		return c.WatchIssueV2(key, assignee)
	}
	return c.WatchIssue(key, assignee)
}
