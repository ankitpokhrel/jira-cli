package view

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestNormalizeNavigationKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, current, want string
	}{
		{name: "same project", input: "A-2", current: "A-1", want: "A-2"},
		{name: "cross project", input: "B-2", current: "A-1", want: "B-2"},
		{name: "case and trim", input: "  b0_test-2  ", current: "A-1", want: "B0_TEST-2"},
		{name: "outer unicode whitespace", input: "\u2003b-2\u2003", current: "A-1", want: "B-2"},
		{name: "number", input: " 3 ", current: "b_2-1", want: "B_2-3"},
		{name: "full key with invalid current", input: "B-2", current: "invalid", want: "B-2"},
		{name: "uint64 number", input: "18446744073709551615", current: "A-1", want: "A-18446744073709551615"},
		{name: "uint64 full key", input: "A-18446744073709551615", want: "A-18446744073709551615"},
		{name: "maximum key bytes", input: strings.Repeat("A", navigationInputLimit-2) + "-1", want: strings.Repeat("A", navigationInputLimit-2) + "-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeNavigationKey(tc.input, tc.current)
			if err != nil || got != tc.want {
				t.Errorf("normalized key = %q, error %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestNormalizeNavigationRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"", " ", "0", "-1", "+1", "01", "000", "A-0", "A-01", "A--1", "A-+1",
		"A", "A-", "A-B-1", "1A-1", "_A-1", "A.B-1", "A/1", "../A-1", "A-1/../../B-2",
		"https://jira.invalid/browse/A-1", "//jira.invalid/A-1", "A-1?query=x", "A-1#fragment",
		"A -1", "A- 1", "A-1 B-2", "A-1\t", "\nA-1", "A-1\r\nInjected: value", "A-1\x00", "A-1\x7f",
		"A-1\u0085", "A-1\x1b[31m", "A\u00a0-1", "ſ-1", "界-1", "A-１", "A-1\xff",
		"18446744073709551616", "A-18446744073709551616", strings.Repeat("A", navigationInputLimit+1),
		" " + strings.Repeat("A", navigationInputLimit-2) + "-1",
	}
	for idx, input := range inputs {
		got, err := normalizeNavigationKey(input, "A-1")
		if got != "" || !errors.Is(err, errNavigationInput) {
			t.Errorf("invalid case %d returned key %q, error %v", idx, got, err)
		}
	}
}

func TestNavigationNumberNeedsValidCurrentProject(t *testing.T) {
	t.Parallel()
	for _, current := range []string{"", "invalid", "A-0", "A--1", "1A-1", "A-18446744073709551616", "A-1\n"} {
		key, err := normalizeNavigationKey("2", current)
		if key != "" || !errors.Is(err, errNavigationProject) {
			t.Errorf("invalid current key allowed numeric navigation: key %q, error %v", key, err)
		}
	}
	current := strings.Repeat("A", navigationInputLimit-2) + "-1"
	if key, err := normalizeNavigationKey("10", current); key != "" || !errors.Is(err, errNavigationInput) {
		t.Errorf("inferred key exceeded byte limit: key %q, error %v", key, err)
	}
}

func TestIssueNavigationResolvesWithoutIO(t *testing.T) {
	t.Parallel()
	newRenderer := newMDRendererFactory()
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	list := IssueList{Project: "CONFIG", Data: getIssues(), Display: DisplayFormat{Comments: 1, CommentThreads: true}}
	original := list.detailView(navigationTestClient(server.URL), "A-1", newRenderer)
	crossProject := openNavigation(t, original, " b-2 ")
	number := openNavigation(t, crossProject, "3")
	if original.Title != "A-1" || crossProject.Title != "B-2" || number.Title != "B-3" {
		t.Fatal("navigation mutated an existing descriptor or used the configured project")
	}
	for _, input := range []string{"", "0", "../B-2", "https://jira.invalid/A-1", "B-2\nInjected"} {
		next, err := original.OpenTicket(input)
		if err == nil || !reflect.DeepEqual(next, tui.DetailView{}) {
			t.Errorf("invalid navigation constructed a descriptor: %+v, error %v", next, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("constructing or rejecting navigation performed HTTP requests")
	}
	if list.Project != "CONFIG" || !reflect.DeepEqual(list.Data, getIssues()) {
		t.Fatal("navigation mutated the issue list")
	}
}

type navigationRequest struct {
	method, path, body string
}

func TestIssueNavigationRoutesLoadAndComments(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	newRenderer := newMDRendererFactory()
	setNavigationCloud(t)
	requests := make(chan navigationRequest, 8)
	server := httptest.NewTLSServer(navigationFlatHandler(t, requests))
	defer server.Close()
	list := IssueList{Server: server.URL, Project: "CONFIG", Display: DisplayFormat{Comments: 1}}
	original := list.detailView(navigationTestClient(server.URL), "A-1", newRenderer)
	next := openNavigation(t, original, "new-2")
	assertNoNavigationRequests(t, requests)
	content, err := next.Load()
	if err != nil || !strings.Contains(content.Text, "NEW-2") || !strings.Contains(content.Text, server.URL+"/browse/NEW-2") {
		t.Fatalf("load navigated ticket: content %+v, error %v", content, err)
	}
	takeNavigationRequest(t, requests, navigationRequest{method: http.MethodGet, path: "/rest/api/3/issue/NEW-2"})
	if err := next.SubmitComment("New ticket comment"); err != nil {
		t.Fatal(err)
	}
	takeNavigationRequest(t, requests, navigationRequest{method: http.MethodPost, path: "/rest/api/2/issue/NEW-2/comment", body: "New ticket comment"})
	missing := openNavigation(t, next, "MISSING-9")
	assertNoNavigationRequests(t, requests)
	content, err = missing.Load()
	assertZeroDetailContent(t, content)
	var responseErr *jira.ErrUnexpectedResponse
	if !errors.As(err, &responseErr) || responseErr.StatusCode != http.StatusNotFound || !strings.Contains(err.Error(), "load issue MISSING-9") {
		t.Fatalf("navigation changed load error propagation: %v", err)
	}
	takeNavigationRequest(t, requests, navigationRequest{method: http.MethodGet, path: "/rest/api/3/issue/MISSING-9"})
	content, err = original.Load()
	if err != nil || original.Title != "A-1" || !strings.Contains(content.Text, "A-1") {
		t.Fatalf("failed lookup changed original descriptor: %+v, %v", content, err)
	}
	takeNavigationRequest(t, requests, navigationRequest{method: http.MethodGet, path: "/rest/api/3/issue/A-1"})
	if err := original.SubmitComment("Original ticket comment"); err != nil {
		t.Fatal(err)
	}
	takeNavigationRequest(t, requests, navigationRequest{method: http.MethodPost, path: "/rest/api/2/issue/A-1/comment", body: "Original ticket comment"})
}

func TestIssueNavigationInheritsCommentOptions(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "notty")
	newRenderer := newMDRendererFactory()
	setNavigationCloud(t)
	for _, tc := range []struct {
		name    string
		threads bool
		limit   uint
	}{
		{name: "flat zero"},
		{name: "flat positive", limit: 1},
		{name: "native zero", threads: true},
		{name: "native positive", threads: true, limit: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int64
			var denyReply atomic.Bool
			replyRequests := make(chan navigationRequest, 2)
			reads := threadDetailHandler(t, new(atomic.Bool), new(atomic.Int64))
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if denyReply.Load() {
					replyRequests <- navigationRequest{method: r.Method, path: r.URL.Path}
					w.WriteHeader(http.StatusForbidden)
					return
				}
				reads.ServeHTTP(w, r)
			}))
			defer server.Close()
			list := IssueList{Server: server.URL, Display: DisplayFormat{CommentThreads: tc.threads, Comments: tc.limit}}
			next := openNavigation(t, list.detailView(navigationTestClient(server.URL), "A-1", newRenderer), "TEST-2")
			if requests.Load() != 0 {
				t.Fatal("resolver fetched the next ticket")
			}
			content, err := next.Load()
			if err != nil {
				t.Fatal(err)
			}
			assertNavigationOptions(t, next, content, tc.threads, tc.limit)
			if next.SubmitReply != nil {
				denyReply.Store(true)
				missing := openNavigation(t, next, "MISSING-9")
				failedContent, loadErr := missing.Load()
				assertZeroDetailContent(t, failedContent)
				if loadErr == nil || !strings.Contains(loadErr.Error(), "load issue MISSING-9") {
					t.Fatalf("missing native ticket did not propagate its load error: %v", loadErr)
				}
				takeNavigationRequest(t, replyRequests, navigationRequest{method: http.MethodGet, path: "/rest/api/3/issue/MISSING-9"})
				err := next.SubmitReply("100", "Synthetic navigation reply")
				if err == nil || !strings.Contains(err.Error(), "reply to native thread on TEST-2") {
					t.Fatalf("reply callback did not use the next issue: %v", err)
				}
				takeNavigationRequest(t, replyRequests, navigationRequest{method: http.MethodGet, path: "/rest/api/3/issue/TEST-2"})
				assertNoNavigationRequests(t, replyRequests)
			}
		})
	}
}

