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
