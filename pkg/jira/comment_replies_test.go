package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/md"
)

const testReplyMarkdown = "Synthetic **reply**\nnext\tline"

const (
	testReplyRoot = "11"
	testReplyID   = "31"
)

type replyFixture struct {
	t            *testing.T
	threads      *threadFixture
	rootNode     *threadNode
	project      map[string]any
	root         map[string]any
	confirmation map[string]any
	mutation     map[string]any
	posts        atomic.Int32
	requests     atomic.Int32
	intercept    func(http.ResponseWriter, *http.Request) bool
	post         func(http.ResponseWriter, *http.Request)
}

func testReplyResult(success bool, id string) map[string]any {
	return map[string]any{"data": map[string]any{"jira": map[string]any{"addComment": map[string]any{
		"success": success, "errors": nil, "comment": map[string]any{"commentId": id},
	}}}}
}

func newReplyFixture(t *testing.T) *replyFixture {
	t.Helper()
	threads := newThreadFixture(t)
	root := testThreadComment(testReplyRoot)
	root.ID = "ari:cloud:jira:test-cloud:issue-comment/7/11"
	root.RepliesPaginated = testThreadConnection(false, false)
	threads.roots = map[string]*threadConnection{"": testThreadConnection(false, false, root)}
	return &replyFixture{
		t: t, threads: threads, rootNode: root,
		project:      map[string]any{"id": "7", "key": "TEST-1", "fields": map[string]any{"project": map[string]any{"projectTypeKey": "software"}}},
		root:         map[string]any{"id": testReplyRoot, "properties": []any{map[string]any{"key": "sd.public.comment", "value": map[string]any{"internal": false}}}},
		confirmation: map[string]any{"id": testReplyID, "parentId": 11, "properties": []any{}},
		mutation:     testReplyResult(true, testReplyID),
	}
}

func (f *replyFixture) serve(w http.ResponseWriter, r *http.Request) {
	f.requests.Add(1)
	if r.Method == http.MethodPost {
		f.posts.Add(1)
	}
	if f.intercept != nil && f.intercept(w, r) {
		return
	}
	login, token, ok := r.BasicAuth()
	if !ok || login != "test-login" || token != "test-token" || r.TLS == nil {
		f.t.Error("reply request did not use configured Basic auth over TLS")
	}
	switch {
	case r.Method == http.MethodPost:
		f.checkPayload(r)
		if f.post != nil {
			f.post(w, r)
			return
		}
		f.threads.writeJSON(w, f.mutation)
	case r.URL.Path == "/rest/api/3/issue/TEST-1":
		if r.URL.Query().Get("fields") != "project" {
			f.t.Error("reply preparation did not request fresh project metadata")
		}
		f.threads.writeJSON(w, f.project)
	case r.URL.Path == "/rest/api/3/issue/TEST-1/comment/"+testReplyRoot:
		f.checkCommentRead(r)
		f.threads.writeJSON(w, f.root)
	case r.URL.Path == "/rest/api/3/issue/TEST-1/comment/"+testReplyID:
		f.checkCommentRead(r)
		if f.posts.Load() != 1 {
			f.t.Error("reply confirmation was not after exactly one POST")
		}
		f.threads.writeJSON(w, f.confirmation)
	default:
		f.threads.serve(w, r)
	}
}

func (f *replyFixture) checkCommentRead(r *http.Request) {
	f.t.Helper()
	if r.Method != http.MethodGet || r.URL.Query().Get("expand") != "properties" {
		f.t.Error("comment privacy properties were not freshly requested")
	}
}

