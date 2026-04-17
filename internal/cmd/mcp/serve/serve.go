package serve

import (
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
