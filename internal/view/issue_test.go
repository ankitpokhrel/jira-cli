package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestIssueDetailsRenderInPlainView(t *testing.T) {
	var b bytes.Buffer

	data := &jira.Issue{
		Key: "TEST-1",
		Fields: jira.IssueFields{
			Summary: "This is a test",
			Resolution: struct {
				Name string `json:"name"`
			}{Name: "Fixed"},
			Description: &adf.ADF{
				Version: 1,
				DocType: "doc",
				Content: []*adf.Node{
					{
						NodeType: "paragraph",
						Content: []*adf.Node{
							{NodeType: "text", NodeValue: adf.NodeValue{Text: "Test description"}},
						},
					},
				},
			},
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
			Components: []struct {
				Name string `json:"name"`
			}{{Name: "BE"}, {Name: "FE"}},
			Created: "2020-12-13T14:05:20.974+0100",
			Updated: "2020-12-13T14:07:20.974+0100",
		},
	}

	issue := Issue{
		Data:    data,
		Display: DisplayFormat{Plain: true},
	}
	assert.NoError(t, issue.renderPlain(&b))

	expected := "🐞 Bug  ✅ Done  ⌛ Sun, 13 Dec 20  👷 Person A\n# This is a test\n⏱️  Sun, 13 Dec 20  🔎 Person Z  🚀 High  📦 BE, FE  🏷️  None\n\n-----------\nTest description\n\n"
	assert.Equal(t, tui.TextData(expected), issue.data())

	rendered := "\n  🐞 Bug  ✅ Done  ⌛ Sun, 13 Dec 20  👷 Person A                                                                     \n                                                                                                                      \n  # This is a test                                                                                                    \n                                                                                                                      \n  ⏱️  Sun, 13 Dec 20  🔎 Person Z  🚀 High  📦 BE, FE  🏷️  None                                                       \n                                                                                                                      \n  --------                                                                                                            \n                                                                                                                      \n  Test description                                                                                                    \n\n"
	assert.Equal(t, rendered, b.String())
}
