package view

import (
	"bytes"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"

	"github.com/rethab/jira-cli/pkg/adf"
	"github.com/rethab/jira-cli/pkg/jira"
	"github.com/rethab/jira-cli/pkg/tui"
)

func TestIssueDetailsRenderInPlainView(t *testing.T) {
	t.Parallel()

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
			Components: []struct {
				Name string `json:"name"`
			}{{Name: "BE"}, {Name: "FE"}},
			Comment: struct {
				Comments []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				} `json:"comments"`
				Total int `json:"total"`
			}{Total: 0},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: true, WatchCount: 4},
			Created: "2020-12-13T14:05:20.974+0100",
			Updated: "2020-12-13T14:07:20.974+0100",
		},
	}

	issue := Issue{
		Server:  "https://test.local",
		Data:    data,
		Display: DisplayFormat{Plain: true},
	}

	expected := "Type: Bug  Status: Done  Updated: Sun, 13 Dec 20  Assignee: Person A  Key: TEST-1  Comments: 0  Linked: 0\n# This is a test\nCreated: Sun, 13 Dec 20  Reporter: Person Z  Priority: High  Components: BE, FE  Labels: None  Watchers: You + 3 watchers\n\n------------------------ Description ------------------------\n\nTest description\n\n\n"
	if xterm256() {
		expected += "\x1b[38;5;242mView this issue on Jira: https://test.local/browse/TEST-1\x1b[m"
	} else {
		expected += "\x1b[0;90mView this issue on Jira: https://test.local/browse/TEST-1\x1b[0m"
	}
	actual := issue.String()

	assert.NoError(t, issue.renderPlain(&b))
	assert.Equal(t, tui.TextData(expected), tui.TextData(actual))
}

