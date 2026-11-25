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
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	issuefilter "github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

const testThreadIssueID = "ari:cloud:jira:test-cloud:issue/7"

func testThreadConnection(previous, next bool, nodes ...*threadNode) *threadConnection {
	conn := &threadConnection{
		Edges:    make([]threadEdge, 0, len(nodes)),
		PageInfo: &threadPageInfo{HasPreviousPage: &previous, HasNextPage: &next},
	}
	for _, node := range nodes {
		conn.Edges = append(conn.Edges, threadEdge{Cursor: "cursor-" + node.CommentID, Node: node})
	}
	if len(conn.Edges) > 0 {
		conn.PageInfo.StartCursor = conn.Edges[0].Cursor
		conn.PageInfo.EndCursor = conn.Edges[len(conn.Edges)-1].Cursor
	}
	return conn
}

func testThreadComment(id string) *threadNode {
	return &threadNode{ID: "global/" + id, CommentID: id}
}

func testThreadBody(id string) IssueComment {
	return IssueComment{
		ID: id, Author: User{AccountID: "synthetic"}, Created: "2000-01-01T00:00:00.000+0000",
		Body: &adf.ADF{Version: 1, DocType: "doc", Content: []*adf.Node{}},
	}
}

type threadFixture struct {
	t           *testing.T
	roots       map[string]*threadConnection
	replies     map[string]*threadConnection
	rest        map[int]*threadRESTPage
	intercept   func(http.ResponseWriter, *http.Request) bool
	replyNodeID string
	rootNodeID  string
	page        string
	issue       map[string]any
	rootCalls   atomic.Int32
	replyCalls  atomic.Int32
	restCalls   atomic.Int32
}

func newThreadFixture(t *testing.T) *threadFixture {
	t.Helper()
	root, other := testThreadComment("r1"), testThreadComment("r2")
	root.RepliesPaginated = testThreadConnection(false, true, testThreadComment("c1"))
	root.ChildComments = &threadConnection{Edges: []threadEdge{{Node: testThreadComment("c1")}}}
	other.RepliesPaginated = testThreadConnection(false, false)
	fields := &IssueFields{Description: testThreadBody("").Body}
	fields.Comment.Total = 17
	fields.Comment.Comments = []IssueComment{testThreadBody("embedded")}
	f := &threadFixture{
		t: t,
		roots: map[string]*threadConnection{
			"": testThreadConnection(false, true, root), "cursor-r1": testThreadConnection(true, false, other),
		},
		replies: map[string]*threadConnection{
			"cursor-c1": testThreadConnection(true, true, testThreadComment("c2")),
			"cursor-c2": testThreadConnection(true, false, testThreadComment("c3")),
		},
		rest:  make(map[int]*threadRESTPage),
		page:  `<html><meta content="test-cloud" name="ajs-cloud-id"></html>`,
		issue: map[string]any{"id": "7", "key": "TEST-1", "fields": fields},
	}
	ids := []string{"r2", "c1", "r1", "c2", "c3"}
	const chunkSize = 2
	for offset := 0; offset < len(ids); offset += chunkSize {
		start, total := offset, len(ids)
		page := &threadRESTPage{StartAt: &start, Total: &total, Comments: []IssueComment{}}
		for _, id := range ids[offset:min(offset+chunkSize, len(ids))] {
			page.Comments = append(page.Comments, testThreadBody(id))
		}
		f.rest[offset] = page
	}
	return f
}

