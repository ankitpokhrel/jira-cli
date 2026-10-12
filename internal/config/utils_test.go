package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

const (
	testServer = "https://example.atlassian.net"
	testLogin  = "user@example.com"
)

// setupNetrc points NETRC at a temp file. The netrc package parses the file once per process,
// so every test sets it up to make sure the first read sees the same entry.
func setupNetrc(t *testing.T) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "netrc")
	content := "machine example.atlassian.net login user@example.com password secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETRC", path)
}

func TestHasStoredToken(t *testing.T) {
	cases := []struct {
		name     string
		server   string
		login    string
		envToken string
		keyring  string
		expected bool
	}{
		{
			name:     "it returns true if env token is set",
			server:   "https://other.example.com",
			login:    "nobody",
			envToken: "token",
			expected: true,
		},
		{
			name:     "it returns true if netrc entry exists",
			server:   testServer,
			login:    testLogin,
			expected: true,
		},
		{
			name:     "it returns true if keyring has a token",
			server:   "https://other.example.com",
			login:    "nobody",
			keyring:  "token",
			expected: true,
		},
		{
			name:     "it returns false if no token is available",
			server:   "https://other.example.com",
			login:    "nobody",
			expected: false,
		},
		{
			name:     "it returns false for an invalid server",
			server:   "",
			login:    "nobody",
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupNetrc(t)
			keyring.MockInit()
			t.Setenv("JIRA_API_TOKEN", tc.envToken)

			if tc.keyring != "" {
				assert.NoError(t, keyring.Set("jira-cli", tc.login, tc.keyring))
			}

			assert.Equal(t, tc.expected, HasStoredToken(tc.server, tc.login))
		})
	}
}

func TestHasStoredTokenKeyringError(t *testing.T) {
	setupNetrc(t)
	keyring.MockInitWithError(assert.AnError)
	t.Setenv("JIRA_API_TOKEN", "")

	assert.False(t, HasStoredToken("https://other.example.com", "nobody"))
}
