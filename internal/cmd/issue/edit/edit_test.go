package edit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rethab/jira-cli/pkg/adf"
	"github.com/rethab/jira-cli/pkg/jira"
)

func TestBuildEditRequestConvertsMarkdownBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantBody string
	}{
		{
			name:     "markdown body is converted to wiki markup",
			body:     "# Heading\n\nSome **bold** text",
			wantBody: "h1. Heading\nSome *bold* text\n\n",
		},
		{
			name:     "empty body is left untouched",
			body:     "",
			wantBody: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			edr := buildEditRequest("TEST", &jira.Issue{}, &editParams{body: tc.body})

			assert.Equal(t, tc.wantBody, edr.Body)
		})
	}
}

func TestEditorBodyIsMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		description interface{} // string on-prem/Server, *adf.ADF on Cloud
		want        string
	}{
		{
			name:        "on-prem/server wiki markup is translated to markdown",
			description: "h1. Heading\n\nSome *bold* text",
			want:        "# Heading\nSome **bold** text\n",
		},
		{
			name: "cloud ADF is translated to markdown",
			description: &adf.ADF{
				Version: 1,
				DocType: "doc",
				Content: []*adf.Node{
					{
						NodeType:   adf.NodeHeading,
						Attributes: map[string]any{"level": float64(1)},
						Content: []*adf.Node{
							{NodeType: adf.ChildNodeText, NodeValue: adf.NodeValue{Text: "Heading"}},
						},
					},
				},
			},
			want: "# Heading\n",
		},
		{
			name:        "missing description yields an empty body",
			description: nil,
			want:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := &jira.Issue{Fields: jira.IssueFields{Description: tc.description}}

			assert.Equal(t, tc.want, editorBody(issue))
		})
	}
}

// An on-prem/Server description is served as wiki markup but the editor always shows
// markdown, so an interactive edit must not re-interpret the wiki markup as markdown.
func TestServerInteractiveEditRoundTrip(t *testing.T) {
	const wiki = "h2. Heading\n\n# first ordered item\n# second ordered item\n\nSome *bold wiki* text\n"

	issue := &jira.Issue{Fields: jira.IssueFields{Description: wiki}}

	prefilled := editorBody(issue)
	assert.Equal(t, "## Heading\n\n- first ordered item\n- second ordered item\nSome **bold wiki** text\n", prefilled)

	// The user edits a single word in $EDITOR and saves.
	edited := strings.Replace(prefilled, "Heading", "Updated heading", 1)

	edr := buildEditRequest("TEST", issue, &editParams{body: edited})

	assert.Equal(
		t,
		"h2. Updated heading\n* first ordered item\n* second ordered item\nSome *bold wiki* text\n\n",
		edr.Body,
	)
	assert.NotContains(t, edr.Body, "h1.", "wiki ordered list items must not become headings")
	assert.NotContains(t, edr.Body, "_bold wiki_", "wiki bold must not become italic")
}

func TestBuildEditRequestMergesIssueFieldsWithParams(t *testing.T) {
	issue := &jira.Issue{
		Fields: jira.IssueFields{
			Labels: []string{"existing"},
			Components: []struct {
				Name string `json:"name"`
			}{{Name: "BE"}},
			FixVersions: []struct {
				Name string `json:"name"`
			}{{Name: "v1.0"}},
			AffectsVersions: []struct {
				Name string `json:"name"`
			}{{Name: "v0.9"}},
			Parent: &struct {
				Key string `json:"key"`
			}{Key: "TEST-1"},
		},
	}

	t.Run("params are appended to the existing issue fields", func(t *testing.T) {
		params := &editParams{
			summary:         "New summary",
			priority:        "High",
			labels:          []string{"added"},
			components:      []string{"FE"},
			fixVersions:     []string{"v2.0"},
			affectsVersions: []string{"v1.9"},
			customFields:    map[string]string{"story-points": "5"},
			skipNotify:      true,
		}

		edr := buildEditRequest("TEST", issue, params)

		assert.Equal(t, "New summary", edr.Summary)
		assert.Equal(t, "High", edr.Priority)
		assert.Equal(t, []string{"added", "existing"}, edr.Labels)
		assert.Equal(t, []string{"BE", "FE"}, edr.Components)
		assert.Equal(t, []string{"v1.0", "v2.0"}, edr.FixVersions)
		assert.Equal(t, []string{"v0.9", "v1.9"}, edr.AffectsVersions)
		assert.Equal(t, map[string]string{"story-points": "5"}, edr.CustomFields)
		assert.True(t, edr.SkipNotify)
	})

	t.Run("existing parent is kept when no parent is given", func(t *testing.T) {
		edr := buildEditRequest("TEST", issue, &editParams{})

		assert.Equal(t, "TEST-1", edr.ParentIssueKey)
	})

	t.Run("given parent overrides the existing one and is qualified with the project", func(t *testing.T) {
		edr := buildEditRequest("TEST", issue, &editParams{parentIssueKey: "2"})

		assert.Equal(t, "TEST-2", edr.ParentIssueKey)
	})
}
