package view

import (
	"strings"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	colPadding  = 1
	maxColWidth = 70
)

var header = []string{
	"TYPE",
	"KEY",
	"SUMMARY",
	"ASSIGNEE",
	"REPORTER",
	"PRIORITY",
	"STATUS",
	"CREATED",
	"UPDATED",
}

// List is a list view.
type List struct {
	Data []jira.Issue
}

func (l *List) data() tui.TableData {
	var data tui.TableData

	data = append(data, header)

	for _, issue := range l.Data {
		data = append(data, []string{
			issue.Fields.IssueType.Name,
			issue.Key,
			strings.TrimSpace(issue.Fields.Summary),
			issue.Fields.Assignee.Name,
			issue.Fields.Reporter.Name,
			issue.Fields.Priority.Name,
			issue.Fields.Status.Name,
			formatDateTime(issue.Fields.Created),
			formatDateTime(issue.Fields.Updated),
		})
	}

	return data
}

// Render renders the list view.
func (l *List) Render() error {
	table := tui.NewTable(
		tui.NewScreen(),
		tui.WithColPadding(colPadding),
		tui.WithMaxColWidth(maxColWidth),
	)

	return table.Render(l.data())
}
