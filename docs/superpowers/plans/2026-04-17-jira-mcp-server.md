# Jira CLI MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Model Context Protocol (MCP) server to `jira-cli`, exposed as `jira mcp serve` over stdio, with five tools: `search_issues`, `get_issue`, `create_issue`, `add_comment`, `transition_issue`.

**Architecture:** Thin MCP layer in a new `internal/mcp/` package. Tool handlers depend only on `pkg/jira` (via the existing `api` package's v2/v3 proxies) and `pkg/adf`. A new `internal/cmd/mcp/` package wires `jira mcp serve` into the existing Cobra tree, builds dependencies from viper, and runs the SDK's stdio transport. Tool handlers are unit-tested with `httptest.NewServer` (matching `pkg/jira/*_test.go` style); the full server is round-tripped once with `mcp.NewInMemoryTransports`.

**Tech Stack:** Go 1.25, Cobra, Viper, `github.com/modelcontextprotocol/go-sdk` v1.5.0, `httptest`, `testify`.

**Spec:** `docs/superpowers/specs/2026-04-17-jira-mcp-server-design.md`

**Conventions used in this plan:**
- All file paths are relative to the repo root.
- All `go test` commands run from the repo root.
- All commits use the project's existing message style (`feat:`, `test:`, `docs:`).
- The MCP code path **must never write to stdout** (it would corrupt JSON-RPC framing). All logs go to stderr; tool results are returned as values.

---

## Task 0: Create empty package skeleton

**Files:**
- Create: `internal/mcp/doc.go`
- Create: `internal/mcp/tools/doc.go`

The MCP Go SDK is intentionally **not** added in this task. Adding it now would leave `go.mod` with a require line that no source justifies, which `go mod tidy` would revert. The SDK is added in Task 2, alongside the first source file that imports it.

- [ ] **Step 1: Create empty package marker for `internal/mcp`**

Create `internal/mcp/doc.go`:

```go
// Package mcp implements a Model Context Protocol server that exposes
// a subset of jira-cli's capabilities to MCP-aware hosts (e.g. Cursor,
// Claude Desktop). Wiring lives in internal/cmd/mcp.
package mcp
```

- [ ] **Step 2: Create empty package marker for `internal/mcp/tools`**

Create `internal/mcp/tools/doc.go`:

```go
// Package tools holds the individual MCP tool handlers. Each handler is
// a small adapter from a typed input/output struct onto the existing
// pkg/jira client. Handlers must not depend on cobra, viper, survey, or
// tui; their dependencies are injected via the Deps struct.
package tools
```

- [ ] **Step 3: Verify the module still builds**

```bash
go build ./...
```

Expected: exit 0, no output.

- [ ] **Step 4: Commit**

```bash
git add internal/mcp/doc.go internal/mcp/tools/doc.go
git commit -m "feat(mcp): add internal/mcp package skeleton"
```

---

## Task 1: Define the `tools.Deps` struct

**Files:**
- Create: `internal/mcp/tools/deps.go`
- Create: `internal/mcp/tools/deps_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/deps_test.go`:

```go
package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeps_IssueURL(t *testing.T) {
	d := &Deps{Server: "https://example.atlassian.net"}
	assert.Equal(t, "https://example.atlassian.net/browse/TEST-1", d.IssueURL("TEST-1"))
}

func TestDeps_IssueURL_TrimsTrailingSlash(t *testing.T) {
	d := &Deps{Server: "https://example.atlassian.net/"}
	assert.Equal(t, "https://example.atlassian.net/browse/TEST-1", d.IssueURL("TEST-1"))
}

func TestDeps_ResolveProject_UsesDefaultWhenEmpty(t *testing.T) {
	d := &Deps{DefaultProject: "ABC"}
	assert.Equal(t, "ABC", d.ResolveProject(""))
}

func TestDeps_ResolveProject_PrefersExplicit(t *testing.T) {
	d := &Deps{DefaultProject: "ABC"}
	assert.Equal(t, "XYZ", d.ResolveProject("XYZ"))
}

```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestDeps -v
```

Expected: FAIL — `Deps` undefined.

- [ ] **Step 3: Implement `Deps`**

Create `internal/mcp/tools/deps.go`:

```go
package tools

import (
	"strings"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Deps bundles the runtime dependencies every MCP tool handler needs.
// It is constructed once in internal/cmd/mcp/serve and shared (read-only)
// across all tool invocations.
type Deps struct {
	Client         *jira.Client
	Server         string
	DefaultProject string
	Installation   string
}

// IssueURL returns the browser URL for a given issue key.
func (d *Deps) IssueURL(key string) string {
	return strings.TrimRight(d.Server, "/") + "/browse/" + key
}

// ResolveProject returns explicit if non-empty, otherwise the configured
// default project key.
func (d *Deps) ResolveProject(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return d.DefaultProject
}

```

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestDeps -v
```

Expected: PASS, all 5 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/deps.go internal/mcp/tools/deps_test.go
git commit -m "feat(mcp): add tools.Deps with project/url/installation helpers"
```

---

## Task 2: Add the `search_issues` tool

**Files:**
- Create: `internal/mcp/tools/search_issues.go`
- Create: `internal/mcp/tools/search_issues_test.go`

The tool calls `api.ProxySearch`, which selects the v2 or v3 endpoint based on the configured installation type. The `api` package reads `viper.GetString("installation")` internally; tests set that via `viper.Set("installation", ...)`.

The MCP Go SDK is **not** added in this task either — Task 2's source files don't import it (only `api`, `context`, `fmt`, `strings`). The SDK fetch lands in Task 8, the first task whose source actually imports `github.com/modelcontextprotocol/go-sdk/mcp`.

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/search_issues_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestSearchIssues -v
```

Expected: FAIL — `SearchIssues`, `SearchIssuesInput` undefined.

- [ ] **Step 3: Implement the tool**

Create `internal/mcp/tools/search_issues.go`:

```go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ankitpokhrel/jira-cli/api"
)

// SearchIssuesInput is the input schema for the search_issues tool.
type SearchIssuesInput struct {
	JQL      string `json:"jql,omitempty" jsonschema:"raw JQL to execute. If set, other filter fields are ignored except project (which scopes the JQL when present)."`
	Project  string `json:"project,omitempty" jsonschema:"project key (defaults to the configured project)"`
	Status   string `json:"status,omitempty" jsonschema:"filter by status name, e.g. \"To Do\""`
	Assignee string `json:"assignee,omitempty" jsonschema:"filter by assignee. Use \"me\" for the configured user, \"none\" for unassigned, or a username/account id."`
	Limit    int    `json:"limit,omitempty" jsonschema:"maximum number of issues to return (default 50, clamped to 100)"`
}

// SearchIssuesOutput is the structured result of the search_issues tool.
type SearchIssuesOutput struct {
	Total  int          `json:"total"`
	Issues []IssueBrief `json:"issues"`
}

// IssueBrief is a lean projection of jira.Issue used for list-style outputs.
type IssueBrief struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Assignee string `json:"assignee"`
	Reporter string `json:"reporter"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
	URL      string `json:"url"`
}

// SearchIssues runs the search_issues tool.
func SearchIssues(_ context.Context, d *Deps, in SearchIssuesInput) (SearchIssuesOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	jql := strings.TrimSpace(in.JQL)

	if jql == "" {
		// No JQL → compose one from the simple filters, scoped to the resolved
		// (default-or-explicit) project so plain "list my open issues" calls
		// stay inside the configured project.
		jql = composeJQL(d.ResolveProject(in.Project), in.Status, in.Assignee)
	} else if in.Project != "" && !strings.Contains(strings.ToLower(jql), "project") {
		// JQL is a power-user escape hatch: pass it through unmodified by
		// default, and only wrap with a project clause when the caller
		// explicitly opted in by setting Project on the input.
		jql = fmt.Sprintf(`project = %q AND (%s)`, in.Project, jql)
	}

	res, err := api.ProxySearch(d.Client, jql, 0, uint(limit))
	if err != nil {
		return SearchIssuesOutput{}, err
	}

	out := SearchIssuesOutput{Issues: make([]IssueBrief, 0, len(res.Issues))}
	for _, iss := range res.Issues {
		out.Issues = append(out.Issues, IssueBrief{
			Key:      iss.Key,
			Summary:  iss.Fields.Summary,
			Status:   iss.Fields.Status.Name,
			Type:     iss.Fields.IssueType.Name,
			Priority: iss.Fields.Priority.Name,
			Assignee: iss.Fields.Assignee.Name,
			Reporter: iss.Fields.Reporter.Name,
			Created:  iss.Fields.Created,
			Updated:  iss.Fields.Updated,
			URL:      d.IssueURL(iss.Key),
		})
	}
	out.Total = len(out.Issues)
	return out, nil
}

func composeJQL(project, status, assignee string) string {
	var clauses []string
	if project != "" {
		clauses = append(clauses, fmt.Sprintf(`project = %q`, project))
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf(`status = %q`, status))
	}
	switch strings.ToLower(assignee) {
	case "":
		// no clause
	case "me":
		clauses = append(clauses, "assignee = currentUser()")
	case "none", "x":
		clauses = append(clauses, "assignee is EMPTY")
	default:
		clauses = append(clauses, fmt.Sprintf(`assignee = %q`, assignee))
	}
	if len(clauses) == 0 {
		return ""
	}
	return strings.Join(clauses, " AND ") + " ORDER BY created DESC"
}
```

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestSearchIssues -v
```

