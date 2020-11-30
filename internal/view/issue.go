package view

import (
	"fmt"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	colPadding  = 1
	maxColWidth = 60
)

// List is a list view.
type List struct {
	Total   int
	Project string
	Data    []jira.Issue
}

func (l List) header() []string {
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

func (l List) data() tui.TableData {
	var data tui.TableData

	data = append(data, l.header())

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
			formatDateTime(issue.Fields.Created),
			formatDateTime(issue.Fields.Updated),
		})
	}

	return data
}

// Render renders the list view.
func (l List) Render() error {
	data := l.data()

	table := tui.NewTable(
		tui.NewScreen(),
		tui.WithColPadding(colPadding),
		tui.WithMaxColWidth(maxColWidth),
		tui.WithFooterText(fmt.Sprintf("Showing %d of %d results for project \"%s\"", len(data)-1, l.Total, l.Project)),
	)

	return table.Render(data)
}
