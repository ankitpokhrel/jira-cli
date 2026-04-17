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
