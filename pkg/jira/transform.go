// Code generated with assistance from Claude (Anthropic AI)
// https://github.com/ankitpokhrel/jira-cli/pull/909

package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TransformResult contains the transformed JSON and any warnings.
type TransformResult struct {
	Data     []byte
	Warnings []string
}

// TransformIssueFields transforms Jira custom field IDs to human-readable names.
// It takes raw JSON, field mappings from config, and optional field filter.
// Returns transformed JSON with customfield_xxxxx replaced by their configured names in camelCase.
// If collisions are detected, both fields are skipped and a warning is returned.
func TransformIssueFields(rawJSON []byte, fieldMappings []IssueTypeField, fieldFilter []string) (*TransformResult, error) {
	var warnings []string

	if len(fieldMappings) == 0 {
		// No mappings available, return original JSON formatted
		var data interface{}
		if err := json.Unmarshal(rawJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Apply field filter if provided
		if len(fieldFilter) > 0 {
			data = filterFields(data, fieldFilter)
		}

		formatted, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format JSON: %w", err)
		}

		return &TransformResult{Data: formatted, Warnings: warnings}, nil
	}

	var data interface{}
	if err := json.Unmarshal(rawJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Build reverse mapping: customfield_xxx -> human-readable-name
	// Also detect duplicate target names to prevent data loss
	fieldMap := make(map[string]string)
	nameToKeys := make(map[string][]string) // Track which keys map to each name
	skipFields := make(map[string]bool)     // Fields to skip due to collisions

	for _, field := range fieldMappings {
		// Convert "Story Points" -> "storyPoints" (camelCase)
		humanName := ToFieldName(field.Name)
		fieldMap[field.Key] = humanName
		nameToKeys[humanName] = append(nameToKeys[humanName], field.Key)
	}

	// Build a set of explicitly requested customfield IDs from the filter
	// These should NOT be skipped or transformed even if there's a collision
	explicitlyRequested := make(map[string]bool)
	if len(fieldFilter) > 0 {
		for _, path := range fieldFilter {
			parts := strings.Split(path, ".")
			for _, part := range parts {
				if strings.HasPrefix(part, "customfield_") {
					explicitlyRequested[part] = true
				}
			}
		}
	}

	// Detect collisions and handle them
	for humanName, keys := range nameToKeys {
		if len(keys) > 1 {
			// For colliding fields:
			// - If explicitly requested by ID in filter: keep it with raw ID (don't transform)
			// - Otherwise: skip it to avoid ambiguity
			for _, key := range keys {
				if explicitlyRequested[key] {
					// Remove from transformation map so it keeps its raw customfield ID
					delete(fieldMap, key)
				} else {
					// Skip this field entirely
					skipFields[key] = true
				}
			}

			// Format keys with full path for easy copy-paste
			pathKeys := make([]string, len(keys))
			for i, key := range keys {
				pathKeys[i] = "fields." + key
			}

			warnings = append(warnings, fmt.Sprintf(
				"Skipping fields with naming collision '%s': %v. Use --json-filter to explicitly select one, e.g.: --json-filter \"key,%s\"",
				humanName, pathKeys, pathKeys[0],
			))
		}
	}

	// Transform the fields recursively, skipping collisions
	transformed := transformFields(data, fieldMap, skipFields)

	// Apply field filter if provided
	if len(fieldFilter) > 0 {
		// Expand filter paths: if user specifies "fields.customfield_XXX" and there's a mapping,
		// also add "fields.humanName" so filtering works with either reference
		expandedFilter := make([]string, 0, len(fieldFilter)*2)
		for _, path := range fieldFilter {
			expandedFilter = append(expandedFilter, path)

			// Check if this path references a customfield that has a mapping
			parts := strings.Split(path, ".")
			for i, part := range parts {
				if humanName, ok := fieldMap[part]; ok {
					// Build the alternate path with the human name
					altParts := make([]string, len(parts))
					copy(altParts, parts)
					altParts[i] = humanName
					altPath := strings.Join(altParts, ".")
					expandedFilter = append(expandedFilter, altPath)
					break // Only replace the first customfield in the path
				}
			}
		}

		transformed = filterFields(transformed, expandedFilter)
	}

	formatted, err := json.MarshalIndent(transformed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format JSON: %w", err)
	}

	return &TransformResult{Data: formatted, Warnings: warnings}, nil
}

// transformFields recursively transforms custom field keys in the JSON structure.
// It skips any fields marked in skipFields.
func transformFields(data interface{}, fieldMap map[string]string, skipFields map[string]bool) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Skip fields that have naming collisions
			if skipFields[key] {
				continue
			}

			// Check if this key should be transformed
			if newKey, ok := fieldMap[key]; ok {
				result[newKey] = transformFields(value, fieldMap, skipFields)
			} else {
				result[key] = transformFields(value, fieldMap, skipFields)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = transformFields(item, fieldMap, skipFields)
		}
		return result
	default:
		return v
	}
}

// filterFields filters JSON data to include only specified field paths.
// Supports dot notation like "key", "fields.summary", "fields.status.name".
func filterFields(data interface{}, fieldPaths []string) interface{} {
	if len(fieldPaths) == 0 {
		return data
	}

	// Parse field paths into a tree structure for efficient filtering
	pathTree := buildPathTree(fieldPaths)

	return filterByPathTree(data, pathTree, "")
}

// pathTree represents a tree of field paths for filtering.
type pathTree struct {
	includeAll bool                 // If true, include all fields at this level
	children   map[string]*pathTree // Child paths
}

// buildPathTree constructs a tree from field paths like ["key", "fields.summary", "fields.status.name"].
func buildPathTree(paths []string) *pathTree {
	root := &pathTree{children: make(map[string]*pathTree)}

	for _, path := range paths {
		parts := strings.Split(path, ".")
		current := root

		for i, part := range parts {
			if current.children == nil {
				current.children = make(map[string]*pathTree)
			}

			if _, exists := current.children[part]; !exists {
				current.children[part] = &pathTree{children: make(map[string]*pathTree)}
			}

			current = current.children[part]

			// If this is the last part, mark as a leaf (include all below)
			if i == len(parts)-1 {
				current.includeAll = true
			}
		}
	}

	return root
}

// filterByPathTree recursively filters data based on the path tree.
func filterByPathTree(data interface{}, tree *pathTree, currentPath string) interface{} {
	// If tree is nil or includeAll is true, return everything at this level
	if tree == nil || tree.includeAll {
		return data
	}

	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})

		for key, value := range v {
			if childTree, exists := tree.children[key]; exists {
				// This field is in our filter list
				// If this is a leaf node (includeAll or no children), always include it even if null
				if childTree.includeAll || len(childTree.children) == 0 {
					result[key] = value
				} else {
					filtered := filterByPathTree(value, childTree, key)
					if filtered != nil {
						result[key] = filtered
					}
				}
			}
		}

		if len(result) > 0 {
			return result
		}
		return nil

	case []interface{}:
		// For arrays, apply the same filter to each element
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			filtered := filterByPathTree(item, tree, currentPath)
			if filtered != nil {
				result = append(result, filtered)
			}
		}
		if len(result) > 0 {
			return result
		}
		return nil

	default:
		// Primitive value - return as is
		return v
	}
}

// ToFieldName converts a field name to camelCase for use in JSON output.
// For example: "Story Points" -> "storyPoints", "Rank" -> "rank"
func ToFieldName(name string) string {
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
