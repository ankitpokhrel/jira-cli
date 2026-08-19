package cmdutil

import (
	"os"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFormatDateTimeHuman(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		format   func() string
		expected string
	}{
		{
			name: "it returns input date for invalid date input",
			format: func() string {
				return FormatDateTimeHuman("2020-12-03 10:00:00", jira.RFC3339)
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it returns input date for invalid input format",
			format: func() string {
				return FormatDateTimeHuman("2020-12-03 10:00:00", "invalid")
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it format input date from jira date format",
			format: func() string {
				return FormatDateTimeHuman("2020-12-13T14:05:20.974+0100", jira.RFC3339)
			},
			expected: "Sun, 13 Dec 20",
		},
		{
			name: "it format input date from RFC3339 date format",
			format: func() string {
				return FormatDateTimeHuman("2020-12-13T16:12:00.000Z", time.RFC3339)
			},
			expected: "Sun, 13 Dec 20",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.format())
		})
	}
}

func TestGetConfigHome(t *testing.T) {
	t.Parallel()

	userHome, err := homedir.Dir()
	assert.NoError(t, err)

	os.Clearenv()

	configHome, err := GetConfigHome()
	assert.NoError(t, err)
	assert.Equal(t, userHome+"/.config", configHome)

	assert.NoError(t, os.Setenv("XDG_CONFIG_HOME", "./test"))

	configHome, err = GetConfigHome()
	assert.NoError(t, err)
	assert.Equal(t, "./test", configHome)
}

func TestGetJiraIssueKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		project  string
		input    string
		expected string
	}{
		{
			name:     "full key on same project",
			project:  "ANK",
			input:    "ANK-11",
			expected: "ANK-11",
		},
		{
			name:     "full key on different project",
			project:  "POK",
			input:    "ANK-11",
			expected: "ANK-11",
		},
		{
			name:     "key number only",
			project:  "ANK",
			input:    "11",
			expected: "ANK-11",
		},
		{
			name:     "text only key",
			project:  "POK",
			input:    "ANK",
			expected: "ANK",
		},
		{
			name:     "invalid key format",
			project:  "POK",
			input:    "ANK-",
			expected: "ANK-",
		},
		{
			name:     "empty project and numeric key",
			project:  "",
			input:    "11",
			expected: "11",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, GetJiraIssueKey(tc.project, tc.input))
		})
	}
}

func TestNormalizeJiraError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Received an error with message",
			input:    "Error:\n- The request reported 404.",
			expected: "The request reported 404.",
		},
		{
			name:     "no error",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, NormalizeJiraError(tc.input))
		})
	}
}

func TestGetSubtaskHandle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     []*jira.IssueType
		inputType string
		expected  string
	}{
		{
			name: "should get default issue type handle for sub-task",
			input: []*jira.IssueType{
				{
					ID:      "123",
					Name:    "Story",
					Handle:  "story",
					Subtask: false,
				},
			},
			inputType: "Sub-task",
			expected:  "Sub-task",
		},
		{
			name: "should get valid sub-task handle",
			input: []*jira.IssueType{
				{
					ID:      "123",
					Name:    "Story",
					Handle:  "story",
					Subtask: false,
				},
				{
					ID:      "234",
					Name:    "Sub-Task",
					Handle:  "Sub-Task",
					Subtask: true,
				},
			},
			inputType: "Sub-task",
			expected:  "Sub-Task",
		},
		{
			name: "should get sub-task name as handle",
			input: []*jira.IssueType{
				{
					ID:      "123",
					Name:    "Story",
					Handle:  "story",
					Subtask: false,
				},
				{
					ID:      "234",
					Name:    "Subtask",
					Subtask: true,
				},
			},
			inputType: "Sub-task",
			expected:  "Subtask",
		},
		{
			name: "exact matches for a custom sub-task should take precedence",
			input: []*jira.IssueType{
				{
					ID:      "123",
					Name:    "Story",
					Handle:  "story",
					Subtask: false,
				},
				{
					ID:      "234",
					Name:    "Sub-Task",
					Handle:  "Sub-Task",
					Subtask: true,
				},
				{
					ID:      "567",
					Name:    "Custom Sub-Task",
					Handle:  "Custom Sub-Task",
					Subtask: true,
				},
			},
			inputType: "Custom Sub-Task",
			expected:  "Custom Sub-Task",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, GetSubtaskHandle(tc.inputType, tc.input))
		})
	}
}

func TestIsNoInputMode(t *testing.T) {
	cases := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name: "returns false when no_input is not set",
			setup: func() {
				viper.Reset()
			},
			expected: false,
		},
		{
			name: "returns true when no_input is set to true",
			setup: func() {
				viper.Reset()
				viper.Set("no_input", true)
			},
			expected: true,
		},
		{
			name: "returns false when no_input is set to false",
			setup: func() {
				viper.Reset()
				viper.Set("no_input", false)
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			assert.Equal(t, tc.expected, IsNoInputMode())
		})
	}
}

func TestShouldPrompt(t *testing.T) {
	cases := []struct {
		name          string
		localNoInput  bool
		globalNoInput bool
		expected      bool
	}{
		{
			name:          "should prompt when both local and global are false",
			localNoInput:  false,
			globalNoInput: false,
			expected:      true,
		},
		{
			name:          "should not prompt when local is true and global is false",
			localNoInput:  true,
			globalNoInput: false,
			expected:      false,
		},
		{
			name:          "should not prompt when local is false and global is true",
			localNoInput:  false,
			globalNoInput: true,
			expected:      false,
		},
		{
			name:          "should not prompt when both local and global are true",
			localNoInput:  true,
			globalNoInput: true,
			expected:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			if tc.globalNoInput {
				viper.Set("no_input", true)
			}

			assert.Equal(t, tc.expected, ShouldPrompt(tc.localNoInput))
		})
	}
}

func TestNoInputConfigurationMethods(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(t *testing.T)
		expected bool
	}{
		{
			name: "reads no_input from viper.Set (config file simulation)",
			setup: func(t *testing.T) {
				viper.Reset()
				viper.Set("no_input", true)
			},
			expected: true,
		},
		{
			name: "reads no_input from environment variable",
			setup: func(t *testing.T) {
				viper.Reset()
				viper.AutomaticEnv()
				viper.SetEnvPrefix("jira")
				t.Setenv("JIRA_NO_INPUT", "true")
			},
			expected: true,
		},
		{
			name: "environment variable false is respected",
			setup: func(t *testing.T) {
				viper.Reset()
				viper.AutomaticEnv()
				viper.SetEnvPrefix("jira")
				t.Setenv("JIRA_NO_INPUT", "false")
			},
			expected: false,
		},
		{
			name: "prefers viper.Set over unset environment variable",
			setup: func(t *testing.T) {
				viper.Reset()
				viper.AutomaticEnv()
				viper.SetEnvPrefix("jira")
				t.Setenv("JIRA_NO_INPUT", "")
				viper.Set("no_input", true)
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			result := IsNoInputMode()
			assert.Equal(t, tc.expected, result)
		})
	}
}
