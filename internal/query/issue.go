package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jql"
)

// Issue is a query type for issue command.
type Issue struct {
	Project string
	Flags   FlagParser

	params *IssueParams
}

const defaultLimit = 100

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

func splitPositiveNegative(labels []string) ([]string, []string) {
	positive := make([]string, 0)
	negative := make([]string, 0)
	for _, label := range labels {
		if strings.HasPrefix(label, "~") {
			negative = append(negative, label[1:])
		} else {
			positive = append(positive, label)
		}
	}
	return positive, negative
}

// Get returns constructed jql query.
func (i *Issue) Get() string {
	var q *jql.JQL

	defer func() {
		if i.params.debug {
			fmt.Printf("JQL: %s\n", q.String())
		}
	}()

	q, obf := jql.NewJQL(i.Project), i.params.OrderBy
	if obf == "created" &&
		(i.params.Updated != "" || i.params.UpdatedBefore != "" || i.params.UpdatedAfter != "") &&
		(i.params.Created == "" && i.params.CreatedBefore == "" && i.params.CreatedAfter == "") {
		obf = "updated"
	}

	if i.params.JQL != "" {
		q.Raw(i.params.JQL)
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
			FilterBy("priority", i.params.Priority).
			FilterBy("reporter", i.params.Reporter).
			FilterBy("assignee", i.params.Assignee).
			FilterBy("component", i.params.Component).
			FilterBy("parent", i.params.Parent)

		i.setCreatedFilters(q)
		i.setUpdatedFilters(q)

		positive, negative := splitPositiveNegative(i.params.Labels)
		if len(positive) > 0 {
			q.In("labels", positive...)
		}

		if len(negative) > 0 {
			q.NotIn("labels", negative...)
		}

		positive, negative = splitPositiveNegative(i.params.Status)
		if len(positive) > 0 {
			q.In("status", positive...)
		}

		if len(negative) > 0 {
			q.NotIn("status", negative...)
		}
	})

	if i.params.Reverse {
		q.OrderBy(obf, jql.DirectionAscending)
	} else {
		q.OrderBy(obf, jql.DirectionDescending)
	}

	return q.String()
}

// Params returns issue command params.
func (i *Issue) Params() *IssueParams {
	return i.params
}

func (*Issue) setDateFilters(q *jql.JQL, field, value string) {
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
	Parent        string
	Status        []string
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
	OrderBy       string
	Reverse       bool
	From          uint
	Limit         uint
	JQL           string

	debug bool
}

func (ip *IssueParams) init(flags FlagParser) error {
	var err error

	boolParams := []string{"history", "watching", "reverse", "debug"}
	stringParams := []string{
		"resolution", "type", "parent", "priority", "reporter", "assignee", "component",
		"created", "created-after", "created-before", "updated", "updated-after", "updated-before",
		"jql", "order-by", "paginate",
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

	status, err := flags.GetStringArray("status")
	if err != nil {
		return err
	}

	paginate, err := flags.GetString("paginate")
	if err != nil {
		return err
	}
	from, limit, err := getPaginateParams(paginate)
	if err != nil {
		return err
	}

	ip.setBoolParams(boolParamsMap)
	ip.setStringParams(stringParamsMap)
	ip.Labels = labels
	ip.Status = status
	ip.From = from
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
		case "parent":
			ip.Parent = v
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
		case "jql":
			ip.JQL = v
		case "order-by":
			ip.OrderBy = v
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

func getPaginateParams(paginate string) (uint, uint, error) {
	var (
		err         error
		from, limit int

		errInvalidPaginateArg = fmt.Errorf(
			"invalid argument for paginate: must be a positive integer in format <from>:<limit>, where <from> is optional",
		)
		errOutOfBounds = fmt.Errorf(
			"invalid argument for paginate: Format <from>:<limit>, where <from> is optional and "+
				"<limit> must be between %d and %d (inclusive)", 1, defaultLimit,
		)
	)

	paginate = strings.TrimSpace(paginate)

	if paginate == "" {
		return 0, defaultLimit, nil
	}

	if !strings.Contains(paginate, ":") {
		limit, err = strconv.Atoi(paginate)
		if err != nil {
			return 0, 0, errInvalidPaginateArg
		}
	} else {
		pieces := strings.Split(paginate, ":")
		if len(pieces) != 2 {
			return 0, 0, errInvalidPaginateArg
		}

		from, err = strconv.Atoi(pieces[0])
		if err != nil {
			return 0, 0, errInvalidPaginateArg
		}

		limit, err = strconv.Atoi(pieces[1])
		if err != nil {
			return 0, 0, errInvalidPaginateArg
		}
	}

	if from < 0 || limit <= 0 {
		return 0, 0, errOutOfBounds
	}
	if limit > defaultLimit {
		return 0, 0, errOutOfBounds
	}

	return uint(from), uint(limit), nil
}
