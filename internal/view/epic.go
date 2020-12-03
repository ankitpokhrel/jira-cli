package view

import (
	"fmt"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const helpText = `Use up and down arrow keys or 'j' and 'k' letters to navigate through the list.

	Press 'w' to toggle focus between the sidebar and the contents screen. On contents screen,
	you can use arrow keys or 'j', 'k', 'h', and 'l' letters to navigate through the epic issue list.

	Press 'q' / ESC / CTRL+c to quit.`

// EpicIssueFunc provides issues for the epic.
type EpicIssueFunc func(string) []jira.Issue

// EpicList is a list view for issues.
type EpicList struct {
	Total   int
	Project string
	Data    []jira.Issue
	Issues  EpicIssueFunc
}

func (el EpicList) data() []tui.PreviewData {
	data := make([]tui.PreviewData, 0, len(el.Data))

	data = append(data, tui.PreviewData{
		Key:  "help",
		Menu: "?",
		Contents: func(s string) interface{} {
			return helpText
		},
	})

	for _, issue := range el.Data {
		data = append(data, tui.PreviewData{
			Key:  issue.Key,
			Menu: fmt.Sprintf("➤ %s: %s", issue.Key, prepareTitle(issue.Fields.Summary)),
			Contents: func(key string) interface{} {
				issues := el.Issues(key)

				return el.tabularize(issues)
			},
		})
	}

	return data
}

func (el EpicList) tabularize(issues []jira.Issue) tui.TableData {
	var data tui.TableData

	data = append(data, []string{
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
	})

	for _, issue := range issues {
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
func (el EpicList) Render() error {
	data := el.data()

	view := tui.NewPreview(
		tui.WithPreviewFooterText(fmt.Sprintf("Showing %d of %d results for project \"%s\"", len(el.Data), el.Total, el.Project)),
		tui.WithInitialText(helpText),
	)

	return view.Render(data)
}
