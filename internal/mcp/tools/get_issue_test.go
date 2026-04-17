package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const getIssueResponseV3 = `{
  "key": "TEST-1",
  "fields": {
    "summary": "Sample bug",
    "status": {"name": "In Progress"},
    "issueType": {"name": "Bug"},
    "priority": {"name": "High"},
    "assignee": {"displayName": "Alice"},
    "reporter": {"displayName": "Bob"},
    "labels": ["backend", "urgent"],
    "components": [{"name": "API"}],
    "fixVersions": [{"name": "v2.0"}],
    "created": "2026-01-01T10:00:00.000+0000",
    "updated": "2026-01-02T10:00:00.000+0000",
    "description": {
      "version": 1,
      "type": "doc",
      "content": [
        {"type": "paragraph", "content": [{"type": "text", "text": "Repro steps"}]}
      ]
    },
    "comment": {
      "total": 1,
      "comments": [
        {
          "id": "100",
          "author": {"displayName": "Carol", "emailAddress": "carol@example.com", "active": true},
          "body": {
            "version": 1,
            "type": "doc",
            "content": [
              {"type": "paragraph", "content": [{"type": "text", "text": "Looking into it"}]}
            ]
          },
          "created": "2026-01-03T10:00:00.000+0000"
        }
      ]
    }
  }
}`

func newIssueTestDeps(t *testing.T, handler http.HandlerFunc) (*Deps, func()) {
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
	return deps, func() {
		server.Close()
		viper.Set("installation", prevInstall)
	}
}

func TestGetIssue_Cloud(t *testing.T) {
	deps, cleanup := newIssueTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(getIssueResponseV3))
	})
	defer cleanup()

	out, err := GetIssue(context.Background(), deps, GetIssueInput{Key: "TEST-1"})
	require.NoError(t, err)

	assert.Equal(t, "TEST-1", out.Key)
	assert.Equal(t, "Sample bug", out.Summary)
	assert.Equal(t, "In Progress", out.Status)
	assert.Equal(t, "Bug", out.Type)
	assert.Equal(t, "High", out.Priority)
	assert.Equal(t, "Alice", out.Assignee)
	assert.Equal(t, "Bob", out.Reporter)
	assert.Equal(t, []string{"backend", "urgent"}, out.Labels)
	assert.Equal(t, []string{"API"}, out.Components)
	assert.Equal(t, []string{"v2.0"}, out.FixVersions)
	assert.Contains(t, out.Description, "Repro steps")
	require.Len(t, out.Comments, 1)
	assert.Equal(t, "Carol", out.Comments[0].Author)
	assert.Contains(t, out.Comments[0].Body, "Looking into it")
	assert.Equal(t, deps.IssueURL("TEST-1"), out.URL)
}

func TestGetIssue_RequiresKey(t *testing.T) {
	deps, cleanup := newIssueTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called when key is missing")
		w.WriteHeader(500)
	})
	defer cleanup()

	_, err := GetIssue(context.Background(), deps, GetIssueInput{Key: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key is required")
}

func TestGetIssue_RespectsCommentLimit(t *testing.T) {
	deps, cleanup := newIssueTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(getIssueResponseV3))
	})
	defer cleanup()

	out, err := GetIssue(context.Background(), deps, GetIssueInput{
		Key:             "TEST-1",
		IncludeComments: boolPtr(false),
	})
	require.NoError(t, err)
	assert.Empty(t, out.Comments)
}

func boolPtr(b bool) *bool { return &b }

const getIssueResponseV2 = `{
  "key": "TEST-1",
  "fields": {
    "summary": "Sample bug",
    "status": {"name": "In Progress"},
    "issueType": {"name": "Bug"},
    "priority": {"name": "High"},
    "assignee": {"displayName": "Alice"},
    "reporter": {"displayName": "Bob"},
    "labels": [],
    "components": [],
    "fixVersions": [],
    "created": "2026-01-01T10:00:00.000+0000",
    "updated": "2026-01-02T10:00:00.000+0000",
    "description": "Repro steps from wiki markup",
    "comment": {"total": 0, "comments": []}
  }
}`

func TestGetIssue_Local(t *testing.T) {
	var capturedPath string
	deps, cleanup := newIssueTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(getIssueResponseV2))
	})
	defer cleanup()

	viper.Set("installation", jira.InstallationTypeLocal)
	deps.Installation = jira.InstallationTypeLocal

	out, err := GetIssue(context.Background(), deps, GetIssueInput{Key: "TEST-1"})
	require.NoError(t, err)

	assert.Equal(t, "/rest/api/2/issue/TEST-1", capturedPath)
	assert.Equal(t, "TEST-1", out.Key)
	assert.Equal(t, "In Progress", out.Status)
	assert.Equal(t, "Repro steps from wiki markup", out.Description)
}
