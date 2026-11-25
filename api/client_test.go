package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

// setInstallation restores global installation state so proxy cases cannot leak configuration.
func setInstallation(t *testing.T, installation string) {
	t.Helper()
	previous := viper.Get("installation")
	viper.Set("installation", installation)
	t.Cleanup(func() { viper.Set("installation", previous) })
}

// TestProxyGetIssueWithCommentThreadsRejectsLocal ensures Cloud-only mode fails before using a client, even with comments hidden.
func TestProxyGetIssueWithCommentThreadsRejectsLocal(t *testing.T) {
	setInstallation(t, jira.InstallationTypeLocal)
	iss, err := ProxyGetIssueWithCommentThreads(nil, "ISSUE-1", issue.NewNumCommentsFilter(0))
	if iss != nil || err == nil || !strings.Contains(err.Error(), "Jira Cloud") || !strings.Contains(err.Error(), "Server/local") {
		t.Fatalf("expected clear Cloud-only rejection without a client call, got issue %v, error %v", iss, err)
	}
}

// TestProxyGetIssueWithCommentThreadsZeroComments protects the base-only fetch without normalizing or replacing embedded comments.
func TestProxyGetIssueWithCommentThreadsZeroComments(t *testing.T) {
	setInstallation(t, jira.InstallationTypeCloud)
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/rest/api/3/issue/ISSUE-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"101","key":"ISSUE-1","fields":{"description":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Synthetic description"}]}]},"comment":{"total":10,"comments":[{"id":"embedded","body":{"type":"doc","version":1,"content":[]}}]}}}`)
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{
		Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-token",
	}, jira.WithTimeout(clientTimeout), jira.WithInsecureTLS(true))
	iss, err := ProxyGetIssueWithCommentThreads(client, "ISSUE-1", issue.NewNumCommentsFilter(0))
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected only the base issue request, got %d requests", requests.Load())
	}
	if iss.Key != "ISSUE-1" || iss.Fields.Comment.Total != 10 || len(iss.Fields.Comment.Comments) != 1 {
		t.Fatalf("base issue was changed: %+v", iss)
	}
	if _, ok := iss.Fields.Description.(*adf.ADF); !ok {
		t.Fatalf("description was not normalized: %T", iss.Fields.Description)
	}
	if _, ok := iss.Fields.Comment.Comments[0].Body.(map[string]any); !ok {
		t.Fatalf("embedded comment was unexpectedly converted: %T", iss.Fields.Comment.Comments[0].Body)
	}
}

// TestProxyGetIssueWithCommentThreadsErrors requires contextual wrapped failures instead of a misleading flat-comment fallback.
func TestProxyGetIssueWithCommentThreadsErrors(t *testing.T) {
	setInstallation(t, jira.InstallationTypeCloud)
	for _, failure := range []string{"base issue", "comment threads"} {
		t.Run(failure, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if failure == "comment threads" && r.URL.Path == "/rest/api/3/issue/ISSUE-1" {
					_, _ = io.WriteString(w, `{"id":"101","key":"ISSUE-1","fields":{"comment":{"total":10,"comments":[]}}}`)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, `{"errorMessages":["synthetic access denied"]}`)
			}))
			defer server.Close()

			client := jira.NewClient(jira.Config{
				Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-token",
			}, jira.WithTimeout(clientTimeout), jira.WithInsecureTLS(true))
			iss, err := ProxyGetIssueWithCommentThreads(client, "ISSUE-1", issue.NewNumCommentsFilter(5))
			if iss != nil || err == nil || !strings.Contains(err.Error(), "ISSUE-1") || !strings.Contains(err.Error(), "comment threads") {
				t.Fatalf("expected contextual error without a flat fallback, got issue %v, error %v", iss, err)
			}
			if errors.Unwrap(err) == nil {
				t.Fatalf("underlying error was not preserved: %v", err)
			}
			if !strings.Contains(err.Error(), "HTTP status 403") {
				t.Fatalf("HTTP error was not preserved: %v", err)
			}
			if failure == "base issue" && requests.Load() != 1 {
				t.Errorf("expected failure before thread requests, got %d requests", requests.Load())
			}
			if failure == "comment threads" && requests.Load() < 2 {
				t.Errorf("thread request was not attempted: %d requests", requests.Load())
			}
		})
	}
}

