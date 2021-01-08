package query

import (
	"fmt"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jql"
)

// Issue is a query type for issue command.
type Issue struct {
	Project string
	Flags   FlagParser
	params  *IssueParams
}

// NewIssue creates and initializes a new Issue type.
func NewIssue(project string, flags FlagParser) (*Issue, error) {
	ip := IssueParams{}
	if err := ip.init(flags); err != nil {
		return nil, err
	}
	return &Issue{
		Project: project,
		Flags:   flags,
		params:  &ip,
	}, nil
}

// Get returns constructed jql query.
func (i *Issue) Get() string {
	q, obf := jql.NewJQL(i.Project), "created"
	if (i.params.Updated != "" || i.params.UpdatedBefore != "" || i.params.UpdatedAfter != "") &&
		(i.params.Created == "" && i.params.CreatedBefore == "" && i.params.CreatedAfter == "") {
		obf = "updated"
	}
	q.And(func() {
		if i.params.Latest {
			q.History()
			obf = "lastViewed"
		}
		if i.params.Watching {
			q.Watching()
		}

		q.FilterBy("type", i.params.IssueType).
			FilterBy("resolution", i.params.Resolution).
			FilterBy("status", i.params.Status).
			FilterBy("priority", i.params.Priority).
			FilterBy("reporter", i.params.Reporter).
			FilterBy("assignee", i.params.Assignee).
			FilterBy("component", i.params.Component)

		i.setCreatedFilters(q)
		i.setUpdatedFilters(q)

		if len(i.params.Labels) > 0 {
			q.In("labels", i.params.Labels...)
		}
	})
	if i.params.Reverse {
		q.OrderBy(obf, jql.DirectionAscending)
	} else {
		q.OrderBy(obf, jql.DirectionDescending)
	}
	if i.params.debug {
		fmt.Printf("JQL: %s\n", q.String())
	}
	return q.String()
}

// Params returns issue command params.
func (i *Issue) Params() *IssueParams {
	return i.params
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
	if i.params.Created != "" {
		i.setDateFilters(q, "createdDate", i.params.Created)
		return
	}
	if i.params.CreatedAfter != "" {
		q.Gt("createdDate", i.params.CreatedAfter, true)
	}
	if i.params.CreatedBefore != "" {
		q.Lt("createdDate", i.params.CreatedBefore, true)
	}
}

func (i *Issue) setUpdatedFilters(q *jql.JQL) {
	if i.params.Updated != "" {
		i.setDateFilters(q, "updatedDate", i.params.Updated)
		return
	}
	if i.params.UpdatedAfter != "" {
		q.Gt("updatedDate", i.params.UpdatedAfter, true)
	}
	if i.params.UpdatedBefore != "" {
		q.Lt("updatedDate", i.params.UpdatedBefore, true)
	}
}

// IssueParams is issue command parameters.
type IssueParams struct {
	Latest        bool
	Watching      bool
	Resolution    string
	IssueType     string
	Status        string
	Priority      string
	Reporter      string
	Assignee      string
	Component     string
	Created       string
	Updated       string
	CreatedAfter  string
	UpdatedAfter  string
	CreatedBefore string
	UpdatedBefore string
	Labels        []string
	Reverse       bool
	Limit         uint
	debug         bool
}

func (ip *IssueParams) init(flags FlagParser) error {
	var err error

	boolParams := []string{"history", "watching", "reverse", "debug"}
	stringParams := []string{
		"resolution", "type", "status", "priority", "reporter", "assignee", "component",
		"created", "created-after", "created-before", "updated", "updated-after", "updated-before",
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
	limit, err := flags.GetUint("limit")
	if err != nil {
		return err
	}

	ip.setBoolParams(boolParamsMap)
	ip.setStringParams(stringParamsMap)
	ip.Labels = labels
	ip.Limit = limit

	return nil
}

func (ip *IssueParams) setBoolParams(paramsMap map[string]bool) {
	for k, v := range paramsMap {
		switch k {
		case "history":
			ip.Latest = v
		case "watching":
			ip.Watching = v
		case "reverse":
			ip.Reverse = v
		case "debug":
			ip.debug = v
		}
	}
}

func (ip *IssueParams) setStringParams(paramsMap map[string]string) {
	for k, v := range paramsMap {
		switch k {
		case "resolution":
			ip.Resolution = v
		case "type":
			ip.IssueType = v
		case "status":
			ip.Status = v
		case "priority":
			ip.Priority = v
		case "reporter":
			ip.Reporter = v
		case "assignee":
			ip.Assignee = v
		case "component":
			ip.Component = v
		case "created":
			ip.Created = v
		case "created-after":
			ip.CreatedAfter = v
		case "created-before":
			ip.CreatedBefore = v
		case "updated":
			ip.Updated = v
		case "updated-after":
			ip.UpdatedAfter = v
		case "updated-before":
			ip.UpdatedBefore = v
		}
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
