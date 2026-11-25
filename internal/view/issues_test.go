package view

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
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
	newRenderer := newMDRendererFactory()
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
			detail := list.detailView(client, list.Data[1].Key, newRenderer)
			list.Data[1].Key = "OTHER-1"
			if detail.Title != "TEST-2" {
				t.Fatalf("detail title = %q, want selected issue", detail.Title)
			}
			out, err := detail.Load()
			if err != nil {
				t.Fatalf("load details: %v", err)
			}
			assertReplyUnavailable(t, out, "--comment-threads")
			assertReplyPostingGated(t, detail)
			if !strings.Contains(out.Text, "Selected ticket") || !strings.Contains(out.Text, "Existing comment") {
				t.Errorf("details omitted summary or comment: %q", out.Text)
			}
			if err := detail.SubmitComment(body); err != nil {
				t.Fatalf("submit comment: %v", err)
			}
			out, err = detail.Load()
			if err != nil || !strings.Contains(out.Text, "Fresh comment") {
				t.Fatalf("reload did not show new comment: output %q, error %v", out.Text, err)
			}
			list.Display.Comments = 0
			out, err = detail.Load()
			if err != nil || strings.Contains(out.Text, "Fresh comment") {
				t.Errorf("zero comment limit not respected: output %q, error %v", out.Text, err)
			}
		})
	}
}

func TestIssueDetailCallbackErrors(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	newRenderer := newMDRendererFactory()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	list := IssueList{Server: server.URL}
	detail := list.detailView(jira.NewClient(jira.Config{Server: server.URL}), "TEST-2", newRenderer)

	content, loadErr := detail.Load()
	assertZeroDetailContent(t, content)
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

func TestIssueDetailReplyCallbackAvailability(t *testing.T) {
	t.Parallel()
	newRenderer := newMDRendererFactory()
	for _, tc := range []struct {
		name    string
		threads bool
		limit   uint
		enabled bool
	}{
		{name: "flat", limit: 1},
		{name: "flat zero"},
		{name: "native zero", threads: true},
		{name: "native", threads: true, limit: 1, enabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			list := IssueList{Display: DisplayFormat{CommentThreads: tc.threads, Comments: tc.limit}}
			detail := list.detailView(nil, "TEST-2", newRenderer)
			if got := detail.SubmitReply != nil; got != tc.enabled {
				t.Errorf("reply callback enabled = %v, want %v", got, tc.enabled)
			}
		})
	}
}

func TestIssueDetailSubmitReply(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	newRenderer := newMDRendererFactory()
	setNavigationCloud(t)
	const body = "Native reply from the detail view"
	var writes atomic.Int64
	server := httptest.NewTLSServer(threadReplyHandler(t, body, &writes))
	defer server.Close()
	list := IssueList{Data: getIssues(), Display: DisplayFormat{CommentThreads: true, Comments: 1}}
	client := jira.NewClient(jira.Config{Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-test-token"}, jira.WithInsecureTLS(true))
	detail := list.detailView(client, list.Data[1].Key, newRenderer)
	t.Setenv("GLAMOUR_STYLE", "missing-style-after-capture")
	list.Data[1].Key = "OTHER-1"
	if detail.SubmitReply == nil {
		t.Fatal("native reply callback is missing")
	}
	if err := detail.SubmitReply("100", body); err != nil {
		t.Fatalf("submit native reply: %v", err)
	}
	if _, err := detail.Load(); err != nil {
		t.Fatalf("reload after native reply lost the captured renderer: %v", err)
	}
	if writes.Load() != 1 {
		t.Errorf("native reply writes = %d, want exactly one", writes.Load())
	}
}

func TestIssueDetailReplyPreflightError(t *testing.T) {
	t.Parallel()
	newRenderer := newMDRendererFactory()
	for _, root := range []string{"100", "unvalidated-root"} {
		t.Run(root, func(t *testing.T) {
			t.Parallel()
			var writes atomic.Int64
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					writes.Add(1)
				}
				w.WriteHeader(http.StatusForbidden)
			}))
			defer server.Close()
			list := IssueList{Display: DisplayFormat{Comments: 1, CommentThreads: true}}
			client := jira.NewClient(jira.Config{Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-test-token"}, jira.WithInsecureTLS(true))
			detail := list.detailView(client, "TEST-2", newRenderer)
			if detail.SubmitReply == nil {
				t.Fatal("native reply callback is missing")
			}
			const body = "Synthetic reply body"
			err := detail.SubmitReply(root, body)
			if err == nil || errors.Unwrap(err) == nil || !strings.Contains(err.Error(), "reply to native thread on TEST-2") {
				t.Fatalf("reply preflight error lacks context/wrapping: %v", err)
			}
			if strings.Contains(err.Error(), body) || strings.Contains(err.Error(), root) {
				t.Errorf("reply error exposes body or unvalidated root: %v", err)
			}
			if writes.Load() != 0 {
				t.Fatal("failed reply preflight must not issue native or top-level writes")
			}
		})
	}
}

