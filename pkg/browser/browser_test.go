package browser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserENVPrecedence(t *testing.T) {
	cases := []struct {
		name     string
		setup    func()
		expected string
		teardown func()
	}{
		{
			name: "it uses JIRA_BROWSER env",
			setup: func() {
				_ = os.Setenv("JIRA_BROWSER", "firefox")
			},
			expected: "firefox",
			teardown: func() {
				_ = os.Unsetenv("JIRA_BROWSER")
			},
		},
		{
			name: "it uses BROWSER env",
			setup: func() {
				_ = os.Setenv("BROWSER", "chrome")
			},
			expected: "chrome",
			teardown: func() {
				_ = os.Unsetenv("BROWSER")
			},
		},
		{
			name: "JIRA_BROWSER gets precedence over BROWSER env if both are set",
			setup: func() {
				_ = os.Setenv("BROWSER", "chrome")
				_ = os.Setenv("JIRA_BROWSER", "firefox")
			},
			expected: "firefox",
			teardown: func() {
				_ = os.Unsetenv("BROWSER")
				_ = os.Unsetenv("JIRA_BROWSER")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			assert.Equal(t, tc.expected, getBrowserFromENV())
			tc.teardown()
		})
	}
}
