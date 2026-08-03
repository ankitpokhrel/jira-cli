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
	defer func() { _ = session.Close() }()

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

func TestRegisterTool_RecoversFromPanic(t *testing.T) {
	srv := mcp.NewServer(&mcp.Implementation{Name: "panic-test", Version: "v0"}, nil)

	type panicIn struct{}
	type panicOut struct{}

	registerTool(srv, "panic_tool", "always panics",
		&tools.Deps{},
		func(context.Context, *tools.Deps, panicIn) (panicOut, error) {
			panic("boom")
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	serverT, clientT := mcp.NewInMemoryTransports()
	go func() { _ = srv.Run(ctx, serverT) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "panic-client", Version: "v0"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "panic_tool",
		Arguments: map[string]any{},
	})
	require.NoError(t, err, "transport should survive a panicking handler")
	assert.True(t, res.IsError, "panic should surface as a tool error, not a transport error")
	require.NotEmpty(t, res.Content)
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		assert.Contains(t, tc.Text, "panic_tool")
		assert.Contains(t, tc.Text, "boom")
	}
}