func TestIssueDetailsWithV2Description(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer

	data := &jira.Issue{
		Key: "TEST-1",
		Fields: jira.IssueFields{
			Summary: "This is a test",
			Resolution: struct {
				Name string `json:"name"`
			}{Name: "Fixed"},
			Description: "h1. Title\nh2. Subtitle\n\nThis is a *bold* and _italic_ text with [a link|https://ankit.pl] in between.",
			IssueType:   jira.IssueType{Name: "Bug"},
			Parent: &struct {
				Key string `json:"key"`
			}{Key: "TEST-0"},
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
			Comment: struct {
				Comments []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				} `json:"comments"`
				Total int `json:"total"`
			}{
				Comments: []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				}{
					{ID: "10033", Author: jira.User{Name: "Person A"}, Body: "Test comment A", Created: "2021-11-22T23:44:13.782+0100"},
					{ID: "10034", Author: jira.User{Name: "Person B"}, Body: "Test comment B", Created: "2021-11-23T23:44:13.782+0100"},
					{ID: "10035", Author: jira.User{Name: "Person C"}, Body: "Test comment C", Created: "2021-11-24T23:44:13.782+0100"},
				},
				Total: 3,
			},
			Subtasks: []jira.Issue{
				{
					Key: "TEST-2",
					Fields: jira.IssueFields{
						Summary: "Subtask 1",
						Status: struct {
							Name string `json:"name"`
						}{Name: "TO DO"},
						Priority: struct {
							Name string `json:"name"`
						}{Name: "High"},
					},
				},
				{
					Key: "TEST-3",
					Fields: jira.IssueFields{
						Summary: "Subtask 2",
						Status: struct {
							Name string `json:"name"`
						}{Name: "Done"},
						Priority: struct {
							Name string `json:"name"`
						}{Name: "Normal"},
					},
				},
			},
			IssueLinks: []struct {
				ID       string `json:"id"`
				LinkType struct {
					Name    string `json:"name"`
					Inward  string `json:"inward"`
					Outward string `json:"outward"`
				} `json:"type"`
				InwardIssue  *jira.Issue `json:"inwardIssue,omitempty"`
				OutwardIssue *jira.Issue `json:"outwardIssue,omitempty"`
			}{
				{
					LinkType: struct {
						Name    string `json:"name"`
						Inward  string `json:"inward"`
						Outward string `json:"outward"`
					}{Name: "blocks", Inward: "blocks", Outward: "is blocked by"},
					InwardIssue: &jira.Issue{
						Key: "TEST-2",
						Fields: jira.IssueFields{
							Summary:   "Something is broken",
							IssueType: jira.IssueType{Name: "Bug"},
							Priority: struct {
								Name string `json:"name"`
							}{Name: "High"}, Status: struct {
								Name string `json:"name"`
							}{Name: "TO DO"},
						},
					},
				},
				{
					LinkType: struct {
						Name    string `json:"name"`
						Inward  string `json:"inward"`
						Outward string `json:"outward"`
					}{Name: "relates", Inward: "relates", Outward: "relates to"},
					OutwardIssue: &jira.Issue{
						Key: "TEST-3",
						Fields: jira.IssueFields{
							Summary:   "Everything is on fire",
							IssueType: jira.IssueType{Name: "Bug"},
							Priority: struct {
								Name string `json:"name"`
							}{Name: "Urgent"}, Status: struct {
								Name string `json:"name"`
							}{Name: "Done"},
						},
					},
				},
			},
			Created: "2020-12-13T14:05:20.974+0100",
			Updated: "2020-12-13T14:07:20.974+0100",
		},
	}

	issue := Issue{
		Server:  "https://test.local",
		Data:    data,
		Display: DisplayFormat{Plain: true},
		Options: IssueOption{NumComments: 2},
	}
	assert.NoError(t, issue.renderPlain(&b))

	expected := "Type: Bug  Status: Done  Updated: Sun, 13 Dec 20  Assignee: Person A  Key: TEST-1  Parent: TEST-0  Comments: 3  Linked: 2\n# This is a test\nCreated: Sun, 13 Dec 20  Reporter: Person Z  Priority: High  Components: BE, FE  Labels: None  Watchers: 0 watchers\n\n------------------------ Description ------------------------\n\n# Title\n## Subtitle\nThis is a **bold** and _italic_ text with [a link](https://ankit.pl) in between.\n\n\n------------------------ 2 Subtasks ------------------------\n\n\n SUBTASKS\n\n  TEST-2 Subtask 1 | High   | TO DO\n  TEST-3 Subtask 2 | Normal | Done \n\n\n\n------------------------ Linked Issues ------------------------\n\n\n BLOCKS\n\n  TEST-2 Something is broken   | Bug | High   | TO DO\n\n RELATES TO\n\n  TEST-3 Everything is on fire | Bug | Urgent | Done \n\n\n\n------------------------ 3 Comments ------------------------\n\n\n Person C | Wed, 24 Nov 21 | Latest comment\n\nTest comment C\n\n\n\n Person B | Tue, 23 Nov 21\n\nTest comment B\n\n"
	if xterm256() {
		expected += "\x1b[38;5;242mUse --comments <limit> with `jira issue view` to load more comments\x1b[m\n\n"
		expected += "\x1b[38;5;242mView this issue on Jira: https://test.local/browse/TEST-1\x1b[m"
	} else {
		expected += "\x1b[0;90mUse --comments <limit> with `jira issue view` to load more comments\x1b[0m\n\n"
		expected += "\x1b[0;90mView this issue on Jira: https://test.local/browse/TEST-1\x1b[0m"
	}
	actual := issue.String()

	assert.Equal(t, tui.TextData(expected), tui.TextData(actual))
}

func nonASCII(msg string) []string {
	var out []string
	for _, r := range msg {
		if r > unicode.MaxASCII {
			out = append(out, string(r))
		}
	}
	return out
}

// A long summary forces shortenAndPad to append its ellipsis.
const longSummary = "This summary is deliberately made long enough that the renderer has to truncate it"

func decoratedIssue() *jira.Issue {
	return &jira.Issue{
		Key: "TEST-1",
		Fields: jira.IssueFields{
			Summary:     "This is a test",
			Description: "Test description",
			IssueType:   jira.IssueType{Name: "Bug"},
			Parent: &struct {
				Key string `json:"key"`
			}{Key: "TEST-0"},
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
			}{{Name: "BE"}},
			Comment: struct {
				Comments []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				} `json:"comments"`
				Total int `json:"total"`
			}{
				Comments: []struct {
					ID      string      `json:"id"`
					Author  jira.User   `json:"author"`
					Body    interface{} `json:"body"`
					Created string      `json:"created"`
				}{
					{ID: "10033", Author: jira.User{Name: "Person A"}, Body: "Test comment A", Created: "2021-11-22T23:44:13.782+0100"},
				},
				Total: 1,
			},
			Subtasks: []jira.Issue{
				{
					Key: "TEST-2",
					Fields: jira.IssueFields{
						Summary: longSummary,
						Status: struct {
							Name string `json:"name"`
						}{Name: "TO DO"},
						Priority: struct {
							Name string `json:"name"`
						}{Name: "High"},
					},
				},
			},
			IssueLinks: []struct {
				ID       string `json:"id"`
				LinkType struct {
					Name    string `json:"name"`
					Inward  string `json:"inward"`
					Outward string `json:"outward"`
				} `json:"type"`
				InwardIssue  *jira.Issue `json:"inwardIssue,omitempty"`
				OutwardIssue *jira.Issue `json:"outwardIssue,omitempty"`
			}{
				{
					LinkType: struct {
						Name    string `json:"name"`
						Inward  string `json:"inward"`
						Outward string `json:"outward"`
					}{Name: "blocks", Inward: "blocks", Outward: "is blocked by"},
					InwardIssue: &jira.Issue{
						Key: "TEST-3",
						Fields: jira.IssueFields{
							Summary:   longSummary,
							IssueType: jira.IssueType{Name: "Bug"},
							Priority: struct {
								Name string `json:"name"`
							}{Name: "High"},
							Status: struct {
								Name string `json:"name"`
							}{Name: "TO DO"},
						},
					},
				},
			},
			Created: "2020-12-13T14:05:20.974+0100",
			Updated: "2020-12-13T14:07:20.974+0100",
		},
	}
}

