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
