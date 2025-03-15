package view

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestEpicData(t *testing.T) {
	epic1 := jira.Issue{
		Key: "TEST-1",
		Fields: jira.IssueFields{
			Summary: "This is a test",
			Resolution: struct {
				Name string `json:"name"`
			}{Name: "Fixed"},
			IssueType: jira.IssueType{Name: "Epic"},
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
	epic2 := jira.Issue{
		Key: "TEST-2",
		Fields: jira.IssueFields{
			Summary:   "[EPIC] This is another test",
			IssueType: jira.IssueType{Name: "Epic"},
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
		},
	}

	epic := EpicList{
		Total:   2,
		Project: "TEST",
		Server:  "https://test.local",
		Data:    []*jira.Issue{&epic1, &epic2},
		Issues: func(s string) []*jira.Issue {
			if s == "TEST-1" {
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
			Key:  "TEST-1",
			Menu: "➤ TEST-1: This is a test",
			Contents: tui.TableData{
				[]string{
					"TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE", "REPORTER", "PRIORITY", "RESOLUTION",
					"CREATED", "UPDATED",
				},
				[]string{
					"Bug", "ISSUE-1", "This is an issue", "Done", "Person A", "Person Z", "High", "Fixed",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20",
				},
			},
		},
		{
			Key:  "TEST-2",
			Menu: "➤ TEST-2: [EPIC[] This is another test",
			Contents: tui.TableData{
				[]string{
					"TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE", "REPORTER", "PRIORITY", "RESOLUTION",
					"CREATED", "UPDATED",
				},
				[]string{
					"Story", "ISSUE-2", "This is another issue", "Open", "", "Person A", "Normal", "",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20",
				},
				[]string{
					"Bug", "ISSUE-1", "This is an issue", "Done", "Person A", "Person Z", "High", "Fixed",
					"2020-12-13 14:05:20", "2020-12-13 14:07:20",
				},
			},
		},
	}

	for i, d := range epic.data() {
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
