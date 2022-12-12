package jql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJQL(t *testing.T) {
	cases := []struct {
		name       string
		initialize func() *JQL
		expected   string
	}{
		{
			name: "filter is initialized",
			initialize: func() *JQL {
				return NewJQL("TEST")
			},
			expected: "project=\"TEST\"",
		},
		{
			name: "it sets order by",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.OrderBy("updated", "DESC")
				return jql
			},
			expected: "project=\"TEST\" ORDER BY updated DESC",
		},
		{
			name: "it queries history",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.History()
				return jql
			},
			expected: "project=\"TEST\" issue IN issueHistory()",
		},
		{
			name: "it queries watched issues",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.Watching()
				return jql
			},
			expected: "project=\"TEST\" issue IN watchedIssues()",
		},
		{
			name: "it queries history and watched issues in order",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.Watching().History()
				})
				return jql
			},
			expected: "project=\"TEST\" AND issue IN watchedIssues() AND issue IN issueHistory()",
		},
		{
			name: "it queries with single field filters",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.FilterBy("type", "Story")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\"",
		},
		{
			name: "it queries with multiple field filters",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						FilterBy("assignee", "test@user.com")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND resolution=\"Done\" AND assignee=\"test@user.com\"",
		},
		{
			name: "it queries with not filter",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.FilterBy("type", "~Story").
						FilterBy("assignee", "~x")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type!=\"Story\" AND assignee IS NOT EMPTY",
		},
		{
			name: "it queries for unassigned issues",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						FilterBy("assignee", "x")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND resolution=\"Done\" AND assignee IS EMPTY",
		},
		{
			name: "it queries with function and field filters grouped in AND operator",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.History().
						Watching().
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						FilterBy("assignee", "test@user.com")
				})
				return jql
			},
			expected: "project=\"TEST\" AND issue IN issueHistory() AND issue IN watchedIssues() AND " +
				"type=\"Story\" AND resolution=\"Done\" AND assignee=\"test@user.com\"",
		},
		{
			name: "it queries with function and field filters grouped in OR operator",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.Or(func() {
					jql.History().
						Watching().
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						FilterBy("assignee", "test@user.com")
				})
				return jql
			},
			expected: "project=\"TEST\" OR issue IN issueHistory() OR issue IN watchedIssues() OR " +
				"type=\"Story\" OR resolution=\"Done\" OR assignee=\"test@user.com\"",
		},
		{
			name: "it queries with greater than filter",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						Gt("created", "startOfMonth()", false).
						Gt("updated", "2020-11-11", true)
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND resolution=\"Done\" AND " +
				"created>startOfMonth() AND updated>\"2020-11-11\"",
		},
		{
			name: "it queries with greater than or equals filter",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						Gte("created", "startOfMonth()", false).
						Gte("updated", "2020-11-11", true)
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND resolution=\"Done\" AND " +
				"created>=startOfMonth() AND updated>=\"2020-11-11\"",
		},
		{
			name: "it queries with less than filter",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						Lt("created", "startOfMonth()", false).
						Lt("updated", "2020-11-11", true)
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND resolution=\"Done\" AND " +
				"created<startOfMonth() AND updated<\"2020-11-11\"",
		},
		{
			name: "it queries with IN and a single label",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.FilterBy("type", "Story")
				jql.And(func() {
					jql.In("labels", "first")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND labels IN (\"first\")",
		},
		{
			name: "it queries with IN and multiple labels",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.Or(func() {
					jql.FilterBy("type", "Story").
						In("labels", "first", "second", "third")
				})
				return jql
			},
			expected: "project=\"TEST\" OR type=\"Story\" OR labels IN (\"first\", \"second\", \"third\")",
		},
		{
			name: "it queries with NOT IN and a single label",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.FilterBy("type", "Story")
				jql.And(func() {
					jql.NotIn("labels", "first")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND labels NOT IN (\"first\")",
		},
		{
			name: "it queries with NOT IN and multiple labels",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.Or(func() {
					jql.FilterBy("type", "Story").
						NotIn("labels", "first", "second", "third")
				})
				return jql
			},
			expected: "project=\"TEST\" OR type=\"Story\" OR labels NOT IN (\"first\", \"second\", \"third\")",
		},
		{
			name: "it queries with IN and NOT IN",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.FilterBy("type", "Story")
				jql.And(func() {
					jql.In("labels", "first", "second")
					jql.NotIn("labels", "third", "fourth")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND labels IN (\"first\", \"second\") AND labels NOT IN (\"third\", \"fourth\")",
		},
		{
			name: "it queries with raw jql",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						Raw("summary !~ cli AND priority = high")
				})
				return jql
			},
			expected: "project=\"TEST\" AND type=\"Story\" AND resolution=\"Done\" AND summary !~ cli AND priority = high",
		},
		{
			name: "it queries with raw jql and project filter",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.And(func() {
					jql.
						FilterBy("type", "Story").
						FilterBy("resolution", "Done").
						Raw("summary !~ cli AND project = TEST1")
				})
				return jql
			},
			expected: "type=\"Story\" AND resolution=\"Done\" AND summary !~ cli AND project = TEST1",
		},
		{
			name: "it queries with raw jql and or filter",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.Or(func() {
					jql.FilterBy("type", "Story").
						Raw("summary ~ cli")
				})
				return jql
			},
			expected: "project=\"TEST\" OR type=\"Story\" OR summary ~ cli",
		},
		{
			name: "it queries with raw jql and project filter in or condition",
			initialize: func() *JQL {
				jql := NewJQL("TEST")
				jql.Or(func() {
					jql.FilterBy("type", "Story").
						Raw("summary ~ cli AND project IN (TEST1,TEST2)")
				})
				return jql
			},
			expected: "type=\"Story\" OR summary ~ cli AND project IN (TEST1,TEST2)",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			jql := tc.initialize()
			assert.Equal(t, tc.expected, jql.String())
		})
	}
}

func TestHasProject(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{
			input:    "project=",
			expected: true,
		},
		{
			input:    "project = TEST",
			expected: true,
		},
		{
			input:    "project     =    TEST",
			expected: true,
		},
		{
			input:    "  assigned = abc and PROJECT =   TEST  ",
			expected: true,
		},
		{
			input:    "assigned = abc and project =   TEST and project.property=abc",
			expected: true,
		},
		{
			input:    "PROJECT IS NOT EMPTY AND assignee IN (currentUser())",
			expected: true,
		},
		{
			input:    "PROJECT IN (TEST, TEST1) AND assignee IN (currentUser())",
			expected: true,
		},
		{
			input:    "PROJECT NOT IN (TEST,TEST1) AND assignee IN (currentUser())",
			expected: true,
		},
		{
			input:    "PROJECT != TEST AND projectType=\"classic\" AND assignee IS EMPTY",
			expected: true,
		},
		{
			input:    "project",
			expected: false,
		},
		{
			input:    "projectType",
			expected: false,
		},
		{
			input:    "project.property = ABC",
			expected: false,
		},
		{
			input:    "projectType=\"classic\" AND type=\"Story\" AND assignee IS EMPTY",
			expected: false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run("", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, hasProjectFilter(tc.input))
		})
	}
}
