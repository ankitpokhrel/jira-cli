package query

import (
	"fmt"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jql"
)

type issueParams struct {
	latest        bool
	watching      bool
	resolution    string
	issueType     string
	status        string
	priority      string
	reporter      string
	assignee      string
	created       string
	updated       string
	createdAfter  string
	updatedAfter  string
	createdBefore string
	updatedBefore string
	labels        []string
	reverse       bool
	debug         bool
}

func (ip *issueParams) init(flags FlagParser) error {
	var err error

	boolParams := []string{"history", "watching", "reverse", "debug"}
	stringParams := []string{
		"resolution", "type", "status", "priority", "reporter", "assignee",
		"created", "created-after", "created-before",
		"updated", "updated-after", "updated-before",
	}

	boolParamsMap := make(map[string]bool)
	for _, param := range boolParams {
		boolParamsMap[param], err = flags.GetBool(param)
		if err != nil {
			return err
		}
	}

	stringParamsMap := make(map[string]string)
	for _, param := range stringParams {
		stringParamsMap[param], err = flags.GetString(param)
		if err != nil {
			return err
		}
	}

	labels, err := flags.GetStringArray("label")
	if err != nil {
		return err
	}

	ip.labels = labels
	ip.setBoolParams(boolParamsMap)
	ip.setStringParams(stringParamsMap)

	return nil
}

func (ip *issueParams) setBoolParams(paramsMap map[string]bool) {
	for k, v := range paramsMap {
		switch k {
		case "history":
			ip.latest = v
		case "watching":
			ip.watching = v
		case "reverse":
			ip.reverse = v
		case "debug":
			ip.debug = v
		}
	}
}

func (ip *issueParams) setStringParams(paramsMap map[string]string) {
	for k, v := range paramsMap {
		switch k {
		case "resolution":
			ip.resolution = v
		case "type":
			ip.issueType = v
		case "status":
			ip.status = v
		case "priority":
			ip.priority = v
		case "reporter":
			ip.reporter = v
		case "assignee":
			ip.assignee = v
		case "created":
			ip.created = v
		case "created-after":
			ip.createdAfter = v
		case "created-before":
			ip.createdBefore = v
		case "updated":
			ip.updated = v
		case "updated-after":
			ip.updatedAfter = v
		case "updated-before":
			ip.updatedBefore = v
		}
	}
}

// Issue is a query type for issue command.
type Issue struct {
	Project string
	Flags   FlagParser
	params  *issueParams
}

// NewIssue creates and initialize new issue type.
func NewIssue(project string, flags FlagParser) (*Issue, error) {
	issue := Issue{
		Project: project,
		Flags:   flags,
	}

	ip := issueParams{}

	err := ip.init(flags)
	if err != nil {
		return nil, err
	}

	issue.params = &ip

	return &issue, nil
}

// Get returns constructed jql query.
func (i *Issue) Get() string {
	obf := "created"

	q := jql.NewJQL(i.Project)

	q.And(func() {
		if i.params.latest {
			q.History()
			obf = "lastViewed"
		}

		if i.params.watching {
			q.Watching()
		}

		q.FilterBy("type", i.params.issueType).
			FilterBy("resolution", i.params.resolution).
			FilterBy("status", i.params.status).
			FilterBy("priority", i.params.priority).
			FilterBy("reporter", i.params.reporter).
			FilterBy("assignee", i.params.assignee)

		i.setCreatedFilters(q)
		i.setUpdatedFilters(q)

		if len(i.params.labels) > 0 {
			q.In("labels", i.params.labels...)
		}
	})

	if i.params.reverse {
		q.OrderBy(obf, jql.DirectionAscending)
	} else {
		q.OrderBy(obf, jql.DirectionDescending)
	}

	if i.params.debug {
		fmt.Printf("JQL: %s\n", q.String())
	}

	return q.String()
}

func (i *Issue) setDateFilters(q *jql.JQL, field, value string) {
	switch value {
	case "today":
		q.Gte(field, "startOfDay()", false)
	case "week":
		q.Gte(field, "startOfWeek()", false)
	case "month":
		q.Gte(field, "startOfMonth()", false)
	case "year":
		q.Gte(field, "startOfYear()", false)
	default:
		q.Gte(field, value, true)

		dt, format, ok := isValidDate(value)
		if ok {
			q.Lt(field, addDay(dt, format), true)
		}
	}
}

func (i *Issue) setCreatedFilters(q *jql.JQL) {
	if i.params.created != "" {
		i.setDateFilters(q, "createdDate", i.params.created)

		return
	}

	if i.params.createdAfter != "" {
		q.Gt("createdDate", i.params.createdAfter, true)
	}

	if i.params.createdBefore != "" {
		q.Lt("createdDate", i.params.createdBefore, true)
	}
}

func (i *Issue) setUpdatedFilters(q *jql.JQL) {
	if i.params.updated != "" {
		i.setDateFilters(q, "updatedDate", i.params.updated)

		return
	}

	if i.params.updatedAfter != "" {
		q.Gt("updatedDate", i.params.updatedAfter, true)
	}

	if i.params.updatedBefore != "" {
		q.Lt("updatedDate", i.params.updatedBefore, true)
	}
}

func isValidDate(date string) (time.Time, string, bool) {
	supportedFormats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 03:04",
		"2006/01/02 03:04",
	}

	for _, format := range supportedFormats {
		dt, err := time.Parse(format, date)
		if err == nil {
			return dt, format, true
		}
	}

	return time.Now(), "", false
}

func addDay(dt time.Time, format string) string {
	return dt.AddDate(0, 0, 1).Format(format)
}