func assertNavigationOptions(t *testing.T, detail tui.DetailView, content tui.DetailContent, threads bool, limit uint) {
	t.Helper()
	if got, want := detail.SubmitReply != nil, threads && limit > 0; got != want {
		t.Errorf("navigated reply eligibility = %v, want %v", got, want)
	}
	switch {
	case !threads:
		assertReplyUnavailable(t, content, "--comment-threads")
		if strings.Contains(content.Text, "Existing comment") != (limit > 0) {
			t.Error("navigation lost the flat comment limit")
		}
	case limit == 0:
		assertReplyUnavailable(t, content, "greater than 0")
		if strings.Contains(content.Text, "Native root") || strings.Contains(content.Text, "Existing comment") {
			t.Error("zero-limit navigation rendered comments")
		}
	default:
		assertThreadReplyTarget(t, detail, content)
	}
}

func navigationFlatHandler(t *testing.T, requests chan<- navigationRequest) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := navigationRequest{method: r.Method, path: r.URL.Path}
		if r.Method == http.MethodPost {
			var comment struct{ Body string }
			if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
				t.Errorf("decode local comment: %v", err)
			}
			request.body = comment.Body
			requests <- request
			w.WriteHeader(http.StatusCreated)
			return
		}
		requests <- request
		if r.URL.Path == "/rest/api/3/issue/MISSING-9" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		payload := issueDetailPayload(jira.InstallationTypeCloud, false)
		payload["key"] = strings.TrimPrefix(r.URL.Path, "/rest/api/3/issue/")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("encode local issue: %v", err)
		}
	})
}

