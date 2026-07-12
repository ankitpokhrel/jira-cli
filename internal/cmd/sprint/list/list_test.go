//nolint:dupl
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

	"github.com/rethab/jira-cli/internal/query"
	"github.com/rethab/jira-cli/pkg/jira"
)

const (
	testBoardID  = 1
	testSprintID = 42
)

// sprintsResponse mimics the board sprints endpoint of the agile API.
const sprintsResponse = `{
  "maxResults": 50,
  "startAt": 0,
  "isLast": true,
  "values": [
    {"id": 42, "name": "Sprint 42", "state": "active", "startDate": "2020-11-15T05:39:24.463Z", "endDate": "2020-11-29T05:39:24.463Z"},
    {"id": 41, "name": "Sprint 41", "state": "closed", "startDate": "2020-11-01T05:39:24.463Z", "endDate": "2020-11-14T05:39:24.463Z"}
  ]
}`

// sprintIssuesResponse mimics the sprint issues endpoint of the agile API.
const sprintIssuesResponse = `{
  "isLast": true,
  "issues": [
    {"key": "TEST-1", "fields": {"summary": "First issue"}},
    {"key": "TEST-2", "fields": {"summary": "Second issue"}}
  ]
}`

// testServer serves the agile endpoints used by the sprint list views.
func testServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/board/1/sprint"):
			_, _ = w.Write([]byte(sprintsResponse))
		case strings.Contains(r.URL.Path, "/sprint/") && strings.HasSuffix(r.URL.Path, "/issue"):
			_, _ = w.Write([]byte(sprintIssuesResponse))
		default:
			t.Errorf("unexpected request to %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// testFlags builds the real flag set of the sprint list command so that tests
// exercise actual flag registration, not a stand-in.
func testFlags(t *testing.T, args ...string) query.FlagParser {
	t.Helper()

	cmd := &cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}}
	SetFlags(cmd)
	// debug is a persistent flag on the root command, which tests don't build.
	cmd.Flags().Bool("debug", false, "")

	assert.NoError(t, cmd.ParseFlags(args))

	return cmd.Flags()
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

func TestSprintExplorerViewRawOutputsSprintsAsJSON(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	flags := testFlags(t, "--raw")
	sprintQuery, err := query.NewSprint(flags)
	assert.NoError(t, err)

	out := captureStdout(t, func() {
		sprintExplorerView(sprintQuery, flags, testBoardID, "TEST", server.URL, client)
	})

	var got []*jira.Sprint
	assert.NoError(t, json.Unmarshal([]byte(out), &got), "output is not valid JSON: %s", out)

	assert.Len(t, got, 2)
	assert.Equal(t, "Sprint 41", got[0].Name)
	assert.Equal(t, "closed", got[0].Status)
	assert.Equal(t, 42, got[1].ID)
}

func TestSingleSprintViewRawOutputsIssuesAsJSON(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	flags := testFlags(t, "--raw")
	sprintQuery, err := query.NewSprint(flags)
	assert.NoError(t, err)

	out := captureStdout(t, func() {
		singleSprintView(sprintQuery, flags, testBoardID, testSprintID, "TEST", server.URL, client, nil)
	})

	var got []*jira.Issue
	assert.NoError(t, json.Unmarshal([]byte(out), &got), "output is not valid JSON: %s", out)

	assert.Len(t, got, 2)
	assert.Equal(t, "TEST-1", got[0].Key)
	assert.Equal(t, "First issue", got[0].Fields.Summary)
}

// --current (like --prev and --next) delegates to the single sprint view, so
// --raw must survive that hand-off and print the sprint's issues.
func TestSprintExplorerViewRawWithCurrentOutputsIssuesAsJSON(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	flags := testFlags(t, "--raw", "--current")
	sprintQuery, err := query.NewSprint(flags)
	assert.NoError(t, err)

	out := captureStdout(t, func() {
		sprintExplorerView(sprintQuery, flags, testBoardID, "TEST", server.URL, client)
	})

	var got []*jira.Issue
	assert.NoError(t, json.Unmarshal([]byte(out), &got), "output is not valid JSON: %s", out)

	assert.Len(t, got, 2)
	assert.Equal(t, "TEST-1", got[0].Key)
}

