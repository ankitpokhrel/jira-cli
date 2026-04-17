package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func newCreateTestDeps(t *testing.T, handler http.HandlerFunc) (*Deps, func()) {
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

func TestCreateIssue_Success(t *testing.T) {
	var capturedBody map[string]any
	deps, cleanup := newCreateTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "10001", "key": "TEST-42"}`))
	})
	defer cleanup()

	out, err := CreateIssue(context.Background(), deps, CreateIssueInput{
		Summary: "New thing",
		Type:    "Task",
	})
	require.NoError(t, err)
	assert.Equal(t, "TEST-42", out.Key)
	assert.Equal(t, deps.IssueURL("TEST-42"), out.URL)

	fields, _ := capturedBody["fields"].(map[string]any)
	require.NotNil(t, fields)
	project, _ := fields["project"].(map[string]any)
	assert.Equal(t, "TEST", project["key"])
	assert.Equal(t, "New thing", fields["summary"])
}

func TestCreateIssue_RequiresSummary(t *testing.T) {
	deps, cleanup := newCreateTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called")
		w.WriteHeader(500)
	})
	defer cleanup()

	_, err := CreateIssue(context.Background(), deps, CreateIssueInput{Type: "Task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "summary is required")
}

func TestCreateIssue_RequiresType(t *testing.T) {
	deps, cleanup := newCreateTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called")
		w.WriteHeader(500)
	})
	defer cleanup()

	_, err := CreateIssue(context.Background(), deps, CreateIssueInput{Summary: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type is required")
}

func TestCreateIssue_OverridesProject(t *testing.T) {
	var capturedProject string
	deps, cleanup := newCreateTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		fields, _ := body["fields"].(map[string]any)
		project, _ := fields["project"].(map[string]any)
		capturedProject, _ = project["key"].(string)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "10001", "key": "OTHER-1"}`))
	})
	defer cleanup()

	_, err := CreateIssue(context.Background(), deps, CreateIssueInput{
		Summary: "x", Type: "Task", Project: "OTHER",
	})
	require.NoError(t, err)
	assert.Equal(t, "OTHER", capturedProject)
}
