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

const transitionsResponse = `{
  "transitions": [
    {"id": "11", "name": "To Do", "isAvailable": true},
    {"id": "21", "name": "In Progress", "isAvailable": true},
    {"id": "31", "name": "Done", "isAvailable": true}
  ]
}`

func newTransitionTestDeps(t *testing.T, handler http.HandlerFunc) (*Deps, func()) {
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

func TestTransitionIssue_Success(t *testing.T) {
	var postedID string
	deps, cleanup := newTransitionTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			assert.Equal(t, "/rest/api/3/issue/TEST-1/transitions", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(transitionsResponse))
		case http.MethodPost:
			assert.Equal(t, "/rest/api/2/issue/TEST-1/transitions", r.URL.Path)
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			tr, _ := body["transition"].(map[string]any)
			postedID, _ = tr["id"].(string)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	defer cleanup()

	out, err := TransitionIssue(context.Background(), deps, TransitionIssueInput{
		Key:        "TEST-1",
		Transition: "In Progress",
	})
	require.NoError(t, err)
	assert.Equal(t, "21", postedID)
	assert.Equal(t, "TEST-1", out.Key)
	assert.Equal(t, "In Progress", out.ToStatus)
	assert.Equal(t, deps.IssueURL("TEST-1"), out.URL)
}

func TestTransitionIssue_CaseInsensitiveMatch(t *testing.T) {
	var postedID string
	deps, cleanup := newTransitionTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(transitionsResponse))
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		tr, _ := body["transition"].(map[string]any)
		postedID, _ = tr["id"].(string)
		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	_, err := TransitionIssue(context.Background(), deps, TransitionIssueInput{
		Key:        "TEST-1",
		Transition: "in progress",
	})
	require.NoError(t, err)
	assert.Equal(t, "21", postedID)
}

func TestTransitionIssue_UnknownTransition(t *testing.T) {
	deps, cleanup := newTransitionTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "POST should not happen for unknown transition")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(transitionsResponse))
	})
	defer cleanup()

	_, err := TransitionIssue(context.Background(), deps, TransitionIssueInput{
		Key:        "TEST-1",
		Transition: "Doing",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transition")
	assert.Contains(t, err.Error(), "To Do")
	assert.Contains(t, err.Error(), "In Progress")
	assert.Contains(t, err.Error(), "Done")
}

func TestTransitionIssue_RequiresKeyAndTransition(t *testing.T) {
	deps, cleanup := newTransitionTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called")
		w.WriteHeader(500)
	})
	defer cleanup()

	_, err := TransitionIssue(context.Background(), deps, TransitionIssueInput{Transition: "Done"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key is required")

	_, err = TransitionIssue(context.Background(), deps, TransitionIssueInput{Key: "TEST-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transition is required")
}
