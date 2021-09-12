package api

import (
	"time"

	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const clientTimeout = 15 * time.Second

var jiraClient *jira.Client

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
	if config.APIToken == "" {
		config.APIToken = viper.GetString("api_token")
	}

	jiraClient = jira.NewClient(config, jira.WithTimeout(clientTimeout))

	return jiraClient
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

// ProxyGetIssue uses either a v2 or v3 version of the Jira GET /issue/{key}
// endpoint to fetch the issue details based on configured installation type.
// Defaults to v3 if installation type is not defined in the config.
func ProxyGetIssue(c *jira.Client, key string) (*jira.Issue, error) {
	var (
		issue *jira.Issue
		err   error
	)

	it := viper.GetString("installation")

	if it == jira.InstallationTypeLocal {
		issue, err = c.GetIssueV2(key)
	} else {
		issue, err = c.GetIssue(key)
	}

	return issue, err
}
