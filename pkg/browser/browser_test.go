package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserENVPrecedence(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "it uses JIRA_BROWSER env",
			env:      map[string]string{"JIRA_BROWSER": "firefox"},
			expected: "firefox",
		},
		{
			name:     "it uses BROWSER env",
			env:      map[string]string{"BROWSER": "chrome"},
			expected: "chrome",
		},
		{
			name:     "JIRA_BROWSER gets precedence over BROWSER env if both are set",
			env:      map[string]string{"BROWSER": "chrome", "JIRA_BROWSER": "firefox"},
			expected: "firefox",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JIRA_BROWSER", "")
			t.Setenv("BROWSER", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tc.expected, getBrowserFromENV())
		})
	}
}
