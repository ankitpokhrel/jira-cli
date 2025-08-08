package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	jiraConfig "github.com/ankitpokhrel/jira-cli/internal/config"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Send authenticated requests to arbitrary Jira API endpoints.

You can use this command to make authenticated API requests to any endpoint in the
Jira REST API using the currently configured authentication settings.

The endpoint path (like /rest/api/3/project) will be appended to your configured
Jira server URL.`
	examples = `# Send a GET request to a custom endpoint
$ jira api /rest/api/3/project

# Send a POST request with a JSON payload
$ jira api -X POST /rest/api/3/issue -d '{"fields":{"project":{"key":"DEMO"},"summary":"Test issue","issuetype":{"name":"Task"}}}'

# Use a file as the request body
$ jira api -X POST /rest/api/3/issue --file payload.json

# Translate customfield_* IDs to their friendly names from config
$ jira api /rest/api/3/issue/PROJ-123 --translate-fields`
)

// NewCmdAPI is an api command.
func NewCmdAPI() *cobra.Command {
	cmd := cobra.Command{
		Use:   "api [endpoint]",
		Short: "Make authenticated requests to the Jira API",
		Long:  helpText,
		Example: examples,
		Annotations: map[string]string{
			"cmd:main":  "true",
			"help:args": "[endpoint]\tEndpoint path to send the request to, will be appended to the configured Jira server URL",
		},
		Args: cobra.ExactArgs(1),
		Run:  runAPI,
	}

	cmd.Flags().StringP("method", "X", "GET", "HTTP method to use (GET, POST, PUT, DELETE)")
	cmd.Flags().StringP("data", "d", "", "JSON payload to send with the request")
	cmd.Flags().String("file", "", "File containing JSON payload to send with the request")
	cmd.Flags().Bool("raw", false, "Output raw response body without formatting")
	cmd.Flags().Bool("translate-fields", false, "Translate customfield_* IDs to their friendly names from config")

	return &cmd
}

// translateCustomFields replaces customfield_* IDs with their friendly names from config
func translateCustomFields(data []byte, debug bool) []byte {
	// Get the custom fields from the config
	configuredFields, err := getCustomFieldsMapping()
	if err != nil && debug {
		fmt.Fprintf(os.Stderr, "Warning: Error getting custom field mapping: %s\n", err)
		return data
	}

	if len(configuredFields) == 0 {
		if debug {
			fmt.Fprintf(os.Stderr, "No custom field mappings found in config. No translations will be applied.\n")
		}
		return data
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Found %d custom field mappings in config.\n", len(configuredFields))
	}

	// Try to detect any customfield_* patterns in the response that aren't in our config
	var unrecognizedFields []string
	re := regexp.MustCompile(`"(customfield_\d+)"`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	fieldSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			fieldID := match[1]
			if _, exists := configuredFields[fieldID]; !exists {
				fieldSet[fieldID] = true
			}
		}
	}

	for field := range fieldSet {
		unrecognizedFields = append(unrecognizedFields, field)
	}

	if len(unrecognizedFields) > 0 && debug {
		fmt.Fprintf(os.Stderr, "Found %d custom fields in the response that aren't mapped in the config:\n", len(unrecognizedFields))
		for _, field := range unrecognizedFields {
			fmt.Fprintf(os.Stderr, "  - %s\n", field)
		}
	}

	dataStr := string(data)
	replacements := 0

	// Replace all occurrences of customfield_* with their friendly names
	for id, name := range configuredFields {
		// Replace the field in keys (like "customfield_12345":)
		pattern := fmt.Sprintf("\"%s\":", id)
		replacement := fmt.Sprintf("\"%s\":", name)
		newStr := strings.ReplaceAll(dataStr, pattern, replacement)

		if newStr != dataStr {
			replacements++
		}

		dataStr = newStr
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Translated %d custom field occurrences in the response.\n", replacements)
	}

	return []byte(dataStr)
}

// getCustomFieldsMapping returns a map of custom field IDs to their friendly names
func getCustomFieldsMapping() (map[string]string, error) {
	var configuredFields []jira.IssueTypeField

	err := viper.UnmarshalKey("issue.fields.custom", &configuredFields)
	if err != nil {
		return nil, err
	}

	// Create a map of custom field IDs to their friendly names
	fieldsMap := make(map[string]string)
	for _, field := range configuredFields {
		// Extract the field ID from the key - typically in the format "customfield_XXXXX"
		if field.Key != "" && strings.HasPrefix(field.Key, "customfield_") {
			fieldsMap[field.Key] = field.Name
		}
	}

	return fieldsMap, nil
}

func runAPI(cmd *cobra.Command, args []string) {
	// Check if the environment is initialized properly
	configFile := viper.ConfigFileUsed()
	if configFile == "" || !jiraConfig.Exists(configFile) {
		cmdutil.Failed("Jira CLI is not initialized. Run 'jira init' first.")
	}

	server := viper.GetString("server")
	if server == "" {
		cmdutil.Failed("Jira server URL is not configured. Run 'jira init' with the --server flag.")
	}

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	endpoint := args[0]

	method, err := cmd.Flags().GetString("method")
	cmdutil.ExitIfError(err)

	data, err := cmd.Flags().GetString("data")
	cmdutil.ExitIfError(err)

	file, err := cmd.Flags().GetString("file")
	cmdutil.ExitIfError(err)

	raw, err := cmd.Flags().GetBool("raw")
	cmdutil.ExitIfError(err)

	var payload []byte

	if file != "" && data != "" {
		cmdutil.Failed("Cannot use both --data and --file")
	}

	if file != "" {
		fmt.Printf("Reading payload from file: %s\n", file)
		payload, err = os.ReadFile(file)
		cmdutil.ExitIfError(err)
	} else if data != "" {
		payload = []byte(data)
		if debug {
			fmt.Printf("Request payload: %s\n", data)
		}
	}

	// Show a progress spinner during the request
	s := cmdutil.Info("Sending request to Jira API...")
	defer s.Stop()

	client := api.Client(jira.Config{Debug: debug})

	var resp *http.Response
	ctx := context.Background()
	headers := jira.Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	// Ensure endpoint starts with a slash
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Combine server URL with the endpoint
	targetURL := server + endpoint
	if debug {
		fmt.Printf("Sending %s request to: %s\n", method, targetURL)
	}
	resp, err = client.RequestURL(ctx, method, targetURL, payload, headers)

	s.Stop()

	if err != nil {
		cmdutil.Failed("Request failed: %s", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	cmdutil.ExitIfError(err)

	translateFields, err := cmd.Flags().GetBool("translate-fields")
	cmdutil.ExitIfError(err)

	// Try to pretty print JSON if the response appears to be JSON and raw mode is not enabled
	if !raw && len(body) > 0 {
		// Check if the response looks like JSON
		trimmedBody := strings.TrimSpace(string(body))
		isJSON := (strings.HasPrefix(trimmedBody, "{") && strings.HasSuffix(trimmedBody, "}")) ||
			(strings.HasPrefix(trimmedBody, "[") && strings.HasSuffix(trimmedBody, "]"))

		if isJSON {
			// If we need to translate custom fields, do that before pretty printing
			if translateFields {
				body = translateCustomFields(body, debug)
			}

			var prettyJSON bytes.Buffer
			err = json.Indent(&prettyJSON, body, "", "  ")
			if err == nil {
				body = prettyJSON.Bytes()
			}
		}
	}

	fmt.Printf("HTTP/%d %s\n", resp.StatusCode, resp.Status)

	// Print response headers if debug mode is enabled
	if debug {
		fmt.Println("\nResponse Headers:")
		for k, v := range resp.Header {
			fmt.Printf("%s: %s\n", k, strings.Join(v, ", "))
		}
		fmt.Println()
	}

	fmt.Println(string(body))
}
