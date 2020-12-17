package view

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	colPadding  = 1
	maxColWidth = 60
)

// DisplayFormat is a issue display type.
type DisplayFormat struct {
	Plain     bool
	NoHeaders bool
}

// IssueList is a list view for issues.
type IssueList struct {
	Total   int
	Project string
	Server  string
	Data    []*jira.Issue
	Display DisplayFormat
}

// Render renders the view.
func (l IssueList) Render() error {
	if l.Display.Plain {
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
		return l.renderPlain(w)
	}

	data := l.data()

	view := tui.NewTable(
		tui.WithColPadding(colPadding),
		tui.WithMaxColWidth(maxColWidth),
		tui.WithTableFooterText(fmt.Sprintf("Showing %d of %d results for project \"%s\"", len(data)-1, l.Total, l.Project)),
		tui.WithSelectedFunc(navigate(l.Server)),
	)

	return view.Render(data)
}

// renderPlain renders the issue in plain view.
func (l IssueList) renderPlain(w io.Writer) error {
	return renderPlain(w, l.data())
}

func (l IssueList) header() []string {
	return []string{
		"TYPE",
		"KEY",
		"SUMMARY",
		"ASSIGNEE",
		"REPORTER",
		"PRIORITY",
		"STATUS",
		"RESOLUTION",
		"CREATED",
		"UPDATED",
	}
}

func (l IssueList) data() tui.TableData {
	var data tui.TableData

	if !(l.Display.Plain && l.Display.NoHeaders) {
		data = append(data, l.header())
	}

	for _, issue := range l.Data {
		data = append(data, []string{
			issue.Fields.IssueType.Name,
			issue.Key,
			prepareTitle(issue.Fields.Summary),
			issue.Fields.Assignee.Name,
			issue.Fields.Reporter.Name,
			issue.Fields.Priority.Name,
			issue.Fields.Status.Name,
			issue.Fields.Resolution.Name,
			formatDateTime(issue.Fields.Created, jira.RFC3339),
			formatDateTime(issue.Fields.Updated, jira.RFC3339),
		})
	}

	return data
}