Expected: PASS, all 5 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/search_issues.go internal/mcp/tools/search_issues_test.go
git commit -m "feat(mcp): add search_issues tool"
```

---

## Task 3: Add the `bodyToMarkdown` helper for ADF/string conversion

The `Description` and comment `Body` fields in `pkg/jira` are typed as `interface{}`: `*adf.ADF` after a v3 fetch (post-`ifaceToADF`), `string` after a v2 fetch. The MCP tools need a single helper to render either to markdown.

**Files:**
- Create: `internal/mcp/tools/markdown.go`
- Create: `internal/mcp/tools/markdown_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/markdown_test.go`:

```go
package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

func TestBodyToMarkdown_String(t *testing.T) {
	assert.Equal(t, "hello", bodyToMarkdown("hello"))
}

func TestBodyToMarkdown_Nil(t *testing.T) {
	assert.Equal(t, "", bodyToMarkdown(nil))
}

func TestBodyToMarkdown_ADF(t *testing.T) {
	doc := &adf.ADF{
		Version: 1,
		DocType: "doc",
		Content: []*adf.Node{
			{
				NodeType: "paragraph",
				Content: []*adf.Node{
					{NodeType: "text", NodeValue: adf.NodeValue{Text: "Hello world"}},
				},
			},
		},
	}
	got := bodyToMarkdown(doc)
	assert.Contains(t, got, "Hello world")
}

