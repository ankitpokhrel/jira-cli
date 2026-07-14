package jql

import (
	"fmt"
	"strings"
	"unicode"
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
	j := &JQL{project: project}
	if project != "" {
		j.filters = append(j.filters, fmt.Sprintf("project=%q", project))
	}
	return j
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
	n := len(value)
	if field != "" && n > 0 {
		var q strings.Builder

		q.WriteString(fmt.Sprintf("%s IN (", field))
		for i, v := range value {
			q.WriteString(fmt.Sprintf("%q", v))
			if i != n-1 {
				q.WriteString(", ")
			}
		}
		q.WriteString(")")

		j.filters = append(j.filters, q.String())
	}
	return j
}

// NotIn constructs a query with NOT IN clause.
func (j *JQL) NotIn(field string, value ...string) *JQL {
	n := len(value)
	if field != "" && n > 0 {
		var q strings.Builder

		q.WriteString(fmt.Sprintf("%s NOT IN (", field))
		for i, v := range value {
			q.WriteString(fmt.Sprintf("%q", v))
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
	if j.project != "" && hasProjectFilter(q) {
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

func (j *JQL) compile() string {
	q := strings.Join(j.filters, " ")
	if j.orderBy != "" {
		if q != "" {
			q += " "
		}
		q += j.orderBy
	}

	return q
}

type tokenKind int

const (
	tokenWord   tokenKind = iota // an identifier or keyword, e.g. project, IN, EMPTY
	tokenString                  // a quoted string literal
	tokenOp                      // an operator, e.g. =, !=, ~, <=
	tokenPunct                   // any other single character, e.g. ( ) ,
)

type token struct {
	kind  tokenKind
	value string
}

// hasProjectFilter reports whether the query already contains a `project` field
// clause, e.g. `project = X`, `project IN (...)` or `project IS NOT EMPTY`.
func hasProjectFilter(str string) bool {
	tokens := tokenize(str)
	for i, t := range tokens {
		if t.kind == tokenWord && strings.EqualFold(t.value, "project") && isProjectOperator(tokens[i+1:]) {
			return true
		}
	}
	return false
}

// isProjectOperator reports whether the tokens immediately following a `project`
// field form a supported project operator: =, !=, IN, NOT IN or IS [NOT] EMPTY.
func isProjectOperator(tokens []token) bool {
	if len(tokens) == 0 {
		return false
	}
	first := tokens[0]

	if first.kind == tokenOp {
		return first.value == "=" || first.value == "!="
	}
	if first.kind != tokenWord {
		return false
	}

	switch {
	case strings.EqualFold(first.value, "in"):
		return true
	case strings.EqualFold(first.value, "not"):
		return len(tokens) > 1 && tokens[1].kind == tokenWord && strings.EqualFold(tokens[1].value, "in")
	case strings.EqualFold(first.value, "is"):
		rest := tokens[1:]
		if len(rest) > 0 && rest[0].kind == tokenWord && strings.EqualFold(rest[0].value, "not") {
			rest = rest[1:]
		}
		return len(rest) > 0 && rest[0].kind == tokenWord && strings.EqualFold(rest[0].value, "empty")
	}
	return false
}

// tokenize splits a JQL string into tokens, consuming quoted string literals.
func tokenize(str string) []token {
	var tokens []token

	runes := []rune(str)
	n := len(runes)

	for i := 0; i < n; {
		c := runes[i]

		switch {
		case unicode.IsSpace(c):
			i++
		case c == '"' || c == '\'':
			var sb strings.Builder
			i++ // opening quote
			for i < n {
				if runes[i] == '\\' && i+1 < n {
					sb.WriteRune(runes[i+1])
					i += 2
					continue
				}
				if runes[i] == c {
					i++ // closing quote
					break
				}
				sb.WriteRune(runes[i])
				i++
			}
			tokens = append(tokens, token{kind: tokenString, value: sb.String()})
		case isWordRune(c):
			start := i
			for i < n && isWordRune(runes[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokenWord, value: string(runes[start:i])})
		case isOperatorRune(c):
			start := i
			for i < n && isOperatorRune(runes[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokenOp, value: string(runes[start:i])})
		default:
			tokens = append(tokens, token{kind: tokenPunct, value: string(c)})
			i++
		}
	}

	return tokens
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}

func isOperatorRune(r rune) bool {
	switch r {
	case '=', '!', '~', '<', '>':
		return true
	default:
		return false
	}
}
