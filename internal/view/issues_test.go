package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestIssueData(t *testing.T) {
	issue := IssueList{
		Total:   2,
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:     false,
			NoHeaders: false,
		},
	}
	expected := tui.TableData{
		[]string{
			"TYPE", "KEY", "SUMMARY", "STATUS", "ASSIGNEE", "REPORTER", "PRIORITY", "RESOLUTION",
			"CREATED", "UPDATED", "LABELS",
		},
		[]string{
			"Bug", "TEST-1", "This is a test", "Done", "Person A", "Person Z", "High", "Fixed",
			"2020-12-13 14:05:20", "2020-12-13 14:07:20", "krakatit",
		},
		[]string{
			"Story", "TEST-2", "This is another test", "Open", "", "Person A", "Normal", "",
			"2020-12-13 14:05:20", "2020-12-13 14:07:20", "pat,mat",
		},
	}
	assert.Equal(t, expected, issue.data())
}

func TestIssueRenderInPlainView(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Total:   2,
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  false,
			NoTruncate: false,
		},
	}
	assert.NoError(t, issue.renderPlain(&b))

	expected := `TYPE	KEY	SUMMARY	STATUS
Bug	TEST-1	This is a test	Done
Story	TEST-2	This is another test	Open
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewAndNoTruncate(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Total:   2,
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  false,
			NoTruncate: true,
		},
	}
	assert.NoError(t, issue.renderPlain(&b))

	expected := `TYPE	KEY	SUMMARY	STATUS	ASSIGNEE	REPORTER	PRIORITY	RESOLUTION	CREATED	UPDATED	LABELS
Bug	TEST-1	This is a test	Done	Person A	Person Z	High	Fixed	2020-12-13 14:05:20	2020-12-13 14:07:20	krakatit
Story	TEST-2	This is another test	Open		Person A	Normal		2020-12-13 14:05:20	2020-12-13 14:07:20	pat,mat
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewWithoutHeaders(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Total:   2,
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  true,
			NoTruncate: true,
		},
	}
	assert.NoError(t, issue.renderPlain(&b))

	expected := `Bug	TEST-1	This is a test	Done	Person A	Person Z	High	Fixed	2020-12-13 14:05:20	2020-12-13 14:07:20	krakatit
Story	TEST-2	This is another test	Open		Person A	Normal		2020-12-13 14:05:20	2020-12-13 14:07:20	pat,mat
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewWithFewColumns(t *testing.T) {
	var b bytes.Buffer

	data := getIssues()

	issue := IssueList{
		Total:   2,
		Project: "TEST",
		Server:  "https://test.local",
		Data:    data,
		Display: DisplayFormat{
			Plain:     true,
			NoHeaders: false,
			Columns:   []string{"key", "type", "status", "created"},
		},
	}
	assert.NoError(t, issue.renderPlain(&b))

	expected := `KEY	TYPE	STATUS	CREATED
TEST-1	Bug	Done	2020-12-13 14:05:20
TEST-2	Story	Open	2020-12-13 14:05:20
`
	assert.Equal(t, expected, b.String())
}

func getIssues() []*jira.Issue {
	return []*jira.Issue{
		{
			Key: "TEST-1",
			Fields: jira.IssueFields{
				Summary: "This is a test",
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
				Labels:  []string{"krakatit"},
			},
		},
		{
			Key: "TEST-2",
			Fields: jira.IssueFields{
				Summary:   "This is another test",
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
				Labels:  []string{"pat", "mat"},
			},
		},
	}
}
