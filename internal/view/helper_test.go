//nolint:dupl
package view

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFormatDateTime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		format   func() string
		expected string
	}{
		{
			name: "it returns input date for invalid date input",
			format: func() string {
				return formatDateTime("2020-12-03 10:00:00", jira.RFC3339, "UTC")
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it returns input date for invalid input format",
			format: func() string {
				return formatDateTime("2020-12-03 10:00:00", "invalid", "UTC")
			},
			expected: "2020-12-03 10:00:00",
		},
		{
			name: "it format input date from jira date format",
			format: func() string {
				return formatDateTime("2020-12-03T14:05:20.974+0100", jira.RFC3339, "UTC")
			},
			expected: "2020-12-03 13:05:20",
		},
		{
			name: "it format input date from RFC3339 date format",
			format: func() string {
				return formatDateTime("2020-12-13T16:12:00.000Z", time.RFC3339, "UTC")
			},
			expected: "2020-12-13 16:12:00",
		},
		{
			name: "it format input date using proper timezone",
			format: func() string {
				return formatDateTime("2020-12-13T16:12:00.000Z", time.RFC3339, "Asia/Kathmandu")
			},
			expected: "2020-12-13 21:57:00",
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
	t.Parallel()

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
			expected: "[BUG[] This is a bug",
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

func TestShortenAndPad(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{
			name:     "it returns full string for zero limit",
			input:    "Some text",
			limit:    0,
			expected: "Some text",
		},
		{
			name:     "it returns full string if limit is <= 1",
			input:    "Some text",
			limit:    1,
			expected: "Some text",
		},
		{
			name:     "it returns full string if limit is equal to string len",
			input:    "Some text",
			limit:    9,
			expected: "Some text",
		},
		{
			name:     "it returns shortened string",
			input:    "Some text",
			limit:    5,
			expected: "Some…",
		},
		{
			name:     "it adds padding if string is shorter than the limit",
			input:    "Some text",
			limit:    15,
			expected: "Some text      ",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, shortenAndPad(tc.input, tc.limit))
		})
	}
}

func TestMax(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    []int
		expected int
	}{
		{
			name:     "a > b",
			input:    []int{5, 3},
			expected: 5,
		},
		{
			name:     "a < b",
			input:    []int{-5, 5},
			expected: 5,
		},
		{
			name:     "a == b",
			input:    []int{3, 3},
			expected: 3,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, max(tc.input[0], tc.input[1]))
		})
	}
}
