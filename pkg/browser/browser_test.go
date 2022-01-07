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
			expected: `exec: "firefox": executable file not found in $PATH`,
			teardown: func() {
				os.Clearenv()
			},
		},
		{
			name: "it uses BROWSER env",
			setup: func() {
				_ = os.Setenv("BROWSER", "chrome")
			},
			expected: `exec: "chrome": executable file not found in $PATH`,
			teardown: func() {
				os.Clearenv()
			},
		},
		{
			name: "JIRA_BROWSER gets precedence over BROWSER env if both are set",
			setup: func() {
				_ = os.Setenv("BROWSER", "chrome")
				_ = os.Setenv("JIRA_BROWSER", "firefox")
			},
			expected: `exec: "firefox": executable file not found in $PATH`,
			teardown: func() {
				os.Clearenv()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			err := Browse("https://test.local")
			assert.Error(t, err)
			assert.Equal(t, tc.expected, err.Error())

			tc.teardown()
		})
	}
}
