package cmdutil

import (
	"os"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
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