func TestIssueDetailRendererErrorClearsContent(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "missing-test-style")
	newRenderer := newMDRendererFactory()
	list := IssueList{Display: DisplayFormat{Comments: 1, CommentThreads: true}}
	content, err := list.detailView(nil, "TEST-2", newRenderer).Load()
	assertZeroDetailContent(t, content)
	if err == nil || !strings.Contains(err.Error(), "initialize issue renderer") {
		t.Fatalf("expected renderer initialization error, got %v", err)
	}
}

func TestIssueDetailThreadCallbacks(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	newRenderer := newMDRendererFactory()
	originalInstallation := viper.Get("installation")
	viper.Set("installation", jira.InstallationTypeCloud)
	t.Cleanup(func() { viper.Set("installation", originalInstallation) })
	var failThreads atomic.Bool
	var threadRequests atomic.Int64
	server := httptest.NewTLSServer(threadDetailHandler(t, &failThreads, &threadRequests))
	defer server.Close()
	list := IssueList{Server: server.URL, Display: DisplayFormat{Comments: 1, CommentThreads: true}}
	client := jira.NewClient(jira.Config{Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-test-token"}, jira.WithInsecureTLS(true))
	detail := list.detailView(client, "TEST-2", newRenderer)
	out, err := detail.Load()
	if err != nil {
		t.Fatalf("load threaded details: %v", err)
	}
	assertThreadReplyTarget(t, detail, out)
	if threadRequests.Load() != 1 {
		t.Fatal("detail text and reply targets must share one thread fetch")
	}
	for _, want := range []string{"Selected ticket", "Native root", "Native reply", "Reply 101 to 100", "Showing 1 of 1 threads"} {
		if !strings.Contains(out.Text, want) {
			t.Errorf("threaded details missing %q: %q", want, out.Text)
		}
	}
	if strings.Contains(out.Text, "Existing comment") {
		t.Fatal("threaded details used the embedded flat comment instead of thread data")
	}
	failThreads.Store(true)
	out, err = detail.Load()
	assertZeroDetailContent(t, out)
	if err == nil || !strings.Contains(err.Error(), "TEST-2") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("thread failure was hidden or flat fallback used: output %q, error %v", out.Text, err)
	}
	before := threadRequests.Load()
	list.Display.Comments = 0
	assertReplyPostingGated(t, list.detailView(client, "TEST-2", newRenderer))
	out, err = detail.Load()
	if err != nil || threadRequests.Load() != before || strings.Contains(out.Text, "Native reply") || strings.Contains(out.Text, "Existing comment") {
		t.Fatalf("zero limit fetched/displayed thread content: output %q, error %v", out.Text, err)
	}
	assertReplyUnavailable(t, out, "greater than 0")
	if !strings.Contains(out.Text, "Showing 0 threads") || !strings.Contains(out.Text, "thread total not loaded") {
		t.Errorf("zero-limit footer invented a thread total: %q", out.Text)
	}
}

func assertZeroDetailContent(t *testing.T, content tui.DetailContent) {
	t.Helper()
	if !reflect.DeepEqual(content, tui.DetailContent{}) {
		t.Errorf("failed load returned nonzero detail content: %+v", content)
	}
}

func assertReplyUnavailable(t *testing.T, content tui.DetailContent, reason string) {
	t.Helper()
	if len(content.ReplyTargets) != 0 || !strings.Contains(content.ReplyUnavailable, reason) {
		t.Errorf("unexpected reply availability: %+v", content)
	}
}

func assertReplyPostingGated(t *testing.T, detail tui.DetailView) {
	t.Helper()
	if detail.SubmitReply != nil {
		t.Fatal("native reply posting must remain disabled")
	}
}

func assertThreadReplyTarget(t *testing.T, detail tui.DetailView, content tui.DetailContent) {
	t.Helper()
	if detail.SubmitReply == nil {
		t.Fatal("native mode with loaded threads must enable the reply callback")
	}
	want := []tui.ReplyTarget{{ID: "100", Author: "Root author", Created: "Thu, 01 Jan 26", Preview: "Native root"}}
	if !reflect.DeepEqual(content.ReplyTargets, want) || content.ReplyUnavailable != "" {
		t.Fatalf("picker targets do not match the rendered native root: %+v", content)
	}
	if !strings.Contains(content.Text, content.ReplyTargets[0].Preview) {
		t.Error("picker preview does not match rendered comment body")
	}
}

func threadDetailHandler(t *testing.T, fail *atomic.Bool, requests *atomic.Int64) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const issueID = "ari:cloud:jira:view-test-cloud:issue/10"
		var payload any
		switch {
		case r.URL.Path == "/browse/TEST-2":
			requests.Add(1)
			if fail.Load() {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if _, err := fmt.Fprint(w, `<meta name="ajs-cloud-id" content="view-test-cloud">`); err != nil {
				t.Errorf("write metadata: %v", err)
			}
			return
		case r.URL.Path == "/rest/api/3/issue/TEST-2":
			baseIssue := issueDetailPayload(jira.InstallationTypeCloud, false)
			baseIssue["id"] = "10"
			baseIssue["fields"].(map[string]any)["project"] = map[string]any{"projectTypeKey": "software"}
			payload = baseIssue
		case strings.HasPrefix(r.URL.Path, "/gateway/api/graphql/pq/"):
			root := map[string]any{
				"id": "native-root", "commentId": "100",
				"repliesPaginated": threadDetailConnection(map[string]any{"id": "native-reply", "commentId": "101"}),
			}
			payload = map[string]any{"data": map[string]any{"node": map[string]any{
				"id": issueID, "key": "TEST-2", "commentsPaginated": threadDetailConnection(root),
			}}}
		case r.URL.Path == "/rest/api/3/issue/TEST-2/comment":
			comments := []map[string]any{
				{"id": "100", "author": map[string]any{"displayName": "Root author", "name": "root-account"}, "created": "2026-01-01T01:00:00.000+0000", "body": threadDetailBody("Native root")},
				{"id": "101", "created": "2026-01-02T01:00:00.000+0000", "body": threadDetailBody("Native reply")},
			}
			payload = map[string]any{"startAt": 0, "total": len(comments), "comments": comments}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("encode threaded issue: %v", err)
		}
	})
}