func (f *threadFixture) serve(w http.ResponseWriter, r *http.Request) {
	if f.intercept != nil && f.intercept(w, r) {
		return
	}
	login, token, ok := r.BasicAuth()
	if !ok || login != "test-login" || token != "test-token" || r.Method != http.MethodGet {
		f.t.Error("unexpected request method or credentials")
	}
	if r.Header.Get("Content-Type") != "application/json" {
		f.t.Error("missing JSON Content-Type on GET")
	}
	switch r.URL.Path {
	case "/browse/TEST-1":
		_, _ = io.WriteString(w, f.page)
	case "/rest/api/3/issue/TEST-1":
		if r.URL.RawQuery != "" {
			f.t.Error("base issue identity should also bootstrap the native query")
		}
		f.writeJSON(w, f.issue)
	case threadRootsRouteV1, threadRepliesRouteV1:
		f.serveGraphQL(w, r)
	case "/rest/api/3/issue/TEST-1/comment":
		f.restCalls.Add(1)
		start, err := strconv.Atoi(r.URL.Query().Get("startAt"))
		if err != nil || r.URL.Query().Get("maxResults") != "100" {
			f.t.Error("invalid REST pagination request")
		}
		page, ok := f.rest[start]
		if !ok {
			http.Error(w, "unexpected REST cursor", http.StatusBadRequest)
			return
		}
		f.writeJSON(w, page)
	default:
		f.t.Errorf("unexpected request path %q", r.URL.Path)
		http.NotFound(w, r)
	}
}

func (f *threadFixture) serveGraphQL(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept"), "application/graphql-response+json") {
		f.t.Error("missing GraphQL Accept")
	}
	var variables map[string]any
	if err := json.Unmarshal([]byte(r.URL.Query().Get("variables")), &variables); err != nil {
		f.t.Error("invalid query variables")
	}
	for k, want := range map[string]bool{
		threadMentionVarV1: true, threadCommentsVarV1: false,
		"issueViewSlackSyncInfoEnabled": true, "shouldFetchAgentAttribution": false,
	} {
		if variables[k] != want {
			f.t.Errorf("incorrect provider variable %s", k)
		}
	}
	after, _ := variables["after"].(string)
	node := &threadNode{}
	if r.URL.Path == threadRootsRouteV1 {
		f.rootCalls.Add(1)
		if r.URL.Query().Get("operation") != threadRootsQueryV1 || variables["id"] != testThreadIssueID || variables["first"] != float64(threadRootPageSize) {
			f.t.Error("incorrect root operation or live bootstrap identity")
		}
		for _, key := range []string{"last", "before", "commentTargetId", "afterTarget", "beforeTarget", "commentsSortBy", "hasCommentTargetId", "skipComments", "jivRelayCommentsEnabled", "scrubberEnabled"} {
			if _, exists := variables[key]; !exists {
				f.t.Errorf("missing required root variable %s", key)
			}
		}
		node.ID, node.Key, node.CommentsPaginated = testThreadIssueID, "TEST-1", f.roots[after]
		if f.rootNodeID != "" {
			node.ID = f.rootNodeID
		}
	} else {
		f.replyCalls.Add(1)
		if r.URL.Query().Get("operation") != threadRepliesQueryV1 || variables["id"] != "global/r1" || variables["first"] != float64(threadReplyPageSize) {
			f.t.Error("incorrect reply operation or parent identity")
		}
		node.ID, node.RepliesPaginated = "global/r1", f.replies[after]
		if f.replyNodeID != "" {
			node.ID = f.replyNodeID
		}
	}
	f.writeJSON(w, map[string]any{"data": map[string]any{"node": node}})
}

func (f *threadFixture) writeJSON(w http.ResponseWriter, value any) {
	f.t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		f.t.Errorf("encode synthetic response: %v", err)
	}
}

func testThreadClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	return NewClient(Config{Server: server.URL, Login: "test-login", APIToken: "test-token"}, WithInsecureTLS(true))
}

func TestGetIssueCommentThreadsPagination(t *testing.T) {
	f := newThreadFixture(t)
	client := testThreadClient(t, f.serve)
	transport := &threadCountingTransport{base: client.transport}
	client.transport = transport
	comments, err := client.GetIssueCommentThreads(context.Background(), "TEST-1")
	if err != nil {
		t.Fatal(err)
	}
	parents := map[string]string{}
	for _, comment := range comments {
		parents[comment.ID] = comment.ParentID
		if _, ok := comment.Body.(*adf.ADF); !ok {
			t.Errorf("comment %s body is not normalized ADF", comment.ID)
		}
		if comment.Author.AccountID != "synthetic" || comment.Created == "" {
			t.Error("REST metadata was not preserved")
		}
	}
	want := map[string]string{"r1": "", "r2": "", "c1": "r1", "c2": "r1", "c3": "r1"}
	if !reflect.DeepEqual(parents, want) {
		t.Fatalf("parents = %v, want %v", parents, want)
	}
	if f.rootCalls.Load() != 2 || f.replyCalls.Load() != 2 || f.restCalls.Load() != 3 || transport.calls.Load() != 9 {
		t.Fatal("did not exhaust all root, reply, and REST pages using configured transport")
	}
	encoded, err := json.Marshal(comments)
	if err != nil || strings.Contains(string(encoded), "ParentID") || strings.Contains(string(encoded), "parentId") {
		t.Fatal("native parentage leaked into the flat JSON contract")
	}
}

