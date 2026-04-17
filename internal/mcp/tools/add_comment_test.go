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

func newCommentTestDeps(t *testing.T, handler http.HandlerFunc) (*Deps, func()) {
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

func TestAddComment_Success(t *testing.T) {
	deps, cleanup := newCommentTestDeps(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1/comment", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "999"}`))
	})
	defer cleanup()

	out, err := AddComment(context.Background(), deps, AddCommentInput{
		Key:  "TEST-1",
		Body: "Hello world",
	})
	require.NoError(t, err)
	assert.Equal(t, "TEST-1", out.Key)
	assert.Equal(t, deps.IssueURL("TEST-1"), out.URL)
}

func TestAddComment_RequiresKey(t *testing.T) {
	deps, cleanup := newCommentTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called")
		w.WriteHeader(500)
	})
	defer cleanup()

	_, err := AddComment(context.Background(), deps, AddCommentInput{Body: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key is required")
}

func TestAddComment_RequiresBody(t *testing.T) {
	deps, cleanup := newCommentTestDeps(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called")
		w.WriteHeader(500)
	})
	defer cleanup()

	_, err := AddComment(context.Background(), deps, AddCommentInput{Key: "TEST-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body is required")
}
