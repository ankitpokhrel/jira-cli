// Package jql is a very simple JQL query builder that cannot do a lot at the moment.
//
// There is no JQL syntax check and relies on the package user to construct a valid query.
//
// It cannot combine AND and OR query currently. That means you cannot construct a query like the one below:
// `project="JQL" AND issue in openSprints() AND (type="Story" OR resolution="Done")`
package jql

import (
	"fmt"
	"strings"
)

// Sort orders.
const (
	DirectionAscending  = "ASC"
	DirectionDescending = "DESC"
)

// GroupFunc groups AND and OR operators.
type GroupFunc func()

// JQL is a jira query language constructor.
type JQL struct {
	project string
	filters []string
	orderBy string
}

// NewJQL initializes jql.
func NewJQL(project string) *JQL {
	jql := JQL{
		project: project,
		filters: []string{fmt.Sprintf("project=\"%s\"", project)},
	}

	return &jql
}

// History search through user issue history.
func (j *JQL) History() *JQL {
	j.filters = append(j.filters, "issue IN issueHistory()")

	return j
}

// Watching search through watched issues.
func (j *JQL) Watching() *JQL {
	j.filters = append(j.filters, "issue IN watchedIssues()")

	return j
}

// FilterBy filters with a given field.
func (j *JQL) FilterBy(field, value string) *JQL {
	if field != "" && value != "" {
		var q string

		if value == "x" {
			q = fmt.Sprintf("%s IS EMPTY", field)
		} else {
			q = fmt.Sprintf("%s=\"%s\"", field, value)
		}

		j.filters = append(j.filters, q)
	}

	return j
}

// Gte is a greater than and equal filter.
func (j *JQL) Gte(field, value string) *JQL {
	if field != "" && value != "" {
		q := fmt.Sprintf("%s>=%s", field, value)

		j.filters = append(j.filters, q)
	}

	return j
}

// In constructs a query with IN clause.
func (j *JQL) In(field string, value ...string) *JQL {
	n := len(value)

	if field != "" && n > 0 {
		var q strings.Builder

		q.WriteString(fmt.Sprintf("%s IN (", field))

		for i, v := range value {
			q.WriteString(fmt.Sprintf("\"%s\"", v))

			if i != n-1 {
				q.WriteString(", ")
			}
		}

		q.WriteString(")")

		j.filters = append(j.filters, q.String())
	}

	return j
}

// OrderBy orders the output in given direction.
func (j *JQL) OrderBy(field, dir string) *JQL {
	j.orderBy = fmt.Sprintf("ORDER BY %s %s", field, dir)

	return j
}

func (j *JQL) mergeFilters(separator string) {
	fLen := len(j.filters)

	var qs strings.Builder

	for i, filter := range j.filters {
		qs.WriteString(filter)

		if i != fLen-1 {
			qs.WriteString(fmt.Sprintf(" %s ", separator))
		}
	}

	s := qs.String()

	if s != "" {
		j.filters = nil
		j.filters = append(j.filters, qs.String())
	}
}

// And combines filter with AND operator.
func (j *JQL) And(fn GroupFunc) *JQL {
	fn()

	j.mergeFilters("AND")

	return j
}

// Or combine filters with OR operator.
func (j *JQL) Or(fn GroupFunc) *JQL {
	fn()

	j.mergeFilters("OR")

	return j
}

// String returns the constructed query.
func (j *JQL) String() string {
	return j.compile()
}

func (j *JQL) compile() string {
	q := strings.Join(j.filters, " ")

	if j.orderBy != "" {
		q += " " + j.orderBy
	}

	return q
}