type threadCountingTransport struct {
	base  http.RoundTripper
	calls atomic.Int32
}

func (r *threadCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls.Add(1)
	return r.base.RoundTrip(req)
}

func TestGetIssueCommentThreadsEmpty(t *testing.T) {
	f := newThreadFixture(t)
	f.roots = map[string]*threadConnection{"": testThreadConnection(false, false)}
	zero := 0
	f.rest = map[int]*threadRESTPage{0: {StartAt: &zero, Total: &zero, Comments: []IssueComment{}}}
	comments, err := testThreadClient(t, f.serve).GetIssueCommentThreads(context.Background(), "TEST-1")
	if err != nil || len(comments) != 0 || comments == nil {
		t.Fatalf("empty issue: comments=%v error=%v", comments, err)
	}
}

func TestGetIssueCommentThreadsRejectsIncompleteData(t *testing.T) {
	cases := []struct {
		name   string
		change func(*threadFixture)
		want   string
	}{
		{"duplicate root", func(f *threadFixture) { f.roots["cursor-r1"].Edges[0].Node.CommentID = "r1" }, "duplicate"},
		{"duplicate global ID", func(f *threadFixture) { f.roots["cursor-r1"].Edges[0].Node.ID = "global/r1" }, "duplicate"},
		{"conflicting parent", func(f *threadFixture) { f.roots["cursor-r1"].Edges[0].Node.CommentID = "c1" }, "conflicting parentage"},
		{"cycle", func(f *threadFixture) { f.replies["cursor-c1"].Edges[0].Node = testThreadComment("r1") }, "cycle"},
		{"missing root ID", func(f *threadFixture) { f.roots[""].Edges[0].Node.ID = "" }, "missing comment identity"},
		{"missing comment ID", func(f *threadFixture) { f.roots[""].Edges[0].Node.CommentID = "" }, "missing comment identity"},
		{"missing node", func(f *threadFixture) { f.roots[""].Edges[0].Node = nil }, "missing comment identity"},
		{"wrong issue node", func(f *threadFixture) { f.rootNodeID = "other-issue" }, "node identity"},
		{"missing roots", func(f *threadFixture) { f.roots[""] = nil }, "missing connection"},
		{"missing edges", func(f *threadFixture) { f.roots[""].Edges = nil }, "missing connection"},
		{"missing page info", func(f *threadFixture) { f.roots[""].PageInfo = nil }, "pagination metadata"},
		{"missing forward flag", func(f *threadFixture) { f.roots[""].PageInfo.HasNextPage = nil }, "pagination metadata"},
		{"missing backward flag", func(f *threadFixture) { f.roots[""].PageInfo.HasPreviousPage = nil }, "pagination metadata"},
		{"initial previous page", func(f *threadFixture) { *f.roots[""].PageInfo.HasPreviousPage = true }, "direction coverage"},
		{"later no previous page", func(f *threadFixture) { *f.roots["cursor-r1"].PageInfo.HasPreviousPage = false }, "direction coverage"},
		{"missing root cursor", func(f *threadFixture) { f.roots[""].Edges[0].Cursor = "" }, "page cursors"},
		{"wrong start cursor", func(f *threadFixture) { f.roots[""].PageInfo.StartCursor = "wrong" }, "page cursors"},
		{"root cursor cycle", func(f *threadFixture) {
			c := f.roots["cursor-r1"]
			c.Edges[0].Cursor, c.PageInfo.StartCursor, c.PageInfo.EndCursor = "cursor-r1", "cursor-r1", "cursor-r1"
		}, "cyclic"},
		{"empty progressive roots", func(f *threadFixture) { f.roots["cursor-r1"] = testThreadConnection(true, false) }, "nonprogressive"},
		{"missing replies", func(f *threadFixture) { f.roots[""].Edges[0].Node.RepliesPaginated = nil }, "missing connection"},
		{"missing reply ID", func(f *threadFixture) { f.replies["cursor-c1"].Edges[0].Node.CommentID = "" }, "missing comment identity"},
		{"wrong reply parent", func(f *threadFixture) { f.replyNodeID = "unrelated-parent" }, "node identity"},
		{"uncovered reply prefix", func(f *threadFixture) { *f.roots[""].Edges[0].Node.RepliesPaginated.PageInfo.HasPreviousPage = true }, "direction coverage"},
		{"reply cursor cycle", func(f *threadFixture) {
			c := f.replies["cursor-c1"]
			c.Edges[0].Cursor, c.PageInfo.StartCursor, c.PageInfo.EndCursor = "cursor-c1", "cursor-c1", "cursor-c1"
		}, "cyclic"},
		{"alias mismatch", func(f *threadFixture) { f.roots[""].Edges[0].Node.ChildComments.Edges[0].Node.CommentID = "unknown" }, "child alias"},
		{"missing REST ID", func(f *threadFixture) { f.rest[0].Comments[0].ID = "" }, "REST comment identity"},
		{"duplicate REST ID", func(f *threadFixture) { f.rest[2].Comments[0].ID = "r2" }, "duplicate REST"},
		{"missing REST body", func(f *threadFixture) { f.rest[0].Comments[0].Body = nil }, "comment body"},
		{"flat REST body", func(f *threadFixture) { f.rest[0].Comments[0].Body = "" }, "comment body"},
		{"malformed ADF", func(f *threadFixture) { f.rest[0].Comments[0].Body = map[string]any{"type": "doc", "version": 1} }, "comment body"},
		{"missing created", func(f *threadFixture) { f.rest[0].Comments[0].Created = "" }, "creation metadata"},
		{"unknown REST comment", func(f *threadFixture) { f.rest[0].Comments[0].ID = "unknown" }, "no verified native relationship"},
		{"coverage mismatch", func(f *threadFixture) {
			f.rest[4].Comments = append(f.rest[4].Comments, testThreadBody("extra"))
			*f.rest[0].Total = 6
			*f.rest[2].Total = 6
			*f.rest[4].Total = 6
		}, "coverage differs"},
		{"REST total grows", func(f *threadFixture) { *f.rest[2].Total = 6 }, "total changed"},
		{"REST total shrinks", func(f *threadFixture) { *f.rest[2].Total = 4 }, "total changed"},
		{"REST wrong offset", func(f *threadFixture) { *f.rest[2].StartAt = 0 }, "REST pagination metadata"},
		{"REST missing total", func(f *threadFixture) { f.rest[0].Total = nil }, "REST pagination metadata"},
		{"REST missing page", func(f *threadFixture) { f.rest[0].Comments = nil }, "REST pagination metadata"},
		{"REST empty page", func(f *threadFixture) { f.rest[2].Comments = []IssueComment{} }, "nonprogressive REST"},
		{"REST overflow total", func(f *threadFixture) { *f.rest[0].Total = 1 }, "incomplete or nonprogressive"},
		{"REST record cap", func(f *threadFixture) { *f.rest[0].Total = threadRecordLimit + 1 }, "record limit"},
		{"missing cloud ID", func(f *threadFixture) { f.page = "<html></html>" }, "lacks Cloud identity"},
		{"conflicting cloud ID", func(f *threadFixture) { f.page += `<meta name="ajs-cloud-id" content="another-cloud">` }, "conflicting Cloud identity"},
		{"unsafe cloud ID", func(f *threadFixture) { f.page = `<meta name="ajs-cloud-id" content="../other">` }, "invalid or conflicting"},
		{"missing issue ID", func(f *threadFixture) { delete(f.issue, "id") }, "issue identity"},
		{"wrong issue key", func(f *threadFixture) { f.issue["key"] = "TEST-2" }, "issue identity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newThreadFixture(t)
			tc.change(f)
			comments, err := testThreadClient(t, f.serve).GetIssueCommentThreads(context.Background(), "TEST-1")
			if err == nil || !strings.Contains(err.Error(), tc.want) || comments != nil {
				t.Fatalf("comments must fail closed: error=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestGetIssueCommentThreadsResponseErrors(t *testing.T) {
	for _, path := range []string{"/browse/TEST-1", "/rest/api/3/issue/TEST-1", threadRootsRouteV1, threadRepliesRouteV1, "/rest/api/3/issue/TEST-1/comment"} {
		t.Run(path, func(t *testing.T) {
			for _, kind := range []string{"HTTP", "JSON", "GraphQL"} {
				if kind == "JSON" && path == "/browse/TEST-1" || kind == "GraphQL" && path != threadRootsRouteV1 && path != threadRepliesRouteV1 {
					continue
				}
				t.Run(kind, func(t *testing.T) {
					f := newThreadFixture(t)
					f.intercept = func(w http.ResponseWriter, r *http.Request) bool {
						if r.URL.Path != path {
							return false
						}
						switch kind {
						case "HTTP":
							w.WriteHeader(http.StatusForbidden)
							_, _ = io.WriteString(w, "PRIVATE_SENTINEL")
						case "JSON":
							_, _ = io.WriteString(w, `{"PRIVATE_SENTINEL":`)
						case "GraphQL":
							f.writeJSON(w, map[string]any{"data": map[string]any{"node": map[string]any{"id": testThreadIssueID}}, "errors": []any{map[string]any{"message": "PRIVATE_SENTINEL"}}})
						}
						return true
					}
					_, err := testThreadClient(t, f.serve).GetIssueCommentThreads(context.Background(), "TEST-1")
					if err == nil || strings.Contains(err.Error(), "PRIVATE_SENTINEL") {
						t.Fatalf("missing or unsanitized error: %v", err)
					}
				})
			}
		})
	}
}

func TestGetIssueCommentThreadsAuthenticationAndOrigin(t *testing.T) {
	for _, auth := range []AuthType{AuthTypeBearer, AuthTypeMTLS, AuthTypeCFAccess, "unknown"} {
		t.Run(string(auth), func(t *testing.T) {
			client := &Client{authType: &auth, server: "unused", login: "test-login", token: "test-token"}
			if _, err := client.GetIssueCommentThreads(context.Background(), "TEST-1"); err == nil || !strings.Contains(err.Error(), "only Basic") {
				t.Fatalf("unsupported auth: %v", err)
			}
		})
	}
	for _, auth := range []*AuthType{nil, new(AuthType), new(AuthTypeBasic)} {
		f := newThreadFixture(t)
		client := testThreadClient(t, f.serve)
		client.authType = auth
		if _, err := client.GetIssueCommentThreads(context.Background(), "TEST-1"); err != nil {
			t.Fatal(err)
		}
	}
	for _, server := range []string{"", ":bad", "file:///local", "http://test.invalid", "https://test.invalid/context", "https://test.invalid/?query", "https://test.invalid/?", "https://test.invalid/%2f", "https://user@test.invalid", "https://test.invalid/#fragment"} {
		client := &Client{server: server, login: "test-login", token: "test-token"}
		if _, err := client.GetIssueCommentThreads(context.Background(), "TEST-1"); err == nil {
			t.Fatalf("accepted invalid origin %q", server)
		}
	}
	client := &Client{}
	if _, err := client.GetIssueCommentThreads(context.Background(), "TEST-1"); err == nil {
		t.Fatal("accepted missing credentials")
	}
	if _, err := client.GetIssueCommentThreads(context.Background(), "../TEST-1"); err == nil || !strings.Contains(err.Error(), "invalid issue key") {
		t.Fatal("accepted unsafe issue key")
	}
}

func testThreadEntryPoints() map[string]func(*Client, context.Context) error {
	return map[string]func(*Client, context.Context) error{
		"comments": func(c *Client, ctx context.Context) error {
			_, err := c.GetIssueCommentThreads(ctx, "TEST-1")
			return err
		},
		"issue with comments": func(c *Client, ctx context.Context) error {
			_, err := c.GetIssueWithCommentThreads(ctx, "TEST-1", true)
			return err
		},
		"issue without comments": func(c *Client, ctx context.Context) error {
			_, err := c.GetIssueWithCommentThreads(ctx, "TEST-1", false)
			return err
		},
	}
}

func TestGetIssueCommentThreadsNoRedirects(t *testing.T) {
	var forwarded, authForwarded atomic.Int32
	targetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded.Add(1)
		if r.Header.Get("Authorization") != "" {
			authForwarded.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	})
	plainTarget := httptest.NewServer(targetHandler)
	defer plainTarget.Close()
	tlsTarget := httptest.NewTLSServer(targetHandler)
	defer tlsTarget.Close()
	for name, read := range testThreadEntryPoints() {
		t.Run(name, func(t *testing.T) {
			for _, location := range []string{plainTarget.URL + "/redirected", strings.Replace(tlsTarget.URL, "127.0.0.1", "localhost", 1) + "/redirected", "/redirected"} {
				t.Run(location, func(t *testing.T) {
					client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path == "/redirected" {
							targetHandler.ServeHTTP(w, r)
							return
						}
						if r.URL.Path != "/rest/api/3/issue/TEST-1" {
							t.Error("first request was not the hardened base issue fetch")
						}
						http.Redirect(w, r, location, http.StatusFound)
					})
					if err := read(client, context.Background()); err == nil || !strings.Contains(err.Error(), "302") {
						t.Fatalf("redirect was not rejected: %v", err)
					}
				})
			}
		})
	}
	if forwarded.Load() != 0 || authForwarded.Load() != 0 {
		t.Fatal("a redirect target was called or received credentials")
	}
}

func TestGetIssueCommentThreadsContext(t *testing.T) {
	for name, read := range testThreadEntryPoints() {
		t.Run(name, func(t *testing.T) {
			for _, mode := range []string{"cancel", "deadline", "configured timeout"} {
				t.Run(mode, func(t *testing.T) {
					client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
					ctx := context.Background()
					var cancel context.CancelFunc
					want := error(context.DeadlineExceeded)
					switch mode {
					case "cancel":
						ctx, cancel = context.WithCancel(ctx)
						cancel()
						want = context.Canceled
					case "deadline":
						ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
						defer cancel()
					case "configured timeout":
						client.timeout = time.Millisecond
					}
					if err := read(client, ctx); !errors.Is(err, want) {
						t.Fatalf("context cause lost: %v", err)
					}
				})
			}
		})
	}
}

func TestGetIssueCommentThreadsCaps(t *testing.T) {
	t.Run("response", func(t *testing.T) {
		client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat(" ", threadResponseLimit+1))
		})
		if _, err := client.GetIssueCommentThreads(context.Background(), "TEST-1"); err == nil || !strings.Contains(err.Error(), "response size limit") {
			t.Fatalf("response cap: %v", err)
		}
	})
	t.Run("requests", func(t *testing.T) {
		f := newThreadFixture(t)
		client := testThreadClient(t, f.serve)
		r, err := client.newThreadReader()
		if err != nil {
			t.Fatal(err)
		}
		r.requests = threadRequestLimit
		if _, err := r.get(context.Background(), "/browse/TEST-1", true); err == nil || !strings.Contains(err.Error(), "request limit") {
			t.Fatalf("request cap: %v", err)
		}
	})
	t.Run("native records", func(t *testing.T) {
		f := newThreadFixture(t)
		conn := testThreadConnection(false, false)
		for i := 0; i <= threadRecordLimit; i++ {
			node := testThreadComment(strconv.Itoa(i))
			node.RepliesPaginated = testThreadConnection(false, false)
			conn.Edges = append(conn.Edges, threadEdge{Cursor: "cursor-" + node.CommentID, Node: node})
		}
		conn.PageInfo.StartCursor = conn.Edges[0].Cursor
		conn.PageInfo.EndCursor = conn.Edges[len(conn.Edges)-1].Cursor
		f.roots[""] = conn
		if _, err := testThreadClient(t, f.serve).GetIssueCommentThreads(context.Background(), "TEST-1"); err == nil || !strings.Contains(err.Error(), "record limit") {
			t.Fatalf("native record cap: %v", err)
		}
	})
}

