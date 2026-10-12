package config

import (
	"os"

	"github.com/ankitpokhrel/jira-cli/pkg/netrc"
	"github.com/zalando/go-keyring"
)

// hasStoredToken reports whether an API token is already available via
// the environment, a .netrc file, or the system keyring.
func HasStoredToken(server string, login string) bool {
	if os.Getenv("JIRA_API_TOKEN") != "" {
		return true
	}

	if nc, _ := netrc.Read(server, login); nc != nil {
		return true
	}

	if secret, _ := keyring.Get("jira-cli", login); secret != "" {
		return true
	}

	return false
}
