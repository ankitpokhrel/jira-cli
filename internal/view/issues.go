package view

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
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
func (l IssueList) Render() error {
	if l.Display.Plain {
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
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
		tui.WithViewModeFunc(func(r, c int, _ interface{}) (func() interface{}, func(interface{}) error) {
			dataFn := func() interface{} {
				issue, _ := api.Client(jira.Config{Debug: true}).GetIssue(data[r][1])
				return issue
			}
			renderFn := func(i interface{}) error {
				out, err := renderer.Render(Issue{Data: i.(*jira.Issue)}.String())
				if err != nil {
					return err
				}
				return PagerOut(out)
			}
			return dataFn, renderFn
		}),
		tui.WithCopyFunc(copyURL(l.Server)),
	)

	return view.Paint(data)
}

// renderPlain renders the issue in plain view.
func (l IssueList) renderPlain(w io.Writer) error {
	return renderPlain(w, l.data())
}

func (l IssueList) validColumnsMap() map[string]struct{} {
	columns := ValidIssueColumns()
	out := make(map[string]struct{}, len(columns))

	for _, c := range columns {
		out[c] = struct{}{}
	}

	return out
}

func (l IssueList) header() []string {
	if len(l.Display.Columns) == 0 {
		validColumns := ValidIssueColumns()
		if l.Display.NoTruncate || !l.Display.Plain {
			return validColumns
		}
		return validColumns[0:4]
	}

	var headers []string

	columnsMap := l.validColumnsMap()
	for _, c := range l.Display.Columns {
		c = strings.ToUpper(c)
		if _, ok := columnsMap[c]; ok {
			headers = append(headers, strings.ToUpper(c))
		}
	}

	return headers
}

func (l IssueList) data() tui.TableData {
	var data tui.TableData

	headers := l.header()
	if !(l.Display.Plain && l.Display.NoHeaders) {
		data = append(data, headers)
	}
	if len(headers) == 0 {
		headers = ValidIssueColumns()
	}
	for _, issue := range l.Data {
		data = append(data, l.assignColumns(headers, issue))
	}

	return data
}

func (l IssueList) assignColumns(columns []string, issue *jira.Issue) []string {
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