func TestGetIssueCommentThreadsNoDebugDump(t *testing.T) {
	f := newThreadFixture(t)
	client := testThreadClient(t, f.serve)
	client.debug = true
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = read.Close() }()
	old := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = old }()
	_, callErr := client.GetIssueCommentThreads(context.Background(), "TEST-1")
	_ = write.Close()
	output, err := io.ReadAll(read)
	if err != nil || callErr != nil || len(output) != 0 {
		t.Fatal("adapter failed or emitted a debug request dump")
	}
}

func TestGetIssuePartialCommentPage(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				comments := make([]IssueComment, count)
				for i := range comments {
					comments[i] = testThreadBody(strconv.Itoa(i))
				}
				response := map[string]any{"key": "TEST-1", "fields": map[string]any{"comment": map[string]any{"total": 100, "comments": comments}}}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client := NewClient(Config{Server: server.URL})
			got, err := client.GetIssue("TEST-1", issuefilter.NewNumCommentsFilter(100))
			if err != nil || got.Fields.Comment.Total != 100 || len(got.Fields.Comment.Comments) != count {
				t.Fatalf("partial flat page: %v", err)
			}
			for _, comment := range got.Fields.Comment.Comments {
				if _, ok := comment.Body.(*adf.ADF); !ok || comment.ParentID != "" {
					t.Fatal("flat comment normalization or parent behavior changed")
				}
			}
		})
	}
}