func TestBodyToMarkdown_UnknownType(t *testing.T) {
	assert.Equal(t, "", bodyToMarkdown(123))
}
```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestBodyToMarkdown -v
```

Expected: FAIL — `bodyToMarkdown` undefined.

- [ ] **Step 3: Implement the helper**

Create `internal/mcp/tools/markdown.go`:

```go
package tools

import (
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

// bodyToMarkdown renders a Jira description-or-comment body field to markdown.
// The body is interface{} because v3 returns *adf.ADF and v2 returns string.
func bodyToMarkdown(body any) string {
	switch v := body.(type) {
	case nil:
		return ""
	case string:
		return v
	case *adf.ADF:
		if v == nil {
			return ""
		}
		return adf.NewTranslator(v, adf.NewMarkdownTranslator()).Translate()
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestBodyToMarkdown -v
```

Expected: PASS, all 4 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/markdown.go internal/mcp/tools/markdown_test.go
git commit -m "feat(mcp): add bodyToMarkdown helper for ADF/string description bodies"
```

---

## Task 4: Add the `get_issue` tool

**Files:**
- Create: `internal/mcp/tools/get_issue.go`
- Create: `internal/mcp/tools/get_issue_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/get_issue_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestGetIssue -v
```

Expected: FAIL — `GetIssue`, `GetIssueInput` undefined.

- [ ] **Step 3: Implement the tool**

Create `internal/mcp/tools/get_issue.go`:

```go
package tools