func (f *replyFixture) checkPayload(r *http.Request) {
	f.t.Helper()
	if r.URL.Path != threadAddRouteV1 || r.URL.Query().Get("operation") != threadAddQueryV1 || r.Header.Get("Content-Type") != "application/json" {
		f.t.Error("unexpected reply write route, operation, or content type")
	}
	if !strings.Contains(r.Header.Get("Accept"), "application/graphql-response+json") {
		f.t.Error("missing GraphQL Accept")
	}
	var got map[string]any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		f.t.Errorf("decode synthetic request: %v", err)
		return
	}
	doc, err := md.ToADF(testReplyMarkdown)
	if err != nil {
		f.t.Error(err)
		return
	}
	want := map[string]any{
		"operationName": threadAddQueryV1,
		"variables": map[string]any{
			"input": map[string]any{
				"issueId": testThreadIssueID, "threadParentId": testReplyRoot,
				"content":         map[string]any{"version": 1, "jsonValue": doc},
				"permissionLevel": nil, "eventOccurredAt": nil,
			},
			"connections":   []string{"client:" + f.rootNode.ID + ":::::JiraPlatformComment:__commentRepliesListRelay_repliesPaginated_connection"},
			"isRootComment": false, "includeRedactedContent": false,
			threadMentionVarV1: true, threadCommentsVarV1: false,
		},
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		f.t.Error(err)
		return
	}
	var normalized map[string]any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		f.t.Error(err)
		return
	}
	if !reflect.DeepEqual(got, normalized) {
		f.t.Error("native reply payload differs from the verified contract or does not contain an ADF object")
	}
}

func TestReplyToIssueThread(t *testing.T) {
	for _, projectType := range []string{"software", "business"} {
		for _, parent := range []any{11, "11"} {
			t.Run(projectType+"/"+reflect.TypeOf(parent).String(), func(t *testing.T) {
				f := newReplyFixture(t)
				f.project["fields"].(map[string]any)["project"].(map[string]any)["projectTypeKey"] = projectType
				f.confirmation["parentId"] = parent
				client := testThreadClient(t, f.serve)
				if err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown); err != nil {
					t.Fatal(err)
				}
				if f.posts.Load() != 1 || f.requests.Load() != 6 || f.threads.rootCalls.Load() != 1 {
					t.Fatal("reply did not use fresh native metadata, exactly one POST, and confirmation")
				}
			})
		}
	}
}

func TestReplyToIssueThreadRejectsBeforeHTTP(t *testing.T) {
	cases := []struct{ name, key, root, body string }{
		{"empty", "TEST-1", "11", ""},
		{"whitespace", "TEST-1", "11", " \t\n"},
		{"bad key", "../TEST-1", "11", testReplyMarkdown},
		{"missing ID", "TEST-1", "", testReplyMarkdown},
		{"zero ID", "TEST-1", "0", testReplyMarkdown},
		{"negative ID", "TEST-1", "-1", testReplyMarkdown},
		{"fraction ID", "TEST-1", "11.0", testReplyMarkdown},
		{"noncanonical ID", "TEST-1", "011", testReplyMarkdown},
		{"overflow ID", "TEST-1", "18446744073709551616", testReplyMarkdown},
		{"unsupported Markdown", "TEST-1", "11", "<div>PRIVATE_SENTINEL</div>"},
		{"invalid UTF8", "TEST-1", "11", string([]byte{0xff})},
		{"oversized body", "TEST-1", "11", strings.Repeat("x", replySourceByteLimit+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReplyFixture(t)
			err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), tc.key, tc.root, tc.body)
			if err == nil || errors.Is(err, ErrReplyOutcomeUnknown) || f.requests.Load() != 0 || strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
				t.Fatalf("invalid input reached HTTP or leaked input: %v", err)
			}
		})
	}
	for _, auth := range []AuthType{AuthTypeBearer, AuthTypeMTLS, AuthTypeCFAccess, "unsupported"} {
		f := newReplyFixture(t)
		client := testThreadClient(t, f.serve)
		client.authType = &auth
		if err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown); err == nil || f.requests.Load() != 0 {
			t.Fatal("unsupported auth reached the network")
		}
	}
	var requests atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer plain.Close()
	client := NewClient(Config{Server: plain.URL, Login: "test-login", APIToken: "test-token"})
	if err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown); err == nil || requests.Load() != 0 {
		t.Fatal("HTTP reply origin was not rejected before any request")
	}
}