func TestGetIssueWithCommentThreads(t *testing.T) {
	for _, includeComments := range []bool{false, true} {
		t.Run(strconv.FormatBool(includeComments), func(t *testing.T) {
			f := newThreadFixture(t)
			client := testThreadClient(t, f.serve)
			transport := &threadCountingTransport{base: client.transport}
			client.transport = transport
			issue, err := client.GetIssueWithCommentThreads(context.Background(), "TEST-1", includeComments)
			if err != nil {
				t.Fatal(err)
			}
			if issue.Key != "TEST-1" {
				t.Fatal("base issue identity was not preserved")
			}
			if _, ok := issue.Fields.Description.(*adf.ADF); !ok {
				t.Fatal("description was not normalized to ADF")
			}
			if !includeComments {
				if transport.calls.Load() != 1 || f.rootCalls.Load() != 0 || f.replyCalls.Load() != 0 || f.restCalls.Load() != 0 {
					t.Fatal("disabled comments fetched more than the base issue")
				}
				if issue.Fields.Comment.Total != 17 || len(issue.Fields.Comment.Comments) != 1 || issue.Fields.Comment.Comments[0].ID != "embedded" {
					t.Fatal("disabled comments changed the base issue's partial page or total")
				}
				return
			}
			parents := make(map[string]string)
			for _, comment := range issue.Fields.Comment.Comments {
				parents[comment.ID] = comment.ParentID
				if _, ok := comment.Body.(*adf.ADF); !ok {
					t.Fatal("comment body was not normalized")
				}
			}
			want := map[string]string{"r1": "", "r2": "", "c1": "r1", "c2": "r1", "c3": "r1"}
			if !reflect.DeepEqual(parents, want) || issue.Fields.Comment.Total != len(want) || transport.calls.Load() != 9 {
				t.Fatal("aggregate issue did not attach complete threads or redundantly fetched bootstrap identity")
			}
		})
	}
}

func TestThreadReadRejectsHTTPBeforeRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	for name, read := range testThreadEntryPoints() {
		t.Run(name, func(t *testing.T) {
			client := NewClient(Config{Server: server.URL, Login: "test-login", APIToken: "test-token"})
			if err := read(client, context.Background()); err == nil || !strings.Contains(err.Error(), "HTTPS") {
				t.Fatalf("HTTP origin was not rejected: %v", err)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatal("HTTP origin received a request")
	}
}

func TestGetIssueWithCommentThreadsValidatesBeforeRequests(t *testing.T) {
	for _, includeComments := range []bool{false, true} {
		for _, invalid := range []string{"auth", "credentials", "key"} {
			t.Run(fmt.Sprintf("%t/%s", includeComments, invalid), func(t *testing.T) {
				var requests atomic.Int32
				client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
				key := "TEST-1"
				switch invalid {
				case "auth":
					client.authType = new(AuthTypeBearer)
				case "credentials":
					client.token = ""
				case "key":
					key = "../TEST-1"
				}
				if _, err := client.GetIssueWithCommentThreads(context.Background(), key, includeComments); err == nil || requests.Load() != 0 {
					t.Fatal("invalid thread-mode configuration reached the network")
				}
			})
		}
	}
}

func TestThreadReadBaseIssueErrors(t *testing.T) {
	cases := []struct {
		name, body, want string
		status           int
	}{
		{"HTTP", "PRIVATE_SENTINEL", "403", http.StatusForbidden},
		{"malformed JSON", `{"PRIVATE_SENTINEL":`, "invalid JSON", http.StatusOK},
		{"null issue", `null`, "issue identity", http.StatusOK},
		{"missing fields", `{"id":"7","key":"TEST-1"}`, "issue identity or fields", http.StatusOK},
		{"bad description", `{"id":"7","key":"TEST-1","fields":{"description":"PRIVATE_SENTINEL"}}`, "description ADF", http.StatusOK},
		{"negative count", `{"id":"7","key":"TEST-1","fields":{"comment":{"total":-1}}}`, "comment count", http.StatusOK},
		{"response cap", strings.Repeat(" ", threadResponseLimit+1), "response size limit", http.StatusOK},
	}
	for name, read := range testThreadEntryPoints() {
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					var requests atomic.Int32
					client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						if r.URL.Path != "/rest/api/3/issue/TEST-1" {
							t.Error("invalid base issue allowed a later request")
						}
						w.WriteHeader(tc.status)
						_, _ = io.WriteString(w, tc.body)
					})
					err := read(client, context.Background())
					if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "PRIVATE_SENTINEL") || requests.Load() != 1 {
						t.Fatalf("base issue error was missing, unsanitized, or retried: %v", err)
					}
				})
			}
		})
	}
}

