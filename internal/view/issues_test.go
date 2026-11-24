package view

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestIssueData(t *testing.T) {
	issue := IssueList{
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
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  false,
			NoTruncate: false,
		},
	}
	assert.NoError(t, issue.renderPlain(&b, "\t"))

	expected := `TYPE	KEY	SUMMARY	STATUS
Bug	TEST-1	This is a test	Done
Story	TEST-2	This is another test	Open
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewWithCustomDelimiter(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  false,
			NoTruncate: false,
		},
	}
	assert.NoError(t, issue.renderPlain(&b, "|"))

	expected := `TYPE|KEY|SUMMARY|STATUS
Bug|TEST-1|This is a test|Done
Story|TEST-2|This is another test|Open
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewAndNoTruncate(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  false,
			NoTruncate: true,
		},
	}
	assert.NoError(t, issue.renderPlain(&b, "\t"))

	expected := `TYPE	KEY	SUMMARY	STATUS	ASSIGNEE	REPORTER	PRIORITY	RESOLUTION	CREATED	UPDATED	LABELS
Bug	TEST-1	This is a test	Done	Person A	Person Z	High	Fixed	2020-12-13 14:05:20	2020-12-13 14:07:20	krakatit
Story	TEST-2	This is another test	Open		Person A	Normal		2020-12-13 14:05:20	2020-12-13 14:07:20	pat,mat
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewWithoutHeaders(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			Plain:      true,
			NoHeaders:  true,
			NoTruncate: true,
		},
	}
	assert.NoError(t, issue.renderPlain(&b, "\t"))

	expected := `Bug	TEST-1	This is a test	Done	Person A	Person Z	High	Fixed	2020-12-13 14:05:20	2020-12-13 14:07:20	krakatit
