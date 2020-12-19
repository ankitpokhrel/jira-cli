package api

import (
	"time"

	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const clientTimeout = 15 * time.Second

var jiraClient *jira.Client

// Client initializes and returns jira client.
func Client(debug bool) *jira.Client {
	if jiraClient != nil {
		return jiraClient
	}

	config := jira.Config{
		Server:   viper.GetString("server"),
		Login:    viper.GetString("login"),
		APIToken: viper.GetString("api_token"),
		Debug:    debug,
	}

	jiraClient = jira.NewClient(config, jira.WithTimeout(clientTimeout))

	return jiraClient
}
