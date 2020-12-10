package view

import (
	"fmt"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// SprintIssueFunc provides issues in the sprint.
type SprintIssueFunc func(boardID, sprintID int) []*jira.Issue

// SprintList is a list view for issues.
type SprintList struct {
	Project string
	Board   string
	Server  string
	Data    []*jira.Sprint
	Issues  SprintIssueFunc

	issueCache map[string]tui.TableData
}

func (sl SprintList) data() []tui.PreviewData {
	data := make([]tui.PreviewData, 0, len(sl.Data))

	data = append(data, tui.PreviewData{
		Key:  "help",
		Menu: "?",
		Contents: func(s string) interface{} {
			return helpText
		},
	})

	for _, s := range sl.Data {
		bid, sid := s.BoardID, s.ID

		data = append(data, tui.PreviewData{
			Key: fmt.Sprintf("%d-%d-%s", bid, sid, s.StartDate),
			Menu: fmt.Sprintf(
				"➤ #%d %s: ⦗%s - %s⦘",
				s.ID,
				prepareTitle(s.Name),
				formatDateTimeHuman(s.StartDate, time.RFC3339),
				formatDateTimeHuman(s.EndDate, time.RFC3339),
			),
			Contents: func(key string) interface{} {
				if sl.issueCache == nil {
					sl.issueCache = make(map[string]tui.TableData)
				}

				if _, ok := sl.issueCache[key]; !ok {
					issues := sl.Issues(bid, sid)

					sl.issueCache[key] = sl.tabularize(issues)
				}

				return sl.issueCache[key]
			},
		})
	}

	return data
}

func (sl SprintList) tabularize(issues []*jira.Issue) tui.TableData {
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

func (sl SprintList) tableHeader() []string {
	return []string{
		"ID",
		"NAME",
		"START DATE",
		"END DATE",
		"COMPLETION DATE",
		"STATUS",
	}
}

func (sl SprintList) tableData() tui.TableData {
	var data tui.TableData

	data = append(data, sl.tableHeader())

	for _, s := range sl.Data {
		data = append(data, []string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			formatDateTime(s.StartDate, time.RFC3339),
			formatDateTime(s.EndDate, time.RFC3339),
			formatDateTime(s.CompleteDate, time.RFC3339),
			s.Status,
		})
	}

	return data
}

// Render renders the sprint explorer view.
func (sl SprintList) Render() error {
	data := sl.data()

	view := tui.NewPreview(
		tui.WithPreviewFooterText(
			fmt.Sprintf(
				"Showing %d results from board \"%s\" of project \"%s\"",
				len(sl.Data), sl.Board, sl.Project,
			),
		),
		tui.WithInitialText(helpText),
		tui.WithContentTableOpts(tui.WithSelectedFunc(navigate(sl.Server))),
	)

	return view.Render(data)
}

// RenderInTable renders the list in table view.
func (sl SprintList) RenderInTable() error {
	data := sl.tableData()

	view := tui.NewTable(
		tui.WithColPadding(colPadding),
		tui.WithMaxColWidth(maxColWidth),
		tui.WithFooterText(
			fmt.Sprintf(
				"Showing %d results from board \"%s\" of project \"%s\"",
				len(sl.Data), sl.Board, sl.Project,
			),
		),
	)

	return view.Render(data)
}
