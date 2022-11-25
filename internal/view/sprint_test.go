package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestSprintPreviewLayoutData(t *testing.T) {
	sprint1 := jira.Sprint{
		ID:        1,
		Name:      "Sprint 1",
		Status:    "closed",
		StartDate: "2020-12-07T16:12:00.000Z",
		EndDate:   "2020-12-13T16:12:00.000Z",
		BoardID:   1,
	}
	sprint2 := jira.Sprint{
		ID:        2,
		Name:      "Sprint 2",
		Status:    "active",
		StartDate: "2020-12-13T16:12:00.000Z",
		EndDate:   "2020-12-19T16:12:00.000Z",
		BoardID:   1,
	}

	issue1 := jira.Issue{
		Key: "ISSUE-1",
		Fields: jira.IssueFields{
			Summary: "This is an issue",
			Resolution: struct {
				Name string `json:"name"`
			}{Name: "Fixed"},
			IssueType: jira.IssueType{Name: "Bug"},
			Assignee: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "High"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person Z"},
			Status: struct {
				Name string `json:"name"`
			}{Name: "Done"},
			Created: "2020-12-13T14:05:20.974+0100",
			Updated: "2020-12-13T14:07:20.974+0100",
			Labels:  []string{"urgent"},
		},
	}
	issue2 := jira.Issue{
		Key: "ISSUE-2",
		Fields: jira.IssueFields{
			Summary:   "This is another issue",
			IssueType: jira.IssueType{Name: "Story"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Normal"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Status: struct {
				Name string `json:"name"`
			}{Name: "Open"},
			Created: "2020-12-13T14:05:20.974+0100",
			Updated: "2020-12-13T14:07:20.974+0100",
			Labels:  []string{"blocked"},
		},
	}

	sprint := SprintList{
		Project: "TEST",
		Board:   "Test Board",
		Server:  "https://test.local",
		Data:    []*jira.Sprint{&sprint1, &sprint2},
		Issues: func(boardID, sprintID int) []*jira.Issue {
			if sprintID == 1 {
				return []*jira.Issue{&issue1}
			}
			return []*jira.Issue{&issue2, &issue1}
		},
	}

	expected := []struct {
		Key      string
		Menu     string
		Contents interface{}
	}{
		{
			Key:      "help",
			Menu:     "?",
			Contents: helpText,
		},
		{
			Key:  "1-1-2020-12-07T16:12:00.000Z",
			Menu: "➤ #1 Sprint 1: ⦗Mon, 07 Dec 20 - Sun, 13 Dec 20⦘",
			Contents: tui.TableData{
				[]string{
					"TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE", "REPORTER", "PRIORITY", "RESOLUTION",
					"CREATED", "UPDATED", "LABELS",
				},
				[]string{
					"Bug", "ISSUE-1", "This is an issue", "Done", "Person A", "Person Z", "High", "Fixed",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20", "urgent",
				},
			},
		},
		{
			Key:  "1-2-2020-12-13T16:12:00.000Z",
			Menu: "➤ #2 Sprint 2: ⦗Sun, 13 Dec 20 - Sat, 19 Dec 20⦘",
			Contents: tui.TableData{
				[]string{
					"TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE", "REPORTER", "PRIORITY", "RESOLUTION",
					"CREATED", "UPDATED", "LABELS",
				},
				[]string{
					"Story", "ISSUE-2", "This is another issue", "Open", "", "Person A", "Normal", "",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20", "blocked",
				},
				[]string{
					"Bug", "ISSUE-1", "This is an issue", "Done", "Person A", "Person Z", "High", "Fixed",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20", "urgent",
				},
			},
		},
	}

	for i, d := range sprint.data() {
		assert.Equal(t, expected[i].Key, d.Key)
		assert.Equal(t, expected[i].Menu, d.Menu)

		switch d.Contents(d.Key).(type) {
		case string:
			assert.Equal(t, expected[i].Contents.(string), d.Contents(d.Key))

		case tui.TableData:
			assert.Equal(t, expected[i].Contents.(tui.TableData), d.Contents(d.Key))
		}
	}
}