Story	TEST-2	This is another test	Open		Person A	Normal		2020-12-13 14:05:20	2020-12-13 14:07:20	pat,mat
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInPlainViewWithFewColumns(t *testing.T) {
	var b bytes.Buffer

	data := getIssues()

	issue := IssueList{
		Project: "TEST",
		Server:  "https://test.local",
		Data:    data,
		Display: DisplayFormat{
			Plain:     true,
			NoHeaders: false,
			Columns:   []string{"key", "type", "status", "created"},
		},
	}
	assert.NoError(t, issue.renderPlain(&b, "\t"))

	expected := `KEY	TYPE	STATUS	CREATED
TEST-1	Bug	Done	2020-12-13 14:05:20
TEST-2	Story	Open	2020-12-13 14:05:20
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInCSVFormat(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			CSV:        true,
			NoHeaders:  false,
			NoTruncate: true,
		},
	}
	assert.NoError(t, issue.renderCSV(&b))

	expected := `TYPE,KEY,SUMMARY,STATUS,ASSIGNEE,REPORTER,PRIORITY,RESOLUTION,CREATED,UPDATED,LABELS
Bug,TEST-1,This is a test,Done,Person A,Person Z,High,Fixed,2020-12-13 14:05:20,2020-12-13 14:07:20,krakatit
Story,TEST-2,This is another test,Open,,Person A,Normal,,2020-12-13 14:05:20,2020-12-13 14:07:20,"pat,mat"
`
	assert.Equal(t, expected, b.String())
}

func TestIssueRenderInCSVFormatWithoutHeaders(t *testing.T) {
	var b bytes.Buffer

	issue := IssueList{
		Project: "TEST",
		Server:  "https://test.local",
		Data:    getIssues(),
		Display: DisplayFormat{
			CSV:        true,
			NoHeaders:  true,
			NoTruncate: true,
		},
	}
	assert.NoError(t, issue.renderCSV(&b))

	expected := `Bug,TEST-1,This is a test,Done,Person A,Person Z,High,Fixed,2020-12-13 14:05:20,2020-12-13 14:07:20,krakatit
Story,TEST-2,This is another test,Open,,Person A,Normal,,2020-12-13 14:05:20,2020-12-13 14:07:20,"pat,mat"
`
	assert.Equal(t, expected, b.String())
}

func TestIssueDetailCallbacks(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	originalInstallation := viper.Get("installation")
	t.Cleanup(func() { viper.Set("installation", originalInstallation) })

	for _, installation := range []string{jira.InstallationTypeCloud, jira.InstallationTypeLocal} {
		t.Run(installation, func(t *testing.T) {
			viper.Set("installation", installation)
			var posted atomic.Bool
			const body = "First line\nSecond line\n\nThird paragraph"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				version := "3"
				if installation == jira.InstallationTypeLocal {
					version = "2"
				}
				switch r.Method {
				case http.MethodGet:
					if want := "/rest/api/" + version + "/issue/TEST-2"; r.URL.Path != want {
						t.Errorf("GET path = %q, want %q", r.URL.Path, want)
					}
					payload := issueDetailPayload(installation, posted.Load())
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(payload); err != nil {
						t.Errorf("encode issue: %v", err)
					}
				case http.MethodPost:
					if r.URL.Path != "/rest/api/2/issue/TEST-2/comment" {
						t.Errorf("POST path = %q", r.URL.Path)
					}
					var request struct {
						Body string `json:"body"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Errorf("decode comment: %v", err)
					}
					if request.Body != body {
						t.Errorf("comment = %q, want %q", request.Body, body)
					}
					posted.Store(true)
					w.WriteHeader(http.StatusCreated)
				default:
					t.Errorf("unexpected method %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()

			list := IssueList{Server: server.URL, Data: getIssues(), Display: DisplayFormat{Comments: 1}}
			client := jira.NewClient(jira.Config{Server: server.URL})
			detail := list.detailView(client, list.Data[1].Key)
			list.Data[1].Key = "OTHER-1"
			if detail.Title != "TEST-2" {
				t.Fatalf("detail title = %q, want selected issue", detail.Title)
			}
			out, err := detail.Load()
			if err != nil {
				t.Fatalf("load details: %v", err)
			}
			if !strings.Contains(out, "Selected ticket") || !strings.Contains(out, "Existing comment") {
				t.Errorf("details omitted summary or comment: %q", out)
			}
			if err := detail.SubmitComment(body); err != nil {
				t.Fatalf("submit comment: %v", err)
			}
			out, err = detail.Load()
			if err != nil || !strings.Contains(out, "Fresh comment") {
				t.Fatalf("reload did not show new comment: output %q, error %v", out, err)
			}
			list.Display.Comments = 0
			out, err = detail.Load()
			if err != nil || strings.Contains(out, "Fresh comment") {
				t.Errorf("zero comment limit not respected: output %q, error %v", out, err)
			}
		})
	}
}

func TestIssueDetailCallbackErrors(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	list := IssueList{Server: server.URL}
	detail := list.detailView(jira.NewClient(jira.Config{Server: server.URL}), "TEST-2")

	_, loadErr := detail.Load()
	postErr := detail.SubmitComment("comment")
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "load", err: loadErr, status: http.StatusForbidden},
		{name: "post", err: postErr, status: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var responseErr *jira.ErrUnexpectedResponse
			if !errors.As(tc.err, &responseErr) {
				t.Fatalf("error = %v, want wrapped Jira response error", tc.err)
			}
			if responseErr.StatusCode != tc.status || !strings.Contains(tc.err.Error(), "TEST-2") {
				t.Errorf("error lacks target/status: %v", tc.err)
			}
		})
	}
}

func issueDetailPayload(installation string, posted bool) map[string]any {
	text := "Existing comment"
	if posted {
		text = "Fresh comment"
	}
	var body any = text
	if installation == jira.InstallationTypeCloud {
		body = map[string]any{
			"type": "doc", "version": 1,
			"content": []any{map[string]any{
				"type":    "paragraph",
				"content": []any{map[string]any{"type": "text", "text": text}},
			}},
		}
	}
	return map[string]any{
		"key": "TEST-2",
		"fields": map[string]any{
			"summary": "Selected ticket",
			"comment": map[string]any{
				"total": 1,
				"comments": []any{map[string]any{
					"body":    body,
					"author":  map[string]any{"displayName": "Tester"},
					"created": "2020-12-13T14:05:20.974+0100",
				}},
			},
		},
	}
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
