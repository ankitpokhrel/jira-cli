package view

import (
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
			IssueType: struct {
				Name string `json:"name"`
			}{Name: "Bug"},
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
		},
	}
	issue2 := jira.Issue{
		Key: "ISSUE-2",
		Fields: jira.IssueFields{
			Summary: "This is another issue",
			IssueType: struct {
				Name string `json:"name"`
			}{Name: "Story"},
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
		},
	}

	sprint := SprintList{
		Project: "TEST",
		Board:   "Test Board",
		Server:  "https://test.local",
		Data: []*jira.Sprint{
			&sprint1,
			&sprint2,
		},
		Issues: func(boardID, sprintID int) []jira.Issue {
			if sprintID == 1 {
				return []jira.Issue{
					issue1,
				}
			}

			return []jira.Issue{
				issue2,
				issue1,
			}
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
			Menu: "➤ Sprint 1: ⦗Mon, 07 Dec 20 - Sun, 13 Dec 20⦘",
			Contents: tui.TableData{
				[]string{
					"TYPE", "KEY", "SUMMARY", "ASSIGNEE", "REPORTER", "PRIORITY", "STATUS", "RESOLUTION",
					"CREATED", "UPDATED",
				},
				[]string{
					"Bug", "ISSUE-1", "This is an issue", "Person A", "Person Z", "High", "Done", "Fixed",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20",
				},
			},
		},
		{
			Key:  "1-2-2020-12-13T16:12:00.000Z",
			Menu: "➤ Sprint 2: ⦗Sun, 13 Dec 20 - Sat, 19 Dec 20⦘",
			Contents: tui.TableData{
				[]string{
					"TYPE", "KEY", "SUMMARY", "ASSIGNEE", "REPORTER", "PRIORITY", "STATUS", "RESOLUTION",
					"CREATED", "UPDATED",
				},
				[]string{
					"Story", "ISSUE-2", "This is another issue", "", "Person A", "Normal", "Open", "",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20",
				},
				[]string{
					"Bug", "ISSUE-1", "This is an issue", "Person A", "Person Z", "High", "Done", "Fixed",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20",
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
		[]string{"ID", "NAME", "START DATE", "END DATE", "COMPLETION DATE", "STATUS"},
		[]string{"1", "Sprint 1", "2020-12-07 16:12:00", "2020-12-13 16:12:00", "2020-12-13 16:12:00", "closed"},
		[]string{"2", "Sprint 2", "2020-12-13 16:12:00", "2020-12-19 16:12:00", "", "active"},
	}

	assert.Equal(t, expected, sprint.tableData())
}
