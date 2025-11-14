// Code generated with assistance from Claude (Anthropic AI)
// https://github.com/ankitpokhrel/jira-cli/pull/909

package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TransformIssueFields transforms Jira custom field IDs to human-readable names.
// It takes raw JSON and field mappings from config, returns transformed JSON with
// customfield_xxxxx replaced by their configured names in camelCase.
func TransformIssueFields(rawJSON []byte, fieldMappings []IssueTypeField) ([]byte, error) {
	if len(fieldMappings) == 0 {
		// No mappings available, return original JSON formatted
		var data interface{}
		if err := json.Unmarshal(rawJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		return json.MarshalIndent(data, "", "  ")
	}

	var data interface{}
	if err := json.Unmarshal(rawJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Build reverse mapping: customfield_xxx -> human-readable-name
	// Also detect duplicate target names to prevent data loss
	fieldMap := make(map[string]string)
	nameToKeys := make(map[string][]string) // Track which keys map to each name

	for _, field := range fieldMappings {
		// Convert "Story Points" -> "storyPoints" (camelCase)
		humanName := toFieldName(field.Name)
		fieldMap[field.Key] = humanName
		nameToKeys[humanName] = append(nameToKeys[humanName], field.Key)
	}

	// Check for duplicate mappings
	for humanName, keys := range nameToKeys {
		if len(keys) > 1 {
			return nil, fmt.Errorf("multiple custom fields map to the same name '%s': %v", humanName, keys)
		}
	}

	// Transform the fields recursively
	transformed := transformFields(data, fieldMap)

	return json.MarshalIndent(transformed, "", "  ")
}

// transformFields recursively transforms custom field keys in the JSON structure.
func transformFields(data interface{}, fieldMap map[string]string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Check if this key should be transformed
			if newKey, ok := fieldMap[key]; ok {
				result[newKey] = transformFields(value, fieldMap)
			} else {
				result[key] = transformFields(value, fieldMap)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = transformFields(item, fieldMap)
		}
		return result
	default:
		return v
	}
}

// toFieldName converts "Story Points" to "storyPoints" (camelCase).
// It handles special characters and multiple words.
func toFieldName(name string) string {
	// Remove special characters and split by space/punctuation
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return ' '
	}, name)

	words := strings.Fields(name)
	if len(words) == 0 {
		return strings.ToLower(strings.ReplaceAll(name, " ", ""))
	}

	// First word lowercase, rest title case
	result := strings.ToLower(words[0])
	for _, word := range words[1:] {
		if len(word) > 0 {
			result += strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return result
}