import (
	"context"
	"errors"

	"github.com/ankitpokhrel/jira-cli/api"
	issuefilter "github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)



// GetIssueInput is the input schema for the get_issue tool.
type GetIssueInput struct {
	Key             string `json:"key" jsonschema:"issue key, e.g. \"PROJ-123\" (required)"`
	IncludeComments *bool  `json:"include_comments,omitempty" jsonschema:"include recent comments in the response (default true)"`
	CommentLimit    int    `json:"comment_limit,omitempty" jsonschema:"maximum number of recent comments to include (default 10)"`
}

// GetIssueOutput is the structured result of the get_issue tool.
type GetIssueOutput struct {
	Key         string         `json:"key"`
	Summary     string         `json:"summary"`
	Status      string         `json:"status"`
	Type        string         `json:"type"`
	Priority    string         `json:"priority"`
	Assignee    string         `json:"assignee"`
	Reporter    string         `json:"reporter"`
	Labels      []string       `json:"labels"`
	Components  []string       `json:"components"`
	FixVersions []string       `json:"fix_versions"`
	Parent      string         `json:"parent,omitempty"`
	Created     string         `json:"created"`
	Updated     string         `json:"updated"`
	Description string         `json:"description"`
	Comments    []CommentBrief `json:"comments,omitempty"`
	URL         string         `json:"url"`
}

// CommentBrief is a lean projection of an issue comment.
type CommentBrief struct {
	ID      string `json:"id"`
	Author  string `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
}

// GetIssue runs the get_issue tool.
func GetIssue(_ context.Context, d *Deps, in GetIssueInput) (GetIssueOutput, error) {
	if in.Key == "" {
		return GetIssueOutput{}, errors.New("key is required")
	}

	includeComments := true
	if in.IncludeComments != nil {
		includeComments = *in.IncludeComments
	}
	commentLimit := in.CommentLimit
	if commentLimit <= 0 {
		commentLimit = 10
	}

	iss, err := api.ProxyGetIssue(d.Client, in.Key, issuefilter.NewNumCommentsFilter(uint(commentLimit)))
	if err != nil {
		return GetIssueOutput{}, err
	}

	out := GetIssueOutput{
		Key:         iss.Key,
		Summary:     iss.Fields.Summary,
		Status:      iss.Fields.Status.Name,
		Type:        iss.Fields.IssueType.Name,
		Priority:    iss.Fields.Priority.Name,
		Assignee:    iss.Fields.Assignee.Name,
		Reporter:    iss.Fields.Reporter.Name,
		Labels:      iss.Fields.Labels,
		Created:     iss.Fields.Created,
		Updated:     iss.Fields.Updated,
		Description: bodyToMarkdown(iss.Fields.Description),
		URL:         d.IssueURL(iss.Key),
	}
	if iss.Fields.Parent != nil {
		out.Parent = iss.Fields.Parent.Key
	}
	for _, c := range iss.Fields.Components {
		out.Components = append(out.Components, c.Name)
	}
	for _, v := range iss.Fields.FixVersions {
		out.FixVersions = append(out.FixVersions, v.Name)
	}

	if includeComments && iss.Fields.Comment.Total > 0 {
		comments := iss.Fields.Comment.Comments
		// Take the last commentLimit comments (newest), preserving original chronological order.
		start := 0
		if len(comments) > commentLimit {
			start = len(comments) - commentLimit
		}
		for i := start; i < len(comments); i++ {
			c := comments[i]
			out.Comments = append(out.Comments, CommentBrief{
				ID:      c.ID,
				Author:  c.Author.Name,
				Body:    bodyToMarkdown(c.Body),
				Created: c.Created,
			})
		}
	}

	return out, nil
}
```

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestGetIssue -v
```

Expected: PASS, all 3 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/get_issue.go internal/mcp/tools/get_issue_test.go
git commit -m "feat(mcp): add get_issue tool"
```

---

## Task 5: Add the `create_issue` tool

**Files:**
- Create: `internal/mcp/tools/create_issue.go`
- Create: `internal/mcp/tools/create_issue_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/create_issue_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestCreateIssue -v
```

Expected: FAIL — `CreateIssue`, `CreateIssueInput` undefined.

- [ ] **Step 3: Implement the tool**

Create `internal/mcp/tools/create_issue.go`:

```go
package tools

import (
	"context"
	"errors"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// CreateIssueInput is the input schema for the create_issue tool.
type CreateIssueInput struct {
	Summary     string   `json:"summary" jsonschema:"issue summary (required)"`
	Type        string   `json:"type" jsonschema:"issue type, e.g. \"Task\", \"Bug\", \"Story\" (required)"`
	Project     string   `json:"project,omitempty" jsonschema:"project key (defaults to the configured project)"`
	Description string   `json:"description,omitempty" jsonschema:"issue description in markdown"`
	Priority    string   `json:"priority,omitempty" jsonschema:"priority name, e.g. \"High\""`
	Labels      []string `json:"labels,omitempty"`
	Components  []string `json:"components,omitempty"`
	Assignee    string   `json:"assignee,omitempty" jsonschema:"assignee account id (Cloud) or username (Local)"`
	Parent      string   `json:"parent,omitempty" jsonschema:"parent issue key (use this for epic link or sub-task parent)"`
}

// CreateIssueOutput is the structured result of the create_issue tool.
type CreateIssueOutput struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// CreateIssue runs the create_issue tool.
func CreateIssue(_ context.Context, d *Deps, in CreateIssueInput) (CreateIssueOutput, error) {
	if in.Summary == "" {
		return CreateIssueOutput{}, errors.New("summary is required")
	}
	if in.Type == "" {
		return CreateIssueOutput{}, errors.New("type is required")
	}

	project := d.ResolveProject(in.Project)
	if project == "" {
		return CreateIssueOutput{}, errors.New("project is required (no default project configured)")
	}

	req := &jira.CreateRequest{
		Project:        project,
		IssueType:      in.Type,
		Summary:        in.Summary,
		Body:           in.Description,
		Priority:       in.Priority,
		Labels:         in.Labels,
		Components:     in.Components,
		Assignee:       in.Assignee,
		ParentIssueKey: in.Parent,
	}
	req.ForInstallationType(d.Installation)

	resp, err := api.ProxyCreate(d.Client, req)
	if err != nil {
		return CreateIssueOutput{}, err
	}
	return CreateIssueOutput{Key: resp.Key, URL: d.IssueURL(resp.Key)}, nil
}
```

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestCreateIssue -v
```

Expected: PASS, all 4 subtests. If a test fails because `jira.CreateRequest.Body` cannot accept a string for v3 (it's `interface{}` per the type def, but the V3 endpoint may require ADF), keep `Body` as the input string for now — the V3 API will accept plain text in many cases; the existing CLI also passes raw strings here. Adjust only if the test against the fake server actually fails on assertion of the body shape.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/create_issue.go internal/mcp/tools/create_issue_test.go
git commit -m "feat(mcp): add create_issue tool"
```

---

## Task 6: Add the `add_comment` tool

**Files:**
- Create: `internal/mcp/tools/add_comment.go`
- Create: `internal/mcp/tools/add_comment_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/add_comment_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestAddComment -v
```

Expected: FAIL — `AddComment`, `AddCommentInput` undefined.

- [ ] **Step 3: Implement the tool**

Create `internal/mcp/tools/add_comment.go`:

```go
package tools

import (
	"context"
	"errors"
)

// AddCommentInput is the input schema for the add_comment tool.
type AddCommentInput struct {
	Key      string `json:"key" jsonschema:"issue key, e.g. \"PROJ-123\" (required)"`
	Body     string `json:"body" jsonschema:"comment body in markdown (required)"`
	Internal bool   `json:"internal,omitempty" jsonschema:"mark as an internal (service-desk) comment"`
}

// AddCommentOutput is the structured result of the add_comment tool.
type AddCommentOutput struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// AddComment runs the add_comment tool.
func AddComment(_ context.Context, d *Deps, in AddCommentInput) (AddCommentOutput, error) {
	if in.Key == "" {
		return AddCommentOutput{}, errors.New("key is required")
	}
	if in.Body == "" {
		return AddCommentOutput{}, errors.New("body is required")
	}
	if err := d.Client.AddIssueComment(in.Key, in.Body, in.Internal); err != nil {
		return AddCommentOutput{}, err
	}
	return AddCommentOutput{Key: in.Key, URL: d.IssueURL(in.Key)}, nil
}
```

Note: `AddIssueComment` does not return the created comment ID, so the spec's `comment_id` field is dropped from the output. If a future spec revision needs it, add a thin wrapper that captures the response body.

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestAddComment -v
```

Expected: PASS, all 3 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/add_comment.go internal/mcp/tools/add_comment_test.go
git commit -m "feat(mcp): add add_comment tool"
```

---

## Task 7: Add the `transition_issue` tool

**Files:**
- Create: `internal/mcp/tools/transition_issue.go`
- Create: `internal/mcp/tools/transition_issue_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/tools/transition_issue_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

```bash
go test ./internal/mcp/tools/ -run TestTransitionIssue -v
```

Expected: FAIL — `TransitionIssue`, `TransitionIssueInput` undefined.

- [ ] **Step 3: Implement the tool**

Create `internal/mcp/tools/transition_issue.go`:

```go
package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// TransitionIssueInput is the input schema for the transition_issue tool.
type TransitionIssueInput struct {
	Key        string `json:"key" jsonschema:"issue key (required)"`
	Transition string `json:"transition" jsonschema:"target transition name, e.g. \"In Progress\" (required, case-insensitive)"`
	Comment    string `json:"comment,omitempty" jsonschema:"optional comment to add as part of the transition (workflow must allow it)"`
	Resolution string `json:"resolution,omitempty" jsonschema:"optional resolution name to set, e.g. \"Fixed\""`
	Assignee   string `json:"assignee,omitempty" jsonschema:"optional new assignee (account id on Cloud, username on Local)"`
}

// TransitionIssueOutput is the structured result of the transition_issue tool.
type TransitionIssueOutput struct {
	Key      string `json:"key"`
	ToStatus string `json:"to_status"`
	URL      string `json:"url"`
}

// TransitionIssue runs the transition_issue tool.
func TransitionIssue(_ context.Context, d *Deps, in TransitionIssueInput) (TransitionIssueOutput, error) {
	if in.Key == "" {
		return TransitionIssueOutput{}, errors.New("key is required")
	}
	if in.Transition == "" {
		return TransitionIssueOutput{}, errors.New("transition is required")
	}

	transitions, err := api.ProxyTransitions(d.Client, in.Key)
	if err != nil {
		return TransitionIssueOutput{}, err
	}

	var match *jira.Transition
	target := strings.ToLower(strings.TrimSpace(in.Transition))
	available := make([]string, 0, len(transitions))
	for _, t := range transitions {
		available = append(available, t.Name)
		if strings.ToLower(t.Name) == target {
			match = t
		}
	}
	if match == nil {
		return TransitionIssueOutput{}, fmt.Errorf(
			"unknown transition %q for %s. Valid transitions: %s",
			in.Transition, in.Key, strings.Join(available, ", "),
		)
	}

	req := &jira.TransitionRequest{
		Transition: &jira.TransitionRequestData{
			ID:   match.ID.String(),
			Name: match.Name,
		},
	}
	if in.Comment != "" {
		req.Update = &jira.TransitionRequestUpdate{}
		req.Update.Comment = append(req.Update.Comment, struct {
			Add struct {
				Body string `json:"body"`
			} `json:"add"`
		}{
			Add: struct {
				Body string `json:"body"`
			}{Body: in.Comment},
		})
	}
	if in.Resolution != "" || in.Assignee != "" {
		req.Fields = &jira.TransitionRequestFields{}
		if in.Resolution != "" {
			req.Fields.Resolution = &struct {
				Name string `json:"name"`
			}{Name: in.Resolution}
		}
		if in.Assignee != "" {
			req.Fields.Assignee = &struct {
				Name string `json:"name"`
			}{Name: in.Assignee}
		}
	}

	if _, err := d.Client.Transition(in.Key, req); err != nil {
		return TransitionIssueOutput{}, err
	}
	return TransitionIssueOutput{
		Key:      in.Key,
		ToStatus: match.Name,
		URL:      d.IssueURL(in.Key),
	}, nil
}
```

- [ ] **Step 4: Run the test and verify it passes**

```bash
go test ./internal/mcp/tools/ -run TestTransitionIssue -v
```

Expected: PASS, all 4 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools/transition_issue.go internal/mcp/tools/transition_issue_test.go
git commit -m "feat(mcp): add transition_issue tool"
```

---

## Task 8: Build the MCP server (registration + in-memory round trip test)

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/mcp/server.go`
- Create: `internal/mcp/server_test.go`

The server constructor takes a `*tools.Deps`, builds a `*mcp.Server`, and registers all five tools using the SDK's `mcp.AddTool` generic helper. Each registration adapts the `(d, in) -> (out, err)` tool function to the SDK's `(ctx, *CallToolRequest, In) -> (*CallToolResult, Out, error)` signature.

This is the first task whose source actually imports the MCP Go SDK, so the SDK dependency is added here. (Adding it earlier would have left `go.mod` in a state that `go mod tidy` would revert.)

- [ ] **Step 1: Add the official MCP Go SDK dependency**

```bash
go get github.com/modelcontextprotocol/go-sdk@v1.5.0
```

Expected: `go get` completes; `go.mod` gains `github.com/modelcontextprotocol/go-sdk v1.5.0`. Once Steps 2–5 below add the source files that import the SDK, `go mod tidy` will keep the requirement and (if previously listed as `// indirect`) promote it to a direct require. If `v1.5.0` is no longer the latest stable, use the latest stable v1.x.

- [ ] **Step 2: Write the failing test**

Create `internal/mcp/server_test.go`:

```go
package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ankitpokhrel/jira-cli/internal/mcp/tools"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestServer_ListsAllTools(t *testing.T) {
	prevInstall := viper.GetString("installation")
	viper.Set("installation", jira.InstallationTypeCloud)
	defer viper.Set("installation", prevInstall)

	jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"issues": []}`))
	}))
	defer jiraServer.Close()

	deps := &tools.Deps{
		Client:         jira.NewClient(jira.Config{Server: jiraServer.URL}, jira.WithTimeout(3*time.Second)),
		Server:         jiraServer.URL,
		DefaultProject: "TEST",
		Installation:   jira.InstallationTypeCloud,
	}

	srv := NewServer(deps)
	require.NotNil(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverT, clientT := mcp.NewInMemoryTransports()

	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Run(ctx, serverT) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	require.NoError(t, err)
	defer session.Close()

	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, tool := range listed.Tools {
		names[tool.Name] = true
	}
	for _, expected := range []string{
		"search_issues", "get_issue", "create_issue", "add_comment", "transition_issue",
	} {
		assert.True(t, names[expected], "expected tool %q to be registered", expected)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_issues",
		Arguments: map[string]any{"jql": "project = TEST"},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError, "search_issues should succeed against the fake server")

	// Validation errors must come back as IsError tool results, not transport errors.
	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_issue",
		Arguments: map[string]any{}, // missing required "key"
	})
	require.NoError(t, err)
	assert.True(t, res.IsError, "missing required key should produce a tool error result")
}
```

- [ ] **Step 3: Run the test and verify it fails**

```bash
go test ./internal/mcp/ -run TestServer -v
```

Expected: FAIL — `NewServer` undefined.

- [ ] **Step 4: Implement the server**

Create `internal/mcp/server.go`:

```go
package mcp

