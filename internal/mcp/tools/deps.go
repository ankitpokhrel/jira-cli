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