func TestThreadReadBaseIssueCancellation(t *testing.T) {
	for name, read := range testThreadEntryPoints() {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var started atomic.Bool
			client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/rest/api/3/issue/TEST-1" {
					t.Error("first fetch was not the base issue")
				}
				started.Store(true)
				cancel()
				<-r.Context().Done()
			})
			client.timeout = time.Second
			if err := read(client, ctx); !started.Load() || !errors.Is(err, context.Canceled) {
				t.Fatalf("base issue fetch did not retain request cancellation: %v", err)
			}
		})
	}
}

func TestGetIssueWithCommentThreadsWholeOperationTimeout(t *testing.T) {
	const timeout = 100 * time.Millisecond
	const delay = 60 * time.Millisecond
	f := newThreadFixture(t)
	client := testThreadClient(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(delay):
			f.serve(w, r)
		case <-r.Context().Done():
		}
	})
	client.timeout = timeout
	_, err := client.GetIssueWithCommentThreads(context.Background(), "TEST-1", true)
	if !errors.Is(err, context.DeadlineExceeded) || f.rootCalls.Load() != 0 {
		t.Fatalf("initial fetch and bootstrap did not share one deadline: %v", err)
	}
}

func TestThreadRequestErrorRedactsCause(t *testing.T) {
	cause := fmt.Errorf("PRIVATE_SENTINEL: %w", context.Canceled)
	err := fmt.Errorf("request: %w", threadRequestError{cause: cause})
	if strings.Contains(err.Error(), "PRIVATE_SENTINEL") || !errors.Is(err, context.Canceled) {
		t.Fatal("transport errors must hide details and retain errors.Is")
	}
}
