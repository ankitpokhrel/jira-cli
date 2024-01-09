package view

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// SprintIssueFunc provides issues in the sprint.
type SprintIssueFunc func(boardID, sprintID int) []*jira.Issue

// SprintList is a list view for sprints.
type SprintList struct {
	Project string
	Board   string
	Server  string
	Data    []*jira.Sprint
	Issues  SprintIssueFunc
	Display DisplayFormat
}

// Render renders the sprint explorer view.
//
//nolint:dupl
func (sl *SprintList) Render() error {
	renderer, err := MDRenderer()
	if err != nil {
		return err
	}

	data := sl.data()
	view := tui.NewPreview(
		tui.WithPreviewFooterText(
			fmt.Sprintf(
				"Showing %d results from board %q of project %q",
				len(sl.Data), sl.Board, sl.Project,
			),
		),
		tui.WithInitialText(helpText),
		tui.WithContentTableOpts(
			tui.WithFixedColumns(sl.Display.FixedColumns),
			tui.WithTableStyle(sl.Display.TableStyle),
			tui.WithSelectedFunc(navigate(sl.Server)),
			tui.WithViewModeFunc(func(r, c int, d interface{}) (func() interface{}, func(interface{}) (string, error)) {
				dataFn := func() interface{} {
					data := d.(tui.TableData)
					ci := data.GetIndex(fieldKey)
					iss, _ := api.ProxyGetIssue(api.DefaultClient(false), data.Get(r, ci), issue.NewNumCommentsFilter(1))
					return iss
				}
				renderFn := func(i interface{}) (string, error) {
					iss := Issue{
						Server:  sl.Server,
						Data:    i.(*jira.Issue),
						Options: IssueOption{NumComments: 1},
					}
					return iss.RenderedOut(renderer)
				}
				return dataFn, renderFn
			}),
			tui.WithCopyFunc(copyURL(sl.Server)),
			tui.WithCopyKeyFunc(copyKey()),
		),
	)

	return view.Paint(data)
}

// RenderInTable renders the list in table view.
func (sl *SprintList) RenderInTable() error {
	if sl.Display.Plain || tui.IsDumbTerminal() || tui.IsNotTTY() {
		w := tabwriter.NewWriter(os.Stdout, 0, tabWidth, 1, '\t', 0)
		return sl.renderPlain(w)
	}

	data := sl.tableData()
	view := tui.NewTable(
		tui.WithFixedColumns(sl.Display.FixedColumns),
		tui.WithTableStyle(sl.Display.TableStyle),
		tui.WithTableFooterText(
			fmt.Sprintf(
				"Showing %d results from board %q of project %q",
				len(sl.Data), sl.Board, sl.Project,
			),
		),
	)

	return view.Paint(data)
}

// renderPlain renders the issue in plain view.
func (sl *SprintList) renderPlain(w io.Writer) error {
	return renderPlain(w, sl.tableData())
}

func (sl *SprintList) data() []tui.PreviewData {
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
				cmdutil.FormatDateTimeHuman(s.StartDate, time.RFC3339),
				cmdutil.FormatDateTimeHuman(s.EndDate, time.RFC3339),
			),
			Contents: func(key string) interface{} {
				issues := sl.Issues(bid, sid)
				return sl.tabularize(issues)
			},
		})
	}

	return data
}

func (sl *SprintList) tabularize(issues []*jira.Issue) tui.TableData {
	var data tui.TableData

	data = append(data, ValidIssueColumns())
	for _, issue := range issues {
		data = append(data, []string{
			issue.Fields.IssueType.Name,
			issue.Key,
			prepareTitle(issue.Fields.Summary),
			issue.Fields.Status.Name,
			issue.Fields.Assignee.Name,
			issue.Fields.Reporter.Name,
			issue.Fields.Priority.Name,
			issue.Fields.Resolution.Name,
			formatDateTime(issue.Fields.Created, jira.RFC3339, sl.Display.Timezone),
			formatDateTime(issue.Fields.Updated, jira.RFC3339, sl.Display.Timezone),
			strings.Join(issue.Fields.Labels, ","),
		})
	}

	return data
}

func (sl *SprintList) validColumnsMap() map[string]struct{} {
	columns := ValidSprintColumns()
	out := make(map[string]struct{}, len(columns))

	for _, c := range columns {
		out[c] = struct{}{}
	}

	return out
}

func (sl *SprintList) tableHeader() []string {
	if len(sl.Display.Columns) == 0 {
		return ValidSprintColumns()
	}

	var headers []string

	columnsMap := sl.validColumnsMap()
	for _, c := range sl.Display.Columns {
		c = strings.ToUpper(c)
		if _, ok := columnsMap[c]; ok {
			headers = append(headers, strings.ToUpper(c))
		}
	}

	return headers
}

func (sl *SprintList) tableData() tui.TableData {
	var data tui.TableData

	headers := sl.tableHeader()
	if !(sl.Display.Plain && sl.Display.NoHeaders) {
		data = append(data, headers)
	}
	if len(headers) == 0 {
		headers = ValidSprintColumns()
	}
	for _, s := range sl.Data {
		data = append(data, sl.assignColumns(headers, s))
	}

	return data
}

func (sl *SprintList) assignColumns(columns []string, sprint *jira.Sprint) []string {
	var bucket []string

	for _, column := range columns {
		switch column {
		case fieldID:
			bucket = append(bucket, fmt.Sprintf("%d", sprint.ID))
		case fieldName:
			bucket = append(bucket, sprint.Name)
		case fieldStartDate:
			bucket = append(bucket, formatDateTime(sprint.StartDate, time.RFC3339, sl.Display.Timezone))
		case fieldEndDate:
			bucket = append(bucket, formatDateTime(sprint.EndDate, time.RFC3339, sl.Display.Timezone))
		case fieldCompleteDate:
			bucket = append(bucket, formatDateTime(sprint.CompleteDate, time.RFC3339, sl.Display.Timezone))
		case fieldState:
			bucket = append(bucket, sprint.Status)
		}
	}

	return bucket
}