// TestProxyGetIssueWithCommentThreadsRejectsHTTPBeforeFetch prevents any insecure request, regardless of the comment limit.
func TestProxyGetIssueWithCommentThreadsRejectsHTTPBeforeFetch(t *testing.T) {
	setInstallation(t, jira.InstallationTypeCloud)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := jira.NewClient(jira.Config{
		Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-token",
	}, jira.WithTimeout(clientTimeout))

	for _, limit := range []uint{0, 1} {
		iss, err := ProxyGetIssueWithCommentThreads(client, "ISSUE-1", issue.NewNumCommentsFilter(limit))
		if iss != nil || err == nil || !strings.Contains(err.Error(), "HTTPS") {
			t.Errorf("limit %d: expected HTTPS rejection, got issue %v, error %v", limit, iss, err)
		}
	}
	if requests.Load() != 0 {
		t.Errorf("HTTP origin received %d requests before validation", requests.Load())
	}
}

// TestProxyGetIssueWithCommentThreadsDoesNotFollowInitialRedirect keeps even the base issue fetch from forwarding thread-mode requests.
func TestProxyGetIssueWithCommentThreadsDoesNotFollowInitialRedirect(t *testing.T) {
	setInstallation(t, jira.InstallationTypeCloud)
	var destinationRequests atomic.Int32
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		destinationRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"101","key":"ISSUE-1","fields":{}}`)
	}))
	defer destination.Close()
	var sourceRequests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceRequests.Add(1)
		if r.URL.Path != "/rest/api/3/issue/ISSUE-1" {
			t.Errorf("unexpected initial request: %s", r.URL.Path)
		}
		http.Redirect(w, r, destination.URL+"/redirected", http.StatusFound)
	}))
	defer server.Close()
	client := jira.NewClient(jira.Config{
		Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-token",
	}, jira.WithTimeout(clientTimeout), jira.WithInsecureTLS(true))

	for _, limit := range []uint{0, 1} {
		iss, err := ProxyGetIssueWithCommentThreads(client, "ISSUE-1", issue.NewNumCommentsFilter(limit))
		if iss != nil || err == nil || !strings.Contains(err.Error(), "302") {
			t.Errorf("limit %d: expected initial redirect rejection, got issue %v, error %v", limit, iss, err)
		}
	}
	if sourceRequests.Load() != 2 || destinationRequests.Load() != 0 {
		t.Errorf("initial redirect was followed: source requests %d, destination requests %d", sourceRequests.Load(), destinationRequests.Load())
	}
}

// TestProxyGetIssueWithCommentThreadsCompleteCollection requires reconciled parentage and normalized bodies without truncating fetched threads.
func TestProxyGetIssueWithCommentThreadsCompleteCollection(t *testing.T) {
	setInstallation(t, jira.InstallationTypeCloud)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("thread loading must be read-only: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		login, token, ok := r.BasicAuth()
		if !ok || login != "synthetic@example.invalid" || token != "synthetic-token" {
			t.Error("thread loading did not reuse configured Basic auth")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/browse/ISSUE-1":
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<meta name="ajs-cloud-id" content="synthetic-cloud">`)
		case r.URL.Path == "/rest/api/3/issue/ISSUE-1":
			_, _ = io.WriteString(w, `{"id":"101","key":"ISSUE-1","fields":{"description":{"type":"doc","version":1,"content":[]},"comment":{"total":99,"comments":[]}}}`)
		case strings.HasPrefix(r.URL.Path, "/gateway/api/graphql/pq/") && r.URL.Query().Get("operation") == "CommentsViewRelayAdapter_RefetchQuery":
			_, _ = io.WriteString(w, `{"data":{"node":{"id":"ari:cloud:jira:synthetic-cloud:issue/101","key":"ISSUE-1","commentsPaginated":{"edges":[
				{"cursor":"root-a","node":{"id":"native-a","commentId":"root-a","repliesPaginated":{"edges":[{"cursor":"reply-a","node":{"id":"native-reply","commentId":"reply-a"}}],"pageInfo":{"hasNextPage":false,"hasPreviousPage":false,"startCursor":"reply-a","endCursor":"reply-a"}}}},
				{"cursor":"root-b","node":{"id":"native-b","commentId":"root-b","repliesPaginated":{"edges":[],"pageInfo":{"hasNextPage":false,"hasPreviousPage":false,"startCursor":"","endCursor":""}}}}
			],"pageInfo":{"hasNextPage":false,"hasPreviousPage":false,"startCursor":"root-a","endCursor":"root-b"}}}}}`)
		case r.URL.Path == "/rest/api/3/issue/ISSUE-1/comment":
			_, _ = io.WriteString(w, `{"startAt":0,"total":3,"comments":[
				{"id":"root-a","body":{"type":"doc","version":1,"content":[]},"created":"2026-01-01T01:00:00.000+0000"},
				{"id":"reply-a","body":{"type":"doc","version":1,"content":[]},"created":"2026-01-02T01:00:00.000+0000"},
				{"id":"root-b","body":{"type":"doc","version":1,"content":[]},"created":"2026-01-03T01:00:00.000+0000"}
			]}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := jira.NewClient(jira.Config{
		Server: server.URL, Login: "synthetic@example.invalid", APIToken: "synthetic-token",
	}, jira.WithTimeout(clientTimeout), jira.WithInsecureTLS(true))

	for _, tt := range []struct {
		name string
		opts []filter.Filter
	}{
		{name: "no filter"},
		{name: "one whole thread", opts: []filter.Filter{issue.NewNumCommentsFilter(1)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			iss, err := ProxyGetIssueWithCommentThreads(client, "ISSUE-1", tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if iss.Fields.Comment.Total != 3 || len(iss.Fields.Comment.Comments) != 3 {
				t.Fatalf("proxy must replace incomplete embedded comments without truncating threads: %+v", iss.Fields.Comment)
			}
			if _, ok := iss.Fields.Description.(*adf.ADF); !ok {
				t.Fatalf("base description was not normalized: %T", iss.Fields.Description)
			}
			for idx, parent := range []string{"", "root-a", ""} {
				comment := iss.Fields.Comment.Comments[idx]
				if comment.ParentID != parent {
					t.Errorf("comment %s parent = %q, want %q", comment.ID, comment.ParentID, parent)
				}
				if _, ok := comment.Body.(*adf.ADF); !ok {
					t.Errorf("comment %s body was not normalized: %T", comment.ID, comment.Body)
				}
			}
		})
	}
}

// TestProxyGetIssueFlatUnchanged protects single-request flat loading and installation-specific body normalization from thread-mode changes.
func TestProxyGetIssueFlatUnchanged(t *testing.T) {
	for _, installation := range []string{jira.InstallationTypeCloud, jira.InstallationTypeLocal} {
		t.Run(installation, func(t *testing.T) {
			setInstallation(t, installation)
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				path := "/rest/api/3/issue/ISSUE-1"
				if installation == jira.InstallationTypeLocal {
					path = "/rest/api/2/issue/ISSUE-1"
				}
				if r.Method != http.MethodGet || r.URL.Path != path {
					t.Errorf("unexpected flat request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				if installation == jira.InstallationTypeLocal {
					_, _ = io.WriteString(w, `{"key":"ISSUE-1","fields":{"comment":{"total":1,"comments":[{"id":"flat","body":"Synthetic flat comment"}]}}}`)
					return
				}
				_, _ = io.WriteString(w, `{"key":"ISSUE-1","fields":{"comment":{"total":1,"comments":[{"id":"flat","body":{"type":"doc","version":1,"content":[]}}]}}}`)
			}))
			defer server.Close()

			client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(clientTimeout))
			iss, err := ProxyGetIssue(client, "ISSUE-1", issue.NewNumCommentsFilter(1))
			if err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 1 || iss.Fields.Comment.Total != 1 || len(iss.Fields.Comment.Comments) != 1 {
				t.Fatalf("flat behavior changed: requests %d, issue %+v", requests.Load(), iss)
			}
			body := iss.Fields.Comment.Comments[0].Body
			if installation == jira.InstallationTypeLocal {
				if body != "Synthetic flat comment" {
					t.Errorf("local flat body changed: %v", body)
				}
			} else if _, ok := body.(*adf.ADF); !ok {
				t.Errorf("Cloud flat body was not normalized: %T", body)
			}
		})
	}
}
