package list

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/rethab/jira-cli/pkg/jira"
)

const testEpicKey = "TEST-1"

// epicsResponse mimics the search endpoint returning issues of type epic.
const epicsResponse = `{
  "isLast": true,
  "issues": [
    {"key": "TEST-1", "fields": {"summary": "First epic"}},
    {"key": "TEST-2", "fields": {"summary": "Second epic"}}
  ]
}`

// epicIssuesResponse mimics the agile endpoint returning issues within an epic.
const epicIssuesResponse = `{
  "isLast": true,
  "issues": [
    {"key": "TEST-3", "fields": {"summary": "Issue in epic"}},
    {"key": "TEST-4", "fields": {"summary": "Another issue in epic"}}
  ]
}`

func testServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/search"):
			_, _ = w.Write([]byte(epicsResponse))
		case strings.Contains(r.URL.Path, "/epic/") && strings.HasSuffix(r.URL.Path, "/issue"):
			_, _ = w.Write([]byte(epicIssuesResponse))
		default:
			t.Errorf("unexpected request to %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// testCmd builds the real epic list command so that tests exercise actual flag
// registration, not a stand-in. The parent is required because the issue list
// only registers the column flags on subcommands.
func testCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()

	parent := &cobra.Command{Use: "epic"}
	cmd := &cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}}
	parent.AddCommand(cmd)

	SetFlags(cmd)
	// debug is a persistent flag on the root command, which tests don't build.
	cmd.Flags().Bool("debug", false, "")

	assert.NoError(t, cmd.ParseFlags(args))

	return cmd
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout

	r, w, err := os.Pipe()
	assert.NoError(t, err)

	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	assert.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	assert.NoError(t, err)

	return string(out)
}

func TestEpicExplorerViewRawOutputsEpicsAsJSON(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	cmd := testCmd(t, "--raw")

	out := captureStdout(t, func() {
		epicExplorerView(cmd, cmd.Flags(), "TEST", "", server.URL, client)
	})

	var got []*jira.Issue
	assert.NoError(t, json.Unmarshal([]byte(out), &got), "output is not valid JSON: %s", out)

	assert.Len(t, got, 2)
	assert.Equal(t, "TEST-1", got[0].Key)
	assert.Equal(t, "First epic", got[0].Fields.Summary)
}

func TestSingleEpicViewRawOutputsIssuesAsJSON(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	cmd := testCmd(t, "--raw")

	out := captureStdout(t, func() {
		singleEpicView(cmd.Flags(), testEpicKey, "TEST", "", server.URL, client)
	})

	var got []*jira.Issue
	assert.NoError(t, json.Unmarshal([]byte(out), &got), "output is not valid JSON: %s", out)

	assert.Len(t, got, 2)
	assert.Equal(t, "TEST-3", got[0].Key)
	assert.Equal(t, "Issue in epic", got[0].Fields.Summary)
}

// Without --raw the view must keep rendering the table output.
func TestSingleEpicViewWithoutRawRendersTable(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	cmd := testCmd(t, "--plain", "--no-headers")

	out := captureStdout(t, func() {
		singleEpicView(cmd.Flags(), testEpicKey, "TEST", "", server.URL, client)
	})

	assert.False(t, json.Valid([]byte(out)), "expected table output, got JSON: %s", out)
	assert.Contains(t, out, "TEST-3")
}
