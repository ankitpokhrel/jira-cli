package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Send authenticated requests to Jira REST endpoints using the configured server.

Specify the full REST path, such as /rest/api/3/project. The path is appended to
the configured server URL, including any installation path prefix.

The method defaults to GET, even when --input is provided. Use --input with a
file path or - to read the request body from standard input. Request bodies are
sent as application/json unless overridden with --header.

Response bodies are written unchanged to standard output. HTTP 4xx and 5xx
responses also print an error to standard error and exit with a non-zero status.`
	examples = `$ jira api /rest/api/2/myself
$ jira api '/rest/api/3/project/search?maxResults=10'
$ jira api -X POST /rest/api/3/issue --input payload.json
$ cat payload.json | jira api -X POST /rest/api/3/issue --input -
$ jira api /rest/api/3/project -H 'Accept: application/json' | jq '.[].key'`
)

type apiOptions struct {
	method  string
	headers []string
	input   string
}

// NewCmdAPI is an api command.
func NewCmdAPI() *cobra.Command {
	return newCmdAPI(api.DefaultClient)
}

func newCmdAPI(client func(bool) *jira.Client) *cobra.Command {
	var opts apiOptions
	cmd := &cobra.Command{
		Use:           "api <endpoint>",
		Short:         "Send authenticated requests to the Jira API",
		Long:          helpText,
		Example:       examples,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		Annotations: map[string]string{
			"cmd:main":  "true",
			"help:args": "<endpoint>\tFull REST path relative to the configured Jira server",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd, args[0], client)
		},
	}
	cmd.Flags().StringVarP(&opts.method, "method", "X", http.MethodGet, "HTTP method (unchanged by --input)")
	cmd.Flags().StringArrayVarP(&opts.headers, "header", "H", nil, "Request header in 'Name: value' format (repeatable; last value wins)")
	cmd.Flags().StringVar(&opts.input, "input", "", "Read the request body from a file (use - for standard input)")
	return cmd
}

func (o apiOptions) run(cmd *cobra.Command, endpoint string, client func(bool) *jira.Client) error {
	if o.method == "" {
		return fmt.Errorf("HTTP method must not be empty")
	}

	hasInput := cmd.Flags().Changed("input")
	headers, err := requestHeaders(o.headers, hasInput)
	if err != nil {
		return err
	}

	var body []byte
	if hasInput {
		if o.input == "-" {
			body, err = io.ReadAll(cmd.InOrStdin())
		} else {
			body, err = os.ReadFile(o.input)
		}
		if err != nil {
			return fmt.Errorf("read request body: %w", err)
		}
	}

	resp, err := client(viper.GetBool("debug")).Request(cmd.Context(), strings.ToUpper(o.method), endpoint, body, headers)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if _, err := io.Copy(cmd.OutOrStdout(), resp.Body); err != nil {
		return fmt.Errorf("copy response body: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("jira API returned %s", resp.Status)
	}
	return nil
}

func requestHeaders(values []string, hasInput bool) (jira.Header, error) {
	headers := jira.Header{"Accept": "application/json"}
	if hasInput {
		headers["Content-Type"] = "application/json"
	}
	for _, header := range values {
		name, value, ok := strings.Cut(header, ":")
		name = strings.Trim(name, " \t")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid header %q: expected 'Name: value'", header)
		}
		headers[http.CanonicalHeaderKey(name)] = strings.Trim(value, " \t")
	}
	return headers, nil
}
