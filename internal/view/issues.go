package view

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	colPadding  = 1
	maxColWidth = 60
)

// DisplayFormat is a issue display type.
type DisplayFormat struct {
	Plain      bool
	NoHeaders  bool
	NoTruncate bool
	Columns    []string
}

// IssueList is a list view for issues.
type IssueList struct {
	Total      int
	Project    string
	Server     string
	Data       []*jira.Issue
	Display    DisplayFormat
	FooterText string
}

// Render renders the view.
func (l *IssueList) Render() error {
	if l.Display.Plain {
		w := tabwriter.NewWriter(os.Stdout, 0, tabWidth, 1, '\t', 0)
		return l.renderPlain(w)
	}

	renderer, err := MDRenderer()
	if err != nil {
		return err
	}

	data := l.data()
	if l.FooterText == "" {
		l.FooterText = fmt.Sprintf("Showing %d of %d results for project \"%s\"", len(data)-1, l.Total, l.Project)
	}

	view := tui.NewTable(
		tui.WithColPadding(colPadding),
		tui.WithMaxColWidth(maxColWidth),
		tui.WithTableFooterText(l.FooterText),
		tui.WithSelectedFunc(navigate(l.Server)),
		tui.WithViewModeFunc(func(r, c int, _ interface{}) (func() interface{}, func(interface{}) (string, error)) {
			dataFn := func() interface{} {
				ci := getKeyColumnIndex(data[0])
				iss, _ := api.ProxyGetIssue(api.Client(jira.Config{}), data[r][ci], issue.NewNumCommentsFilter(1))
				return iss
			}
			renderFn := func(i interface{}) (string, error) {
				iss := Issue{
					Server:  l.Server,
					Data:    i.(*jira.Issue),
					Options: IssueOption{NumComments: 1},
				}
				return iss.RenderedOut(renderer)
			}
			return dataFn, renderFn
		}),
		tui.WithCopyFunc(copyURL(l.Server)),
		tui.WithCopyKeyFunc(copyKey()),
	)

	return view.Paint(data)
}

// renderPlain renders the issue in plain view.
func (l *IssueList) renderPlain(w io.Writer) error {
	return renderPlain(w, l.data())
}

func (*IssueList) validColumnsMap() map[string]struct{} {
	columns := ValidIssueColumns()
	out := make(map[string]struct{}, len(columns))

	for _, c := range columns {
		out[c] = struct{}{}
	}

	return out
}

func (l *IssueList) header() []string {
	if len(l.Display.Columns) == 0 {
		validColumns := ValidIssueColumns()
		if l.Display.NoTruncate || !l.Display.Plain {
			return validColumns
		}
		return validColumns[0:4]
	}

	var (
		headers   []string
		hasKeyCol bool
	)

	columnsMap := l.validColumnsMap()
	for _, c := range l.Display.Columns {
		c = strings.ToUpper(c)
		if _, ok := columnsMap[c]; ok {
			headers = append(headers, strings.ToUpper(c))
		}
		if c == fieldKey {
			hasKeyCol = true
		}
	}

	// Key field is required in TUI to fetch relevant data later.
	// So, we will prepend the field if it is not available.
	if !hasKeyCol {
		headers = append([]string{fieldKey}, headers...)
	}

	return headers
}

func (l *IssueList) data() tui.TableData {
	var data tui.TableData

	headers := l.header()
	if !(l.Display.Plain && l.Display.NoHeaders) {
		data = append(data, headers)
	}
	if len(headers) == 0 {
		headers = ValidIssueColumns()
	}
	for _, iss := range l.Data {
		data = append(data, l.assignColumns(headers, iss))
	}

	return data
}

func (IssueList) assignColumns(columns []string, issue *jira.Issue) []string {
	var bucket []string

	for _, column := range columns {
		switch column {
		case fieldType:
			bucket = append(bucket, issue.Fields.IssueType.Name)
		case fieldKey:
			bucket = append(bucket, issue.Key)
		case fieldSummary:
			bucket = append(bucket, prepareTitle(issue.Fields.Summary))
		case fieldStatus:
			bucket = append(bucket, issue.Fields.Status.Name)
		case fieldAssignee:
			bucket = append(bucket, issue.Fields.Assignee.Name)
		case fieldReporter:
			bucket = append(bucket, issue.Fields.Reporter.Name)
		case fieldPriority:
			bucket = append(bucket, issue.Fields.Priority.Name)
		case fieldResolution:
			bucket = append(bucket, issue.Fields.Resolution.Name)
		case fieldCreated:
			bucket = append(bucket, formatDateTime(issue.Fields.Created, jira.RFC3339))
		case fieldUpdated:
			bucket = append(bucket, formatDateTime(issue.Fields.Updated, jira.RFC3339))
		}
	}

	return bucket
}
