package cmdutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFormatDateTimeHuman(t *testing.T) {
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