func TestReplyToIssueThreadSourceBudget(t *testing.T) {
	if replySourceByteLimit >= threadResponseLimit {
		t.Fatal("Markdown source budget must remain separate from the larger serialized payload cap")
	}
	cases := []struct {
		name, body string
		wantHTTP   int32
		wantError  string
	}{
		{"at budget", strings.Repeat("x", replySourceByteLimit), 1, "403"},
		{"over budget", strings.Repeat("x", replySourceByteLimit+1), 0, "source exceeds"},
		{"over budget whitespace", strings.Repeat(" ", replySourceByteLimit+1), 0, "source exceeds"},
		{"over budget unsupported Markdown", "<div>" + strings.Repeat("x", replySourceByteLimit), 0, "source exceeds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet {
					t.Error("source boundary test must stop at the first preflight GET")
				}
				w.WriteHeader(http.StatusForbidden)
			})
			err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, tc.body)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) || errors.Is(err, ErrReplyOutcomeUnknown) || requests.Load() != tc.wantHTTP {
				t.Fatalf("source budget did not reject before parsing/HTTP or rejected the boundary: %v; requests=%d", err, requests.Load())
			}
		})
	}
}

func TestReplyToIssueThreadPreflightGuards(t *testing.T) {
	cases := []struct {
		name   string
		change func(*replyFixture)
	}{
		{"service desk", func(f *replyFixture) {
			f.project["fields"].(map[string]any)["project"] = map[string]any{"projectTypeKey": "service_desk"}
		}},
		{"unknown project", func(f *replyFixture) { f.project["fields"] = map[string]any{} }},
		{"wrong issue", func(f *replyFixture) { f.project["key"] = "TEST-2" }},
		{"missing issue ID", func(f *replyFixture) { delete(f.project, "id") }},
		{"missing root", func(f *replyFixture) { f.rootNode.CommentID = "12" }},
		{"selected reply", func(f *replyFixture) {
			root := testThreadComment("21")
			root.RepliesPaginated = testThreadConnection(false, false, f.rootNode)
			f.threads.roots[""] = testThreadConnection(false, false, root)
		}},
		{"REST root mismatch", func(f *replyFixture) { f.root["id"] = "12" }},
		{"root has parent", func(f *replyFixture) { f.root["parentId"] = 21 }},
		{"root invalid parent", func(f *replyFixture) { f.root["parentId"] = "" }},
		{"role restriction", func(f *replyFixture) { f.root["visibility"] = map[string]any{"type": "role", "value": "synthetic"} }},
		{"group restriction", func(f *replyFixture) { f.root["visibility"] = map[string]any{"type": "group", "value": "synthetic"} }},
		{"empty visibility", func(f *replyFixture) { f.root["visibility"] = map[string]any{} }},
		{"malformed visibility", func(f *replyFixture) { f.root["visibility"] = "PRIVATE_SENTINEL" }},
		{"missing properties", func(f *replyFixture) { delete(f.root, "properties") }},
		{"null properties", func(f *replyFixture) { f.root["properties"] = nil }},
		{"malformed properties", func(f *replyFixture) { f.root["properties"] = map[string]any{} }},
		{"internal", func(f *replyFixture) { f.root["properties"] = testReplyPrivacy(map[string]any{"internal": true}) }},
		{"invalid internal flag", func(f *replyFixture) { f.root["properties"] = testReplyPrivacy(map[string]any{"internal": "false"}) }},
		{"missing internal flag", func(f *replyFixture) { f.root["properties"] = testReplyPrivacy(map[string]any{}) }},
		{"null internal flag", func(f *replyFixture) { f.root["properties"] = testReplyPrivacy(map[string]any{"internal": nil}) }},
		{"unknown privacy flag", func(f *replyFixture) {
			f.root["properties"] = testReplyPrivacy(map[string]any{"internal": false, "other": false})
		}},
		{"duplicate internal flag", func(f *replyFixture) {
			f.root["properties"] = testReplyPrivacy(json.RawMessage(`{"internal":true,"internal":false}`))
		}},
		{"duplicate privacy property", func(f *replyFixture) {
			f.root["properties"] = append(testReplyPrivacy(map[string]any{"internal": false}), testReplyPrivacy(map[string]any{"internal": false})...)
		}},
		{"duplicate property value", func(f *replyFixture) {
			f.root["properties"] = json.RawMessage(`[{"key":"sd.public.comment","value":{"internal":true},"value":{"internal":false}}]`)
		}},
		{"duplicate visibility", func(f *replyFixture) {
			f.intercept = func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path != "/rest/api/3/issue/TEST-1/comment/11" {
					return false
				}
				_, _ = io.WriteString(w, `{"id":"11","visibility":{"type":"group","value":"synthetic"},"visibility":null,"properties":[]}`)
				return true
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReplyFixture(t)
			tc.change(f)
			err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
			if err == nil || errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 0 || strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
				t.Fatalf("preflight did not fail closed before POST: %v", err)
			}
		})
	}
}