func navigationTestClient(server string) *jira.Client {
	return jira.NewClient(jira.Config{Server: server, Login: "synthetic@example.invalid", APIToken: "synthetic-test-token"}, jira.WithInsecureTLS(true))
}

func openNavigation(t *testing.T, detail tui.DetailView, input string) tui.DetailView {
	t.Helper()
	if detail.OpenTicket == nil {
		t.Fatal("detail descriptor is missing the navigation constructor")
	}
	next, err := detail.OpenTicket(input)
	if err != nil {
		t.Fatalf("construct next ticket: %v", err)
	}
	return next
}

func takeNavigationRequest(t *testing.T, requests <-chan navigationRequest, want navigationRequest) {
	t.Helper()
	select {
	case got := <-requests:
		if got != want {
			t.Errorf("request = %+v, want %+v", got, want)
		}
	default:
		t.Fatalf("missing local request %+v", want)
	}
}

func assertNoNavigationRequests(t *testing.T, requests <-chan navigationRequest) {
	t.Helper()
	select {
	case request := <-requests:
		t.Errorf("unexpected request: %+v", request)
	default:
	}
}

func setNavigationCloud(t *testing.T) {
	t.Helper()
	original := viper.Get("installation")
	viper.Set("installation", jira.InstallationTypeCloud)
	t.Cleanup(func() { viper.Set("installation", original) })
}
