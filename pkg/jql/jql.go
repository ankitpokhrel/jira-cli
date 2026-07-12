package jql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// DirectionAscending is an ascending sort order.
	DirectionAscending = "ASC"
	// DirectionDescending is a descending sort order.
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

// NewJQL initializes jql query builder.
func NewJQL(project string) *JQL {
	return &JQL{
		project: project,
		filters: []string{fmt.Sprintf("project=%q", project)},
	}
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
//
// If the value is `x`, it construct the query with IS EMPTY operator, uses equals otherwise.
func (j *JQL) FilterBy(field, value string) *JQL {
	if field != "" && value != "" {
		var q string

		switch {
		case value == "x":
			q = fmt.Sprintf("%s IS EMPTY", field)
		case value[0] == '~':
			value = value[1:]
			if value == "x" {
				q = fmt.Sprintf("%s IS NOT EMPTY", field)
			} else {
				q = fmt.Sprintf("%s!=%q", field, strings.TrimLeft(value, " "))
			}
		default:
			q = fmt.Sprintf("%s=%q", field, value)
		}

		j.filters = append(j.filters, q)
	}
	return j
}

// Gt is a greater than filter.
func (j *JQL) Gt(field, value string, wrap bool) *JQL {
	if field != "" && value != "" {
		var q string

		if wrap {
			q = fmt.Sprintf("%s>%q", field, value)
		} else {
			q = fmt.Sprintf("%s>%s", field, value)
		}

		j.filters = append(j.filters, q)
	}
	return j
}

// Gte is a greater than and equals filter.
func (j *JQL) Gte(field, value string, wrap bool) *JQL {
	if field != "" && value != "" {
		var q string

		if wrap {
			q = fmt.Sprintf("%s>=%q", field, value)
		} else {
			q = fmt.Sprintf("%s>=%s", field, value)
		}

		j.filters = append(j.filters, q)
	}
	return j
}

// Lt is a less than filter.
func (j *JQL) Lt(field, value string, wrap bool) *JQL {
	if field != "" && value != "" {
		var q string

		if wrap {
			q = fmt.Sprintf("%s<%q", field, value)
		} else {
			q = fmt.Sprintf("%s<%s", field, value)
		}

		j.filters = append(j.filters, q)
	}
	return j
}

// In constructs a query with IN clause.
func (j *JQL) In(field string, value ...string) *JQL {
	if field != "" && len(value) > 0 {
		j.filters = append(j.filters, fmt.Sprintf("%s IN (%s)", field, quoteAll(value)))
	}
	return j
}

// NotIn constructs a query with NOT IN clause.
func (j *JQL) NotIn(field string, value ...string) *JQL {
	if field != "" && len(value) > 0 {
		j.filters = append(j.filters, fmt.Sprintf("%s NOT IN (%s)", field, quoteAll(value)))
	}
	return j
}

// OrderBy orders the output in given direction.
func (j *JQL) OrderBy(field, dir string) *JQL {
	j.orderBy = fmt.Sprintf("ORDER BY %s %s", field, dir)
	return j
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

// Raw sets the passed JQL query along with project context.
func (j *JQL) Raw(q string) *JQL {
	q = strings.TrimSpace(q)
	if q == "" {
		return j
	}
	if hasProjectFilter(q) {
		j.filters = j.filters[1:]
	}
	j.filters = append(j.filters, q)
	return j
}

// String returns the constructed query.
func (j *JQL) String() string {
	return j.compile()
}

func (j *JQL) mergeFilters(separator string) {
	s := strings.Join(j.filters, " "+separator+" ")

	if s != "" {
		j.filters = []string{s}
	}
}

func (j *JQL) compile() string {
	q := strings.Join(j.filters, " ")
	if j.orderBy != "" {
		q += " " + j.orderBy
	}

	return q
}

func quoteAll(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return strings.Join(quoted, ", ")
}

func hasProjectFilter(str string) bool {
	regx := "(?i)((project)[\\s]*?={0,1}\\b)[^'.']"
	m, _ := regexp.MatchString(regx, str)
	return m
}