func testReplyPrivacy(value any) []any {
	return []any{map[string]any{"key": "sd.public.comment", "value": value}}
}

func TestReplyToIssueThreadPreflightHTTPFailures(t *testing.T) {
	for _, path := range []string{"/rest/api/3/issue/TEST-1", "/browse/TEST-1", threadRootsRouteV1, "/rest/api/3/issue/TEST-1/comment/11"} {
		t.Run(path, func(t *testing.T) {
			f := newReplyFixture(t)
			f.intercept = func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path != path {
					return false
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, "PRIVATE_SENTINEL")
				return true
			}
			err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
			if err == nil || errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 0 || strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
				t.Fatalf("preflight failure was ignored or classified as a possible write: %v", err)
			}
		})
	}
}

func TestReplyToIssueThreadFreshPrivacy(t *testing.T) {
	f := newReplyFixture(t)
	client := testThreadClient(t, f.serve)
	if err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown); err != nil {
		t.Fatal(err)
	}
	f.root["visibility"] = map[string]any{"type": "group", "value": "synthetic"}
	if err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown); err == nil || f.posts.Load() != 1 || f.threads.rootCalls.Load() != 2 {
		t.Fatal("reply trusted stale root or privacy metadata")
	}
}

func TestReplyToIssueThreadMutationFailures(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		unknown bool
	}{
		{"unauthorized", http.StatusUnauthorized, "PRIVATE_SENTINEL", false},
		{"forbidden", http.StatusForbidden, "PRIVATE_SENTINEL", false},
		{"server error", http.StatusInternalServerError, "PRIVATE_SENTINEL", true},
		{"bad JSON", http.StatusOK, `{"PRIVATE_SENTINEL":`, true},
		{"missing data", http.StatusOK, `{}`, true},
		{"GraphQL errors", http.StatusOK, `{"errors":[{"message":"PRIVATE_SENTINEL"}]}`, true},
		{"partial errors", http.StatusOK, `{"errors":[{"message":"PRIVATE_SENTINEL"}],"data":{"jira":{"addComment":{"success":true,"errors":null,"comment":{"commentId":"31"}}}}}`, true},
		{"business rejection", http.StatusOK, `{"data":{"jira":{"addComment":{"success":false,"errors":[{"message":"PRIVATE_SENTINEL"}],"comment":null}}}}`, false},
		{"non-success", http.StatusOK, `{"data":{"jira":{"addComment":{"success":false,"errors":null,"comment":null}}}}`, false},
		{"inconsistent success", http.StatusOK, `{"data":{"jira":{"addComment":{"success":true,"errors":[{"message":"PRIVATE_SENTINEL"}],"comment":{"commentId":"31"}}}}}`, true},
		{"missing success", http.StatusOK, `{"data":{"jira":{"addComment":{"errors":null,"comment":{"commentId":"31"}}}}}`, true},
		{"missing errors", http.StatusOK, `{"data":{"jira":{"addComment":{"success":true,"comment":{"commentId":"31"}}}}}`, true},
		{"missing ID", http.StatusOK, `{"data":{"jira":{"addComment":{"success":true,"errors":null,"comment":{}}}}}`, true},
		{"invalid ID", http.StatusOK, `{"data":{"jira":{"addComment":{"success":true,"errors":null,"comment":{"commentId":"PRIVATE_SENTINEL"}}}}}`, true},
		{"root ID returned", http.StatusOK, `{"data":{"jira":{"addComment":{"success":true,"errors":null,"comment":{"commentId":"11"}}}}}`, true},
		{"duplicate success", http.StatusOK, `{"data":{"jira":{"addComment":{"success":false,"success":true,"errors":null,"comment":{"commentId":"31"}}}}}`, true},
		{"response cap", http.StatusOK, strings.Repeat(" ", threadResponseLimit+1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReplyFixture(t)
			f.post = func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}
			err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
			if err == nil || errors.Is(err, ErrReplyOutcomeUnknown) != tc.unknown || f.posts.Load() != 1 || strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
				t.Fatalf("unexpected mutation error classification, retry, or unsafe message: %v", err)
			}
		})
	}
}