import (
	"context"
	"fmt"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ankitpokhrel/jira-cli/internal/mcp/tools"
)

const (
	// ServerName is the implementation name advertised over MCP.
	ServerName = "jira-cli"
	// ServerVersion is the MCP server version advertised to clients.
	// Bumped independently of the jira-cli release version when the MCP
	// surface changes in a backward-incompatible way.
	ServerVersion = "0.1.0"
)

// NewServer constructs a configured *mcp.Server with all jira-cli tools
// registered. The caller is responsible for invoking server.Run with a
// transport.
func NewServer(d *tools.Deps) *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, nil)

	registerTool(srv, "search_issues",
		"Search Jira issues by JQL or simple filters. Defaults to the configured project.",
		d, tools.SearchIssues)

	registerTool(srv, "get_issue",
		"Get full details of a Jira issue including description and recent comments.",
		d, tools.GetIssue)

	registerTool(srv, "create_issue",
		"Create a new Jira issue in the given project.",
		d, tools.CreateIssue)

	registerTool(srv, "add_comment",
		"Add a comment to a Jira issue.",
		d, tools.AddComment)

	registerTool(srv, "transition_issue",
		"Transition a Jira issue to a new status by name (e.g. \"In Progress\", \"Done\").",
		d, tools.TransitionIssue)

	return srv
}

