# Jira CLI MCP Server — Design

**Status:** Approved (brainstorm) — pending implementation plan
**Date:** 2026-04-17
**Scope:** v1 of an MCP server integrated into `jira-cli`, intended to be upstreamed to `ankitpokhrel/jira-cli`.

## Goal

Add a Model Context Protocol (MCP) server to `jira-cli` so that an LLM running in an MCP host (Cursor, Claude Desktop, etc.) can read and modify Jira issues during a coding session. The server reuses the CLI's existing config, auth, and Jira API client.

## Non-goals (v1)

- Multi-user / hosted deployment.
- HTTP/SSE transport (stdio only for v1; design leaves room to add later).
- Mirroring the entire CLI surface (see "Tool surface" for the v1 set).
- MCP resources or prompts (tools only).
- Any read-only mode flag, dry-run mode, or extra confirmation gate beyond what the MCP host already provides.

## Use case

Single-user, IDE-integrated coding assistant. The user runs `jira mcp serve` from an MCP host configured to spawn it over stdio. The LLM uses the tools to look up tickets it's working on, file new ones, comment, and transition state — all gated by the host's own per-tool prompts.

## Approach

**Thin MCP layer over `pkg/jira`.** A new `internal/mcp/` package builds an MCP server using the official `github.com/modelcontextprotocol/go-sdk`. Each tool is a small adapter that calls the existing `pkg/jira` client. No refactor of the existing `internal/cmd/...` layer; no shelling out to the `jira` binary.

This keeps the diff small, isolates MCP code from the TUI/Cobra/Survey machinery in `internal/cmd/...`, and makes the tool layer trivially unit-testable.

## Package layout

```
internal/cmd/mcp/                   // NEW: cobra command surface
    mcp.go                          //   `jira mcp` parent cmd
    serve/serve.go                  //   `jira mcp serve` (wires SDK + tools, stdio)
internal/mcp/                       // NEW: MCP server core
    server.go                       //   builds *mcp.Server, registers tools
    tools/
        deps.go                     //   shared Deps struct + helpers
        search_issues.go
        get_issue.go
        create_issue.go
        add_comment.go
        transition_issue.go
        <name>_test.go              //   per-tool unit tests
    server_test.go                  //   in-memory transport round-trip test
```

Wiring:
- `internal/cmd/root` registers the new `mcp` parent command alongside `issue`, `epic`, etc.
- `internal/cmd/mcp/serve` constructs `tools.Deps` from `viper` + `api.DefaultClient` and hands it to `internal/mcp.NewServer(deps)`, then runs the server over stdio.

Dependency rules:
- `internal/mcp/tools/*` may import `pkg/jira`, `pkg/adf`, and standard library only. No `cobra`, `viper`, `survey`, or `tui`.
- All viper reads live in `internal/cmd/mcp/serve`. Tools receive their dependencies through `tools.Deps`.

## New external dependency

`github.com/modelcontextprotocol/go-sdk` — the official MCP Go SDK (Tier 1, maintained with Google). Used for the server skeleton, schema derivation from Go structs, and the stdio transport. One new top-level dep; acceptable for upstream.

## Tool surface (v1)

All inputs are JSON-Schema-derived from Go structs via the SDK's reflection. All outputs are JSON for structured fields, with the issue browser URL included where applicable so the LLM can cite/link.

### `search_issues`

- **Input:** `{ jql?: string, project?: string, status?: string, assignee?: string, limit?: int (default 50, max 100) }`
- **Behavior:** If `jql` is provided, use it verbatim, scoped to `project` if set (else the configured project). Otherwise compose JQL from the other filters. Calls `client.Search`.
- **Output:** `{ total: int, issues: [{ key, summary, status, type, priority, assignee, reporter, created, updated, url }] }`. Lean rows — no description or comments.

### `get_issue`

- **Input:** `{ key: string (required), include_comments?: bool (default true), comment_limit?: int (default 10) }`
- **Behavior:** `client.GetIssue(key)` plus comment fetch. ADF description converted to markdown via `pkg/adf`.
- **Output:** Full issue: `key, summary, status, type, priority, assignee, reporter, labels, components, fix_versions, parent, links, description (markdown), comments [{ author, body, created }], url`.

### `create_issue`

- **Input:** `{ summary: string (required), type: string (required, e.g. "Task"|"Bug"|"Story"), project?: string, description?: string, priority?: string, labels?: string[], components?: string[], assignee?: string, parent?: string }`
- **Behavior:** Selects `client.Create` vs `client.CreateV2` based on installation type, matching the existing create command's logic.
- **Output:** `{ key, url }`.

### `add_comment`

- **Input:** `{ key: string (required), body: string (required), internal?: bool }`
- **Behavior:** Calls the appropriate add-comment method on the client based on installation type.
- **Output:** `{ key, comment_id, url }`.

### `transition_issue`