func TestReplyToIssueThreadConfirmationFailures(t *testing.T) {
	cases := []struct {
		name   string
		change func(*replyFixture)
	}{
		{"wrong parent", func(f *replyFixture) { f.confirmation["parentId"] = 21 }},
		{"missing parent", func(f *replyFixture) { delete(f.confirmation, "parentId") }},
		{"null parent", func(f *replyFixture) { f.confirmation["parentId"] = nil }},
		{"zero parent", func(f *replyFixture) { f.confirmation["parentId"] = 0 }},
		{"float parent", func(f *replyFixture) { f.confirmation["parentId"] = json.RawMessage(`11.0`) }},
		{"exponent parent", func(f *replyFixture) { f.confirmation["parentId"] = json.RawMessage(`1.1e1`) }},
		{"noncanonical parent", func(f *replyFixture) { f.confirmation["parentId"] = "011" }},
		{"boolean parent", func(f *replyFixture) { f.confirmation["parentId"] = true }},
		{"missing ID", func(f *replyFixture) { delete(f.confirmation, "id") }},
		{"wrong ID", func(f *replyFixture) { f.confirmation["id"] = "32" }},
		{"unexpected visibility", func(f *replyFixture) {
			f.confirmation["visibility"] = map[string]any{"type": "group", "value": "synthetic"}
		}},
		{"missing properties", func(f *replyFixture) { delete(f.confirmation, "properties") }},
		{"unexpected internal", func(f *replyFixture) {
			f.confirmation["properties"] = testReplyPrivacy(map[string]any{"internal": true})
		}},
		{"HTTP failure", func(f *replyFixture) {
			f.intercept = func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path != "/rest/api/3/issue/TEST-1/comment/31" {
					return false
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, "PRIVATE_SENTINEL")
				return true
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReplyFixture(t)
			tc.change(f)
			err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
			if !errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 1 || !strings.Contains(err.Error(), "returned comment 31") || !strings.Contains(err.Error(), "check Jira before retrying") || strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
				t.Fatalf("unconfirmed reply was not reported safely with its ID: %v", err)
			}
		})
	}
}

func TestReplyToIssueThreadPostRedirect(t *testing.T) {
	var forwarded atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded.Add(1) }))
	defer target.Close()
	f := newReplyFixture(t)
	f.post = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}
	err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
	if !errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 1 || forwarded.Load() != 0 {
		t.Fatalf("POST redirect followed or retried: %v", err)
	}
}

