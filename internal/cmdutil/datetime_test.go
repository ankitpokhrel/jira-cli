package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDateStringToJiraFormatInLocation(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		timezone string
		expected string
		err      string
	}{
		{
			name:     "it returns empty for zero value",
			input:    "0000-00-00 00:00:00",
			expected: "",
		},
		{
			name:     "it returns the input for valid jira datetime format",
			input:    "2022-01-01T09:30:00.000+0200",
			expected: "2022-01-01T09:30:00.000+0200",
		},
		{
			name:     "it returns valid jira datetime format for date string",
			input:    "2022-01-02",
			expected: "2022-01-02T00:00:00.000+0000",
		},
		{
			name:     "it returns valid jira datetime format for datetime string",
			input:    "2022-01-02 10:10:05",
			expected: "2022-01-02T10:10:05.000+0000",
		},
		{
			name:     "it can handle timezone",
			input:    "2022-01-02 15:04:05",
			timezone: "Asia/Kathmandu",
			expected: "2022-01-02T15:04:05.000+0545",
		},
		{
			name:     "it can detect central european time",
			input:    "2022-01-02 10:10:05",
			timezone: "Europe/Berlin",
			expected: "2022-01-02T10:10:05.000+0100",
		},
		{
			name:     "it can detect central european summer time",
			input:    "2022-09-01 10:10:05",
			timezone: "Europe/Berlin",
			expected: "2022-09-01T10:10:05.000+0200",
		},
		{
			name:     "it can handle short date format",
			input:    "20220102",
			expected: "2022-01-02T00:00:00.000+0000",
		},
		{
			name:     "it can handle short datetime format",
			input:    "20220102101005",
			timezone: "Asia/Bangkok",
			expected: "2022-01-02T10:10:05.000+0700",
		},
		{
			name:     "it returns error for input in invalid format",
			input:    "2022-01-02T15:04:05",
			expected: "",
			err:      "datetime string should be in a valid format, eg: 2022-01-02 10:10:05 or 2022-01-02",
		},
		{
			name:     "it returns error for completely invalid format",
			input:    "invalid",
			expected: "",
			err:      "datetime string should be in a valid format, eg: 2022-01-02 10:10:05 or 2022-01-02",
		},
		{
			name:     "it returns error for invalid timezone",
			input:    "2022-01-02 10:10:05",
			timezone: "invalid",
			expected: "",
			err:      "timezone should be a valid IANA timezone, eg: Asia/Kathmandu or Europe/Berlin",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dt, err := DateStringToJiraFormatInLocation(tc.input, tc.timezone)
			if tc.err != "" {
				assert.NotNil(t, err)
				assert.Equal(t, tc.err, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.expected, dt)
		})
	}
}