func threadReplyHandler(t *testing.T, wantBody string, writes *atomic.Int64) http.Handler {
	t.Helper()
	reads := threadDetailHandler(t, new(atomic.Bool), new(atomic.Int64))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload any
		switch {
		case r.Method == http.MethodPost:
			writes.Add(1)
			if !strings.HasPrefix(r.URL.Path, "/gateway/api/graphql/pq/") || r.URL.Query().Get("operation") != "useSaveCommentRelayAddCommentMutation" {
				t.Errorf("unexpected write or top-level fallback: %s", r.URL.Path)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var request struct {
				OperationName string `json:"operationName"`
				Variables     struct {
					Input struct {
						IssueID        string `json:"issueId"`
						ThreadParentID string `json:"threadParentId"`
						Content        struct {
							JSONValue *adf.ADF `json:"jsonValue"`
						} `json:"content"`
					} `json:"input"`
				} `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode native reply mutation: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			input := request.Variables.Input
			if request.OperationName != "useSaveCommentRelayAddCommentMutation" || input.IssueID != "ari:cloud:jira:view-test-cloud:issue/10" || input.ThreadParentID != "100" || input.Content.JSONValue == nil {
				t.Errorf("reply callback used the wrong issue, root, or body: %+v", request)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			gotBody := adf.NewTranslator(input.Content.JSONValue, adf.NewMarkdownTranslator()).Translate()
			if strings.TrimSpace(gotBody) != wantBody {
				t.Errorf("reply body = %q, want %q", gotBody, wantBody)
			}
			payload = map[string]any{"data": map[string]any{"jira": map[string]any{"addComment": map[string]any{
				"success": true, "errors": []any{}, "comment": map[string]any{"commentId": "102"},
			}}}}
		case r.URL.Path == "/rest/api/3/issue/TEST-2/comment/100":
			payload = map[string]any{"id": "100", "visibility": nil, "properties": []any{}}
		case r.URL.Path == "/rest/api/3/issue/TEST-2/comment/102":
			payload = map[string]any{"id": "102", "parentId": "100", "visibility": nil, "properties": []any{}}
		default:
			reads.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("encode native reply response: %v", err)
		}
	})
}

func threadDetailConnection(node map[string]any) map[string]any {
	return map[string]any{
		"edges": []any{map[string]any{"cursor": "only-cursor", "node": node}},
		"pageInfo": map[string]any{
			"hasNextPage": false, "hasPreviousPage": false, "startCursor": "only-cursor", "endCursor": "only-cursor",
		},
	}
}

func threadDetailBody(text string) map[string]any {
	return map[string]any{
		"type": "doc", "version": 1,
		"content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": text}}}},
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