func TestReplyToIssueThreadCancellation(t *testing.T) {
	t.Run("before HTTP", func(t *testing.T) {
		f := newReplyFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := testThreadClient(t, f.serve).ReplyToIssueThread(ctx, "TEST-1", testReplyRoot, testReplyMarkdown)
		if !errors.Is(err, context.Canceled) || errors.Is(err, ErrReplyOutcomeUnknown) || f.requests.Load() != 0 {
			t.Fatalf("pre-dispatch cancellation: %v", err)
		}
	})
	t.Run("after dispatch", func(t *testing.T) {
		f := newReplyFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f.post = func(w http.ResponseWriter, r *http.Request) { cancel(); <-r.Context().Done() }
		err := testThreadClient(t, f.serve).ReplyToIssueThread(ctx, "TEST-1", testReplyRoot, testReplyMarkdown)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 1 {
			t.Fatalf("post-dispatch cancellation lost cause, uncertainty, or retried: %v", err)
		}
	})
	t.Run("after dispatch timeout", func(t *testing.T) {
		f := newReplyFixture(t)
		f.post = func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }
		client := testThreadClient(t, f.serve)
		client.timeout = time.Second / 2
		err := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
		if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 1 {
			t.Fatalf("post-dispatch timeout lost cause, uncertainty, or retried: %v", err)
		}
	})
}

func TestReplyToIssueThreadNetworkOutcome(t *testing.T) {
	f := newReplyFixture(t)
	f.post = func(w http.ResponseWriter, r *http.Request) {
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	}
	err := testThreadClient(t, f.serve).ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
	if !errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 1 {
		t.Fatalf("connection loss after POST was retried or misclassified: %v", err)
	}
}

func TestReplyPostBudgetAndCause(t *testing.T) {
	f := newReplyFixture(t)
	client := testThreadClient(t, f.serve)
	r, err := client.newThreadReader()
	if err != nil {
		t.Fatal(err)
	}
	r.requests = threadRequestLimit
	if _, err := r.postReply(context.Background(), []byte(`{}`)); err == nil || errors.Is(err, ErrReplyOutcomeUnknown) || f.posts.Load() != 0 {
		t.Fatal("request budget exhaustion was not a known pre-dispatch failure")
	}
	cause := errors.New("synthetic network cause")
	wrapped := unknownReply(testReplyID, fmt.Errorf("safe request failure: %w", threadRequestError{cause: cause}))
	if !errors.Is(wrapped, ErrReplyOutcomeUnknown) || !errors.Is(wrapped, cause) || strings.Contains(wrapped.Error(), cause.Error()) {
		t.Fatal("unknown reply errors must preserve causes without displaying transport details")
	}
}

func TestReplyToIssueThreadNoReplayOrDebugDump(t *testing.T) {
	f := newReplyFixture(t)
	client := testThreadClient(t, f.serve)
	transport := &replyCheckingTransport{t: t, base: client.transport}
	client.transport = transport
	client.debug = true
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = read.Close() }()
	original := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = original }()
	callErr := client.ReplyToIssueThread(context.Background(), "TEST-1", testReplyRoot, testReplyMarkdown)
	_ = write.Close()
	output, err := io.ReadAll(read)
	if callErr != nil || err != nil || len(output) != 0 || transport.posts.Load() != 1 {
		t.Fatal("reply failed, emitted a debug dump, or did not use configured transport once")
	}
}

type replyCheckingTransport struct {
	t     *testing.T
	base  http.RoundTripper
	posts atomic.Int32
}

func (r *replyCheckingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost {
		r.posts.Add(1)
		if req.GetBody != nil || req.Header.Get("Idempotency-Key") != "" || req.Header.Get("X-Idempotency-Key") != "" {
			r.t.Error("reply request permits HTTP transport replay")
		}
	}
	return r.base.RoundTrip(req)
}