// registerTool adapts a tools.* handler (which takes Deps + Input and returns
// Output + error) onto the SDK's expected handler signature. It also recovers
// from panics in the handler body so a single bad call cannot kill the server
// mid-session, and converts both errors and panics into MCP tool errors that
// the LLM can read.
func registerTool[In, Out any](
	srv *mcpsdk.Server,
	name, description string,
	d *tools.Deps,
	fn func(context.Context, *tools.Deps, In) (Out, error),
) {
	mcpsdk.AddTool(srv,
		&mcpsdk.Tool{Name: name, Description: description},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in In) (result *mcpsdk.CallToolResult, out Out, err error) {
			defer func() {
				if r := recover(); r != nil {
					var zero Out
					fmt.Fprintf(os.Stderr, "mcp: panic in tool %q: %v\n", name, r)
					result = &mcpsdk.CallToolResult{
						IsError: true,
						Content: []mcpsdk.Content{&mcpsdk.TextContent{
							Text: fmt.Sprintf("internal error in tool %q: %v", name, r),
						}},
					}
					out = zero
					err = nil
				}
			}()

			out, callErr := fn(ctx, d, in)
			if callErr != nil {
				var zero Out
				return &mcpsdk.CallToolResult{
					IsError: true,
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: callErr.Error()}},
				}, zero, nil
			}
			return nil, out, nil
		},
	)
}
```

- [ ] **Step 5: Run the test and verify it passes**

```bash
go test ./internal/mcp/ -run TestServer -v
```

Expected: PASS. If the SDK's exact symbol names differ slightly between v1.5.0 minor versions (e.g. `mcp.NewInMemoryTransports` vs `mcp.NewInMemoryTransport`), check `pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp` for the current API and adjust the test only — the production code in `server.go` follows the README example verbatim.

- [ ] **Step 6: Run the whole MCP test suite to confirm everything still passes**

```bash
go test ./internal/mcp/... -v
```

Expected: PASS, all suites.

- [ ] **Step 7: Commit**

Before committing, run `go mod tidy` to make sure `go.mod`/`go.sum` are minimal and that the SDK requirement is now recorded as a direct require (since `server.go` and `server_test.go` import it).

```bash
go mod tidy
git add go.mod go.sum internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat(mcp): add Server constructor and in-memory round-trip test"
```

---

## Task 9: Wire `jira mcp serve` into the Cobra tree

**Files:**
- Create: `internal/cmd/mcp/mcp.go`
- Create: `internal/cmd/mcp/serve/serve.go`
- Modify: `internal/cmd/root/root.go`

The `serve` command is the only place viper is read on the MCP path. It builds `tools.Deps` and runs the server over stdio. **It must not write to stdout**; all logging goes to stderr.

- [ ] **Step 1: Create the `mcp` parent command**

Create `internal/cmd/mcp/mcp.go`:

```go
package mcp

