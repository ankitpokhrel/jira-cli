package view

import (
	"fmt"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// EpicIssueFunc provides issues for the epic.
type EpicIssueFunc func(string) []*jira.Issue

// EpicList is a list view for epics.
type EpicList struct {
	Total   int
	Project string
	Server  string
	Data    []*jira.Issue
	Issues  EpicIssueFunc

	issueCache map[string]tui.TableData
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
				if el.issueCache == nil {
					el.issueCache = make(map[string]tui.TableData)
				}

				if _, ok := el.issueCache[key]; !ok {
					issues := el.Issues(key)

					el.issueCache[key] = el.tabularize(issues)
				}

				return el.issueCache[key]
			},
		})
	}

	return data
}

func (el EpicList) tabularize(issues []*jira.Issue) tui.TableData {
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
			formatDateTime(issue.Fields.Created, jira.RFC3339),
			formatDateTime(issue.Fields.Updated, jira.RFC3339),
		})
	}

	return data
}

// Render renders the epic explorer view.
func (el EpicList) Render() error {
	data := el.data()

	view := tui.NewPreview(
		tui.WithPreviewFooterText(fmt.Sprintf("Showing %d of %d results for project \"%s\"", len(el.Data), el.Total, el.Project)),
		tui.WithInitialText(helpText),
		tui.WithSidebarSelectedFunc(navigate(el.Server)),
		tui.WithContentTableOpts(tui.WithSelectedFunc(navigate(el.Server))),
	)

	return view.Render(data)
}