// Plain output must survive a non-UTF-8 locale like LC_ALL=C, so none of the
// decorations the view adds around issue data may be non-ASCII.
func TestPlainIssueViewIsASCIIOnly(t *testing.T) {
	t.Parallel()

	issue := Issue{
		Server:  "https://test.local",
		Data:    decoratedIssue(),
		Display: DisplayFormat{Plain: true},
		Options: IssueOption{NumComments: 2},
	}

	assert.Empty(t, nonASCII(issue.String()))
}

func TestIssueViewKeepsDecorationsWhenNotPlain(t *testing.T) {
	t.Parallel()

	issue := Issue{
		Server:  "https://test.local",
		Data:    decoratedIssue(),
		Display: DisplayFormat{Plain: false},
		Options: IssueOption{NumComments: 2},
	}

	assert.NotEmpty(t, nonASCII(issue.String()))
}

func TestIssueDescription(t *testing.T) {
	t.Parallel()

	t.Run("it translates ADF description to markdown", func(t *testing.T) {
		t.Parallel()

		issue := Issue{
			Data: &jira.Issue{
				Key: "TEST-1",
				Fields: jira.IssueFields{
					Summary: "This is a test",
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
				},
			},
		}

		assert.Equal(t, "Test description\n\n", issue.Description())
	})

	t.Run("it returns an empty string when there is no description", func(t *testing.T) {
		t.Parallel()

		issue := Issue{
			Data: &jira.Issue{
				Key:    "TEST-1",
				Fields: jira.IssueFields{Summary: "This is a test"},
			},
		}

		assert.Equal(t, "", issue.Description())
	})
}

func TestSeparator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		body        string
		plain       bool
		expected    string
		expected256 string
	}{
		{
			name:        "it returns straight horizontal bar for empty message",
			body:        "",
			expected:    "\x1b[0;90m————————————————————————————————————————————————\x1b[0m",
			expected256: "\x1b[38;5;242m————————————————————————————————————————————————\x1b[m",
		},
		{
			name:        "it returns raw horizontal bar for empty message in plain output",
			body:        "",
			plain:       true,
			expected:    "------------------------------------------------",
			expected256: "------------------------------------------------",
		},
		{
			name:        "it returns greyed out message wrapped in horizontal bar",
			body:        "Some text",
			expected:    "\x1b[0;90m———————————————————————— Some text ————————————————————————\x1b[0m",
			expected256: "\x1b[38;5;242m———————————————————————— Some text ————————————————————————\x1b[m",
		},
		{
			name:        "it returns greyed out message wrapped in raw horizontal bar for plain output",
			body:        "Some text",
			plain:       true,
			expected:    "------------------------ Some text ------------------------",
			expected256: "------------------------ Some text ------------------------",
		},
		{
			name:        "it doesn't trim spaces",
			body:        "  ",
			expected:    "\x1b[0;90m————————————————————————    ————————————————————————\x1b[0m",
			expected256: "\x1b[38;5;242m————————————————————————    ————————————————————————\x1b[m",
		},
		{
			name:        "it doesn't trim spaces for plain output",
			body:        "  ",
			plain:       true,
			expected:    "------------------------    ------------------------",
			expected256: "------------------------    ------------------------",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			issue := Issue{
				Data: &jira.Issue{
					Key: "TEST-1",
				},
				Display: DisplayFormat{Plain: tc.plain},
			}

			if xterm256() {
				assert.Equal(t, tc.expected256, issue.separator(tc.body))
			} else {
				assert.Equal(t, tc.expected, issue.separator(tc.body))
			}
		})
	}
}
