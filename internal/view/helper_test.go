//nolint:dupl
package view

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFormatDateTime(t *testing.T) {
	cases := []struct {
		name     string
		format   func() string
		expected string
	}{
		{
			name: "it returns input date for invalid date input",
			format: func() string {
				return formatDateTime("2020-12-03 10:00:00", jira.RFC3339)
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it returns input date for invalid input format",
			format: func() string {
				return formatDateTime("2020-12-03 10:00:00", "invalid")
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it format input date from jira date format",
			format: func() string {
				return formatDateTime("2020-12-03T14:05:20.974+0100", jira.RFC3339)
			},
			expected: "2020-12-03 14:05:20",
		},
		{
			name: "it format input date from RFC3339 date format",
			format: func() string {
				return formatDateTime("2020-12-13T16:12:00.000Z", time.RFC3339)
			},
			expected: "2020-12-13 16:12:00",
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

func TestFormatDateTimeHuman(t *testing.T) {
	cases := []struct {
		name     string
		format   func() string
		expected string
	}{
		{
			name: "it returns input date for invalid date input",
			format: func() string {
				return formatDateTimeHuman("2020-12-03 10:00:00", jira.RFC3339)
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it returns input date for invalid input format",
			format: func() string {
				return formatDateTimeHuman("2020-12-03 10:00:00", "invalid")
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it format input date from jira date format",
			format: func() string {
				return formatDateTimeHuman("2020-12-13T14:05:20.974+0100", jira.RFC3339)
			},
			expected: "Sun, 13 Dec 20",
		},
		{
			name: "it format input date from RFC3339 date format",
			format: func() string {
				return formatDateTimeHuman("2020-12-13T16:12:00.000Z", time.RFC3339)
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

func TestPrepareTitle(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "it returns empty string for empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "it returns same title as input",
			input:    "<title>",
			expected: "<title>",
		},
		{
			name:     "it returns same title as input with special characters",
			input:    "<title> $#!",
			expected: "<title> $#!",
		},
		{
			name:     "it replace big brackets in title",
			input:    "[BUG] This is a bug",
			expected: "⦗BUG⦘ This is a bug",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, prepareTitle(tc.input))
		})
	}
}

func TestGetPager(t *testing.T) {
	t.Parallel()

	pager := os.Getenv("PAGER")

	_ = os.Setenv("PAGER", "")
	assert.Equal(t, "less -r", GetPager())

	_ = os.Setenv("PAGER", "more")
	assert.Equal(t, "more", GetPager())

	_ = os.Setenv("PAGER", pager)
}