// Without --raw the views must keep rendering the table output.
func TestSprintViewsWithoutRawRenderTable(t *testing.T) {
	server := testServer(t)
	defer server.Close()

	client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))

	t.Run("explorer view", func(t *testing.T) {
		flags := testFlags(t, "--table", "--plain", "--no-headers")
		sprintQuery, err := query.NewSprint(flags)
		assert.NoError(t, err)

		out := captureStdout(t, func() {
			sprintExplorerView(sprintQuery, flags, testBoardID, "TEST", server.URL, client)
		})

		assert.False(t, json.Valid([]byte(out)), "expected table output, got JSON: %s", out)
		assert.Contains(t, out, "Sprint 42")
	})

	t.Run("single sprint view", func(t *testing.T) {
		flags := testFlags(t, "--plain", "--no-headers")
		sprintQuery, err := query.NewSprint(flags)
		assert.NoError(t, err)

		out := captureStdout(t, func() {
			singleSprintView(sprintQuery, flags, testBoardID, testSprintID, "TEST", server.URL, client, nil)
		})

		assert.False(t, json.Valid([]byte(out)), "expected table output, got JSON: %s", out)
		assert.Contains(t, out, "TEST-1")
	})
}

func TestResolveBoardID(t *testing.T) {
	tests := []struct {
		name         string
		configuredID int
		override     int
		overridden   bool
		want         int
	}{
		{name: "flag not given uses configured board", configuredID: 10, override: 0, overridden: false, want: 10},
		{name: "flag replaces configured board", configuredID: 10, override: 20, overridden: true, want: 20},
		{name: "flag is honored even when set to zero", configuredID: 10, override: 0, overridden: true, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBoardID(tc.configuredID, tc.override, tc.overridden)
			if got != tc.want {
				t.Errorf("resolveBoardID(%d, %d, %t) = %d, want %d", tc.configuredID, tc.override, tc.overridden, got, tc.want)
			}
		})
	}
}

func TestResolveBoardName(t *testing.T) {
	tests := []struct {
		name           string
		configuredID   int
		configuredName string
		boardID        int
		want           string
	}{
		{
			name:           "configured board keeps its name",
			configuredID:   10,
			configuredName: "My Board",
			boardID:        10,
			want:           "My Board",
		},
		{
			name:           "overridden board shows its ID instead of the configured name",
			configuredID:   10,
			configuredName: "My Board",
			boardID:        20,
			want:           "#20",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBoardName(tc.configuredID, tc.configuredName, tc.boardID)
			if got != tc.want {
				t.Errorf("resolveBoardName(%d, %q, %d) = %q, want %q", tc.configuredID, tc.configuredName, tc.boardID, got, tc.want)
			}
		})
	}
}

func TestShouldRenderSprintsInTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		table    bool
		plain    bool
		expected bool
	}{
		{
			name:     "neither table nor plain is set and terminal is interactive",
			table:    false,
			plain:    false,
			expected: false,
		},
		{
			name:     "table alone implies table view",
			table:    true,
			plain:    false,
			expected: true,
		},
		{
			name:     "plain alone implies table view",
			table:    false,
			plain:    true,
			expected: true,
		},
		{
			name:     "table and plain both set",
			table:    true,
			plain:    true,
			expected: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// dumbTerminal and notTTY are held false here to isolate the
			// effect of the table/plain flags from the actual test runner
			// environment.
			assert.Equal(t, tc.expected, shouldRenderSprintsInTable(tc.table, tc.plain, false, false))
		})
	}

	t.Run("dumb terminal or non-tty always implies table view", func(t *testing.T) {
		t.Parallel()

		assert.True(t, shouldRenderSprintsInTable(false, false, true, false))
		assert.True(t, shouldRenderSprintsInTable(false, false, false, true))
	})
}
