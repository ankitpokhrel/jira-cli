package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const searchResponseBody = `{
  "issues": [
    {
      "key": "TEST-1",
      "fields": {
        "summary": "First issue",
        "status": {"name": "To Do"},
        "issueType": {"name": "Task"},
        "priority": {"name": "Medium"},
        "assignee": {"displayName": "Alice"},
        "reporter": {"displayName": "Bob"},
        "created": "2026-01-01T10:00:00.000+0000",
        "updated": "2026-01-02T10:00:00.000+0000",
        "labels": []
      }
    }
  ]
}`

func newSearchTestDeps(t *testing.T, handler http.HandlerFunc) (*Deps, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	prevInstall := viper.GetString("installation")
	viper.Set("installation", jira.InstallationTypeCloud)

	deps := &Deps{
		Client:         client,
		Server:         server.URL,
		DefaultProject: "TEST",
		Installation:   jira.InstallationTypeCloud,
	}

	cleanup := func() {
		server.Close()
		viper.Set("installation", prevInstall)
	}
	return deps, cleanup
}

func TestSearchIssues_UsesProvidedJQL(t *testing.T) {
	var capturedJQL string
	deps, cleanup := newSearchTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
		capturedJQL = r.URL.Query().Get("jql")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(searchResponseBody))
	})
	defer cleanup()

	out, err := SearchIssues(context.Background(), deps, SearchIssuesInput{
		JQL: "summary ~ first",
	})
	require.NoError(t, err)

	assert.Equal(t, "summary ~ first", capturedJQL)
	require.Len(t, out.Issues, 1)
	assert.Equal(t, "TEST-1", out.Issues[0].Key)
	assert.Equal(t, "First issue", out.Issues[0].Summary)
	assert.Equal(t, "To Do", out.Issues[0].Status)
	assert.Equal(t, "Alice", out.Issues[0].Assignee)
	assert.True(t, strings.HasSuffix(out.Issues[0].URL, "/browse/TEST-1"))
}

func TestSearchIssues_ComposesJQLFromFilters(t *testing.T) {
	var capturedJQL string
	deps, cleanup := newSearchTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issues": []}`))
	})
	defer cleanup()

	_, err := SearchIssues(context.Background(), deps, SearchIssuesInput{
		Status:   "In Progress",
		Assignee: "alice",
	})
	require.NoError(t, err)

	assert.Contains(t, capturedJQL, `project = "TEST"`)
	assert.Contains(t, capturedJQL, `status = "In Progress"`)
	assert.Contains(t, capturedJQL, `assignee = "alice"`)
}

func TestSearchIssues_AssigneeMe(t *testing.T) {
	var capturedJQL string
	deps, cleanup := newSearchTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issues": []}`))
	})
	defer cleanup()

	_, err := SearchIssues(context.Background(), deps, SearchIssuesInput{Assignee: "me"})
	require.NoError(t, err)
	assert.Contains(t, capturedJQL, `assignee = currentUser()`)
}

func TestSearchIssues_LimitClampedTo100(t *testing.T) {
	var capturedLimit string
	deps, cleanup := newSearchTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		capturedLimit = r.URL.Query().Get("maxResults")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issues": []}`))
	})
	defer cleanup()

	_, err := SearchIssues(context.Background(), deps, SearchIssuesInput{Limit: 500})
	require.NoError(t, err)
	assert.Equal(t, "100", capturedLimit)
}

func TestSearchIssues_DefaultLimit(t *testing.T) {
	var capturedLimit string
	deps, cleanup := newSearchTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		capturedLimit = r.URL.Query().Get("maxResults")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issues": []}`))
	})
	defer cleanup()

	_, err := SearchIssues(context.Background(), deps, SearchIssuesInput{})
	require.NoError(t, err)
	assert.Equal(t, "50", capturedLimit)
}