import (
	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/mcp/serve"
)

const helpText = `Run jira-cli as a Model Context Protocol (MCP) server, exposing
Jira operations to MCP-aware hosts (e.g. Cursor, Claude Desktop).`

// NewCmdMCP is the parent command for MCP-related subcommands.
func NewCmdMCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run jira-cli as an MCP server",
		Long:  helpText,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(serve.NewCmdServe())
	return cmd
}
```

- [ ] **Step 2: Create the `serve` subcommand**

Create `internal/cmd/mcp/serve/serve.go`:

```go
package serve

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	jiramcp "github.com/ankitpokhrel/jira-cli/internal/mcp"
	"github.com/ankitpokhrel/jira-cli/internal/mcp/tools"
)

const helpText = `Start an MCP server over stdio.

Configure your MCP host (Cursor, Claude Desktop, etc.) like this:

  {
    "mcpServers": {
      "jira": {
        "command": "jira",
        "args": ["mcp", "serve"],
        "env": { "JIRA_API_TOKEN": "..." }
      }
    }
  }

The server inherits the same configuration as every other jira-cli command:
JIRA_CONFIG_FILE, ~/.config/.jira/.config.yml, .netrc, and keychain all work
unchanged. The server reads from stdin and writes JSON-RPC frames to stdout;
all logs go to stderr.`

// NewCmdServe is the `jira mcp serve` command.
func NewCmdServe() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start an MCP server over stdio",
		Long:  helpText,
		RunE:  run,
	}
}