- **Input:** `{ key: string (required), transition: string (required, e.g. "In Progress"|"Done"), comment?: string, resolution?: string, assignee?: string }`
- **Behavior:** Resolves transition name → id via `client.GetTransitions`. If the name is unknown, returns a tool error listing valid transitions. Then calls `client.Transition`.
- **Output:** `{ key, from_status, to_status, url }`.

## Cross-cutting behavior

- **Project defaulting:** When a tool's `project` field is omitted, fall back to `viper.GetString("project.key")`. Only required to be passed for cross-project work.
- **URLs:** Every output that references an issue includes `{server}/browse/{key}` so the LLM can cite/link in chat.
- **Concurrency:** The SDK invokes tool handlers concurrently. The `pkg/jira` client is already safe for concurrent use (HTTP client under the hood). No additional locking.
- **Cancellation:** Each handler receives a `context.Context` from the SDK. It is threaded into outbound HTTP calls so a host-cancelled tool call also cancels the upstream Jira request. If `pkg/jira` does not currently accept a context on the relevant methods, add a context-accepting variant (smallest possible change) rather than refactoring the existing API.

## Lifecycle and transport

`jira mcp serve`:

1. Cobra command parses inherited globals (`--config`, `--debug`); no MCP-specific flags in v1.
2. Reads viper config the same way every other `jira` subcommand does, so `JIRA_API_TOKEN`, `JIRA_CONFIG_FILE`, `~/.config/.jira/.config.yml`, `.netrc`, and keychain auth all keep working unchanged.
3. Constructs `api.DefaultClient(debug)` once, builds `tools.Deps{ Client, Project, Server, Installation }`.
4. Builds `mcp.NewServer(...)`, registers the five tools, and calls `server.Run(ctx, mcp.NewStdioTransport())`.
5. Blocks until stdin closes or SIGINT/SIGTERM is received.

**Stdout discipline:** With stdio transport, stdout is reserved exclusively for JSON-RPC frames. The MCP code path must not call any of the CLI's stdout printers; it returns structured tool results as values and emits all logs to stderr.

**Failing fast:** If config is missing at startup (e.g. no config file and no `JIRA_API_TOKEN`), the command fails before the MCP handshake with a message pointing at `jira init`. Better than appearing connected and breaking on first call.

## Configuration in MCP hosts

Documented snippet for Cursor / Claude Desktop:

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

This snippet appears in both the README and `jira mcp serve --help`.

## Error handling

Three categories, each handled differently:

1. **Bad input from the LLM** (missing required field, unknown transition name, malformed JQL): returned as a structured MCP tool error (`isError: true`) with a clear message and, where useful, the list of valid options. Example: `transition_issue` with `transition: "Doing"` returns `"Unknown transition 'Doing' for ISSUE-1. Valid transitions: To Do, In Progress, Done"`. The LLM can self-correct on the next turn.
2. **Jira API errors** (4xx/5xx from upstream): pass the upstream error message through to the tool result so the LLM can reason about it. Auth failures get a one-liner pointing at `JIRA_API_TOKEN` so the user can fix their setup.
3. **Server-internal errors**: logged to stderr with full context. Tool handlers wrap their body in `defer recover()` that converts panics into tool errors so a single bad call cannot kill the server mid-session.

## Testing

Mirroring the existing repo's pattern (`*_test.go` next to source, `testify`):

- **Per-tool unit tests** in `internal/mcp/tools/<name>_test.go`. Each test uses a fake `pkg/jira` client (matching whatever fake/`httptest` pattern `pkg/jira/*_test.go` already uses). Coverage targets: happy path, required-field validation, default-project fallback, error passthrough, and (for `transition_issue`) the unknown-transition case.
- **One integration test** in `internal/mcp/server_test.go` using `mcp.NewInMemoryTransports()` to exercise the round-trip `tools/list` plus a `tools/call` for one read tool and one write tool. Catches schema-derivation and SDK-API drift.
- No live-Jira tests in CI (matches the rest of the repo).

Coverage density should match `internal/cmd/issue/*_test.go`.

## Documentation

- **README:** New top-level "MCP server" section after "Scripts", with the host config snippet, the list of five tools, and a short example transcript.
- **`jira mcp serve --help`:** Includes the host config snippet inline so users can find it from the CLI.

## Rollout

Single PR containing the new packages, tests, and README update. Self-contained; the upstream maintainer can evaluate it as a unit. No changes to existing commands or to the public API of `pkg/jira` except the targeted addition of context-accepting variants if needed for cancellation.

## Future work (explicitly deferred)

- HTTP/SSE transport via a `--http :PORT` flag on `serve`.
- Additional tools: edit, assign, link/unlink, list sprints/epics, list projects/boards, get current user, delete (with extra gating), worklog.
- MCP resources (e.g. "current sprint") and prompts (e.g. "summarize my open issues").
- A `--read-only` startup flag and per-tool `dry_run` if real-world use surfaces a need.