func TestSprintTableLayoutData(t *testing.T) {
	sprint := SprintList{
		Project: "TEST",
		Board:   "Test Board",
		Server:  "https://test.local",
		Data: []*jira.Sprint{
			{
				ID:           1,
				Name:         "Sprint 1",
				Status:       "closed",
				StartDate:    "2020-12-07T16:12:00.000Z",
				EndDate:      "2020-12-13T16:12:00.000Z",
				CompleteDate: "2020-12-13T16:12:00.000Z",
				BoardID:      1,
			},
			{
				ID:        2,
				Name:      "Sprint 2",
				Status:    "active",
				StartDate: "2020-12-13T16:12:00.000Z",
				EndDate:   "2020-12-19T16:12:00.000Z",
				BoardID:   1,
			},
		},
	}

	expected := tui.TableData{
		[]string{"ID", "NAME", "START", "END", "COMPLETE", "STATE"},
		[]string{"1", "Sprint 1", "2020-12-07 16:12:00", "2020-12-13 16:12:00", "2020-12-13 16:12:00", "closed"},
		[]string{"2", "Sprint 2", "2020-12-13 16:12:00", "2020-12-19 16:12:00", "", "active"},
	}
	assert.Equal(t, expected, sprint.tableData())
}

func TestSprintRenderInPlainView(t *testing.T) {
	var b bytes.Buffer

	sprint := SprintList{
		Project: "TEST",
		Board:   "Test Board",
		Server:  "https://test.local",
		Data: []*jira.Sprint{
			{
				ID:           1,
				Name:         "Sprint 1",
				Status:       "closed",
				StartDate:    "2020-12-07T16:12:00.000Z",
				EndDate:      "2020-12-13T16:12:00.000Z",
				CompleteDate: "2020-12-13T16:12:00.000Z",
				BoardID:      1,
			},
			{
				ID:        2,
				Name:      "Sprint 2",
				Status:    "active",
				StartDate: "2020-12-13T16:12:00.000Z",
				EndDate:   "2020-12-19T16:12:00.000Z",
				BoardID:   1,
			},
		},
		Display: DisplayFormat{
			Plain:     true,
			NoHeaders: false,
		},
	}
	assert.NoError(t, sprint.renderPlain(&b))

	expected := `ID	NAME	START	END	COMPLETE	STATE
1	Sprint 1	2020-12-07 16:12:00	2020-12-13 16:12:00	2020-12-13 16:12:00	closed
2	Sprint 2	2020-12-13 16:12:00	2020-12-19 16:12:00		active
`
	assert.Equal(t, expected, b.String())
}

func TestSprintRenderInPlainViewWithoutHeaders(t *testing.T) {
	var b bytes.Buffer

	sprint := SprintList{
		Project: "TEST",
		Board:   "Test Board",
		Server:  "https://test.local",
		Data: []*jira.Sprint{
			{
				ID:           1,
				Name:         "Sprint 1",
				Status:       "closed",
				StartDate:    "2020-12-07T16:12:00.000Z",
				EndDate:      "2020-12-13T16:12:00.000Z",
				CompleteDate: "2020-12-13T16:12:00.000Z",
				BoardID:      1,
			},
			{
				ID:        2,
				Name:      "Sprint 2",
				Status:    "active",
				StartDate: "2020-12-13T16:12:00.000Z",
				EndDate:   "2020-12-19T16:12:00.000Z",
				BoardID:   1,
			},
		},
		Display: DisplayFormat{
			Plain:     true,
			NoHeaders: true,
		},
	}
	assert.NoError(t, sprint.renderPlain(&b))

	expected := `1	Sprint 1	2020-12-07 16:12:00	2020-12-13 16:12:00	2020-12-13 16:12:00	closed
2	Sprint 2	2020-12-13 16:12:00	2020-12-19 16:12:00		active
`
	assert.Equal(t, expected, b.String())
}

func TestSprintRenderInPlainViewWithFewColumns(t *testing.T) {
	var b bytes.Buffer

	sprint := SprintList{
		Project: "TEST",
		Board:   "Test Board",
		Server:  "https://test.local",
		Data: []*jira.Sprint{
			{
				ID:           1,
				Name:         "Sprint 1",
				Status:       "closed",
				StartDate:    "2020-12-07T16:12:00.000Z",
				EndDate:      "2020-12-13T16:12:00.000Z",
				CompleteDate: "2020-12-13T16:12:00.000Z",
				BoardID:      1,
			},
			{
				ID:        2,
				Name:      "Sprint 2",
				Status:    "active",
				StartDate: "2020-12-13T16:12:00.000Z",
				EndDate:   "2020-12-19T16:12:00.000Z",
				BoardID:   1,
			},
		},
		Display: DisplayFormat{
			Plain:     true,
			NoHeaders: false,
			Columns:   []string{"name", "start", "end"},
		},
	}
	assert.NoError(t, sprint.renderPlain(&b))

	expected := `NAME	START	END
Sprint 1	2020-12-07 16:12:00	2020-12-13 16:12:00
Sprint 2	2020-12-13 16:12:00	2020-12-19 16:12:00
`
	assert.Equal(t, expected, b.String())
}