func run(cmd *cobra.Command, _ []string) error {
	server := viper.GetString("server")
	if server == "" {
		return fmt.Errorf("no Jira server configured. Run 'jira init' to set up the tool")
	}

	// Honor browse_server override the same way internal/cmdutil.GenerateServerBrowseURL does,
	// so MCP-emitted issue URLs match what the rest of the CLI produces for users whose web
	// client and API endpoints differ.
	browseServer := server
	if v := viper.GetString("browse_server"); v != "" {
		browseServer = v
	}

	debug := viper.GetBool("debug")
	deps := &tools.Deps{
		Client:         api.DefaultClient(debug),
		Server:         browseServer,
		DefaultProject: viper.GetString("project.key"),
		Installation:   viper.GetString("installation"),
	}

	srv := jiramcp.NewServer(deps)

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintln(os.Stderr, "jira-cli MCP server: listening on stdio")
	if err := srv.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Register the new command in root**

Modify `internal/cmd/root/root.go`. In the import block, add:

```go
"github.com/ankitpokhrel/jira-cli/internal/cmd/mcp"
```

In the `addChildCommands` function, add `mcp.NewCmdMCP()` to the call to `cmd.AddCommand`. After the change, the function looks like:

```go
func addChildCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		initCmd.NewCmdInit(),
		issue.NewCmdIssue(),
		epic.NewCmdEpic(),
		sprint.NewCmdSprint(),
		board.NewCmdBoard(),
		project.NewCmdProject(),
		open.NewCmdOpen(),
		me.NewCmdMe(),
		serverinfo.NewCmdServerInfo(),
		completion.NewCmdCompletion(),
		version.NewCmdVersion(),
		release.NewCmdRelease(),
		man.NewCmdMan(),
		mcp.NewCmdMCP(),
	)
}
```

- [ ] **Step 4: Allowlist `mcp` and `serve` in `cmdRequireToken`?**

No. The MCP server requires a configured Jira instance to do anything useful, and the existing token check (`PersistentPreRun` in root) is exactly what we want to fail early with a clear message. No change needed here.

- [ ] **Step 5: Verify the binary builds and the help text appears**

```bash
go build ./...
```

Expected: exit 0.

```bash
go run ./cmd/jira mcp --help
```

Expected: prints the parent help text, listing `serve` as a subcommand.

```bash
go run ./cmd/jira mcp serve --help
```

Expected: prints the serve help, including the JSON config snippet.

- [ ] **Step 6: Commit**

```bash
git add internal/cmd/mcp/mcp.go internal/cmd/mcp/serve/serve.go internal/cmd/root/root.go
git commit -m "feat(mcp): add 'jira mcp serve' cobra command (stdio transport)"
```

---

## Task 10: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Find the insertion point**

Open `README.md` and locate the `## Scripts` heading. The new section will be inserted *before* `## Scripts`, after the closing of `### Other commands`.

- [ ] **Step 2: Add the MCP section**

Insert the following new section immediately after the `### Other commands` block and immediately before the `## Scripts` heading. Use the existing markdown style (no emojis, plain headings).

```markdown
## MCP server

`jira-cli` ships an embedded [Model Context Protocol](https://modelcontextprotocol.io) server so MCP-aware hosts (Cursor, Claude Desktop, etc.) can read and modify Jira issues during a coding session. The server reuses the same config, auth, and Jira API client as the rest of the CLI.

Start it from your MCP host configuration:

```json
{
  "mcpServers": {
    "jira": {
      "command": "jira",
      "args": ["mcp", "serve"],
      "env": { "JIRA_API_TOKEN": "..." }
    }
  }
}
```

The server speaks stdio and exposes the following tools:

| Tool | Purpose |
| --- | --- |
| `search_issues` | Search by raw JQL or simple `status`/`assignee` filters. |
| `get_issue` | Full issue details including description and recent comments. |
| `create_issue` | Create a new issue in a project. |
| `add_comment` | Add a comment to an issue. |
| `transition_issue` | Move an issue to a new status by name. |

Every tool that returns an issue also returns its browser URL so the LLM can cite or link to it directly.
```

- [ ] **Step 3: Verify the README renders cleanly**

Open `README.md` in any markdown previewer (or just re-read the diff) and confirm:
- The new section appears between `### Other commands` and `## Scripts`.
- The fenced JSON block renders.
- The table renders.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: document the jira-cli MCP server"
```

---

## Task 11: Final verification

- [ ] **Step 1: Run all tests**

```bash
go test ./...
```

Expected: PASS across the whole repo. No new failures in pre-existing packages.

- [ ] **Step 2: Run the project's CI recipe**

```bash
make ci
```

Expected: lint + tests pass. If golangci-lint flags style issues in new files (e.g. missing comments on exported symbols), fix them.

- [ ] **Step 3: Smoke-test the binary end-to-end**

```bash
go install ./cmd/jira
jira mcp serve --help
```

Expected: help text appears, including the JSON snippet.

Optionally, with a configured Jira instance:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"0"}}}' | jira mcp serve
```

Expected: a JSON-RPC `initialize` response on stdout, server log lines on stderr, then the process exits when stdin closes.

- [ ] **Step 4: Confirm the git log is clean**

```bash
git log --oneline | head -15
```

Expected: a sequence of small, focused commits matching this plan's tasks. Nothing else.

---

## Notes for the implementer

- **Stdout discipline:** Search the new code for any accidental `fmt.Println`, `fmt.Printf`, or `os.Stdout` writes before merging. Stdout belongs exclusively to JSON-RPC.
- **Context cancellation is best-effort in v1:** `pkg/jira` high-level methods do not currently accept a `context.Context`. The MCP handlers receive a context from the SDK but cannot thread it into outbound HTTP calls. A 15-second client timeout still bounds individual requests. Adding context-accepting variants to `pkg/jira` is explicitly deferred per the spec.
- **`api` package and viper:** Tools call `api.Proxy*` functions, which read `viper.GetString("installation")` internally. Tests must set `viper.Set("installation", jira.InstallationTypeCloud)` (or `InstallationTypeLocal`) and restore the previous value. The helper functions in each test file already do this.
- **SDK API drift:** The plan targets `v1.5.0`. If the implementer pulls a newer version with breaking changes, the canonical reference is the README example at https://github.com/modelcontextprotocol/go-sdk — adapt only `server.go` and `server_test.go`; the rest of the code is SDK-independent.
