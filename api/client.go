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
