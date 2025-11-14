// Code generated with assistance from Claude (Anthropic AI)
// https://github.com/ankitpokhrel/jira-cli/pull/909

package jira

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformIssueFields(t *testing.T) {
	t.Run("transforms custom field IDs to readable names", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"summary": "Test Issue",
				"customfield_10001": "My Epic Name",
				"customfield_10002": 5
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Epic Name", Key: "customfield_10001"},
			{Name: "Story Points", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		// Verify transformation
		resultStr := string(result)
		assert.Contains(t, resultStr, `"epicName"`)
		assert.Contains(t, resultStr, `"storyPoints"`)
		assert.NotContains(t, resultStr, `"customfield_10001"`)
		assert.NotContains(t, resultStr, `"customfield_10002"`)

		// Verify standard fields unchanged
		assert.Contains(t, resultStr, `"summary"`)
		assert.Contains(t, resultStr, `"Test Issue"`)
	})

	t.Run("handles nested custom fields", func(t *testing.T) {
		rawJSON := []byte(`{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"customfield_10001": "Value1"
					}
				},
				{
					"key": "TEST-2",
					"fields": {
						"customfield_10001": "Value2"
					}
				}
			]
		}`)

		mappings := []IssueTypeField{
			{Name: "Epic Link", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"epicLink"`)
		assert.NotContains(t, resultStr, `"customfield_10001"`)
	})

	t.Run("handles empty mappings gracefully", func(t *testing.T) {
		rawJSON := []byte(`{"key": "TEST-1", "fields": {"customfield_10001": "value"}}`)
		mappings := []IssueTypeField{}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		// Should return formatted JSON unchanged
		assert.Contains(t, string(result), `"customfield_10001"`)
	})

	t.Run("handles malformed JSON", func(t *testing.T) {
		rawJSON := []byte(`{invalid json}`)
		mappings := []IssueTypeField{
			{Name: "Epic Name", Key: "customfield_10001"},
		}

		_, err := TransformIssueFields(rawJSON, mappings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})

	t.Run("preserves complex nested structures", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": {
					"nested": {
						"value": "complex"
					}
				},
				"issuelinks": [
					{
						"id": "10001",
						"type": {
							"name": "Blocks"
						}
					}
				]
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Sprint", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		// Verify structure preserved
		var resultData map[string]interface{}
		err = json.Unmarshal(result, &resultData)
		assert.NoError(t, err)

		fields := resultData["fields"].(map[string]interface{})
		assert.Contains(t, fields, "sprint")
		assert.Contains(t, fields, "issuelinks")

		// Verify nested structure intact
		sprint := fields["sprint"].(map[string]interface{})
		nested := sprint["nested"].(map[string]interface{})
		assert.Equal(t, "complex", nested["value"])
	})

	t.Run("handles null custom field values", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"summary": "Test",
				"customfield_10001": null,
				"customfield_10002": {
					"value": null
				}
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Epic Link", Key: "customfield_10001"},
			{Name: "T-Shirt Size", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"epicLink": null`)
		assert.Contains(t, resultStr, `"tShirtSize"`)
	})

	t.Run("handles custom fields with object values", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": {
					"value": "Large",
					"id": "123"
				}
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "T-Shirt Size", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		// Verify object structure preserved
		var resultData map[string]interface{}
		err = json.Unmarshal(result, &resultData)
		assert.NoError(t, err)

		fields := resultData["fields"].(map[string]interface{})
		tShirtSize := fields["tShirtSize"].(map[string]interface{})
		assert.Equal(t, "Large", tShirtSize["value"])
		assert.Equal(t, "123", tShirtSize["id"])
	})

	t.Run("handles custom fields with array values", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": [
					{"name": "Sprint 1", "id": 1},
					{"name": "Sprint 2", "id": 2}
				],
				"customfield_10002": ["tag1", "tag2"]
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Sprint", Key: "customfield_10001"},
			{Name: "Tags", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"sprint"`)
		assert.Contains(t, resultStr, `"tags"`)
		assert.Contains(t, resultStr, `"Sprint 1"`)
		assert.Contains(t, resultStr, `"tag1"`)
	})

	t.Run("does not transform standard fields", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"summary": "Test Issue",
				"status": {"name": "Done"},
				"customfield_10001": "Epic Name"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Epic Name", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		// Standard fields should remain unchanged
		assert.Contains(t, resultStr, `"summary"`)
		assert.Contains(t, resultStr, `"status"`)
		// Custom field should be transformed
		assert.Contains(t, resultStr, `"epicName"`)
		assert.NotContains(t, resultStr, `"customfield_10001"`)
	})

	t.Run("handles name collisions gracefully", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"summary": "Test",
				"customfield_10001": "Custom Value"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Summary", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		// Should have both summary (standard) and summary from custom field
		// Last one wins in our current implementation
		resultStr := string(result)
		assert.Contains(t, resultStr, `"summary"`)
	})

	t.Run("handles deeply nested custom fields", func(t *testing.T) {
		rawJSON := []byte(`{
			"level1": {
				"level2": {
					"level3": {
						"customfield_10001": "deep value"
					}
				}
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Deep Field", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"deepField"`)
		assert.NotContains(t, resultStr, `"customfield_10001"`)
	})

	t.Run("handles single empty field name gracefully", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": "value1",
				"summary": "test"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Empty name converts to empty string key, which is valid JSON
	})

	t.Run("returns error for empty and whitespace field names that collide", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": "value1",
				"customfield_10002": "value2"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "", Key: "customfield_10001"},
			{Name: "   ", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "multiple custom fields map to the same name")
	})

	t.Run("returns error for multiple custom fields mapping to same name", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": "Value 1",
				"customfield_10002": "Value 2"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Story Points", Key: "customfield_10001"},
			{Name: "Story Points", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "multiple custom fields map to the same name")
		assert.Contains(t, err.Error(), "storyPoints")
	})

	t.Run("handles empty arrays in custom fields", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": [],
				"customfield_10002": {}
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Labels", Key: "customfield_10001"},
			{Name: "Metadata", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"labels": []`)
		assert.Contains(t, resultStr, `"metadata": {}`)
	})

	t.Run("handles numeric string values", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": "0.0",
				"customfield_10002": "9223372036854775807",
				"customfield_10003": "3.14159"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Business Value", Key: "customfield_10001"},
			{Name: "Rank", Key: "customfield_10002"},
			{Name: "Confidence", Key: "customfield_10003"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"businessValue": "0.0"`)
		assert.Contains(t, resultStr, `"rank": "9223372036854775807"`)
		assert.Contains(t, resultStr, `"confidence": "3.14159"`)
	})

	t.Run("handles empty string values", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"summary": "Test",
				"customfield_10001": ""
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Notes", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"notes": ""`)
	})

	t.Run("handles Jira option objects pattern", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": {
					"self": "https://jira.example.com/rest/api/2/customFieldOption/1001",
					"value": "No",
					"id": "1001",
					"disabled": false
				},
				"customfield_10002": [
					{
						"self": "https://jira.example.com/rest/api/2/customFieldOption/2001",
						"value": "High priority item",
						"id": "2001",
						"disabled": false
					}
				]
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Ready", Key: "customfield_10001"},
			{Name: "Priority Flags", Key: "customfield_10002"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		// Verify the structure is preserved
		var resultData map[string]interface{}
		err = json.Unmarshal(result, &resultData)
		assert.NoError(t, err)

		fields := resultData["fields"].(map[string]interface{})

		// Check single option object
		ready := fields["ready"].(map[string]interface{})
		assert.Equal(t, "No", ready["value"])
		assert.Equal(t, "1001", ready["id"])

		// Check array of option objects
		priorityFlags := fields["priorityFlags"].([]interface{})
		assert.Len(t, priorityFlags, 1)
		firstFlag := priorityFlags[0].(map[string]interface{})
		assert.Equal(t, "High priority item", firstFlag["value"])
	})

	t.Run("handles very long string values", func(t *testing.T) {
		longString := strings.Repeat("Lorem ipsum dolor sit amet ", 1000)
		rawJSON := []byte(fmt.Sprintf(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": "%s"
			}
		}`, longString))

		mappings := []IssueTypeField{
			{Name: "Description", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify it contains our field name
		resultStr := string(result)
		assert.Contains(t, resultStr, `"description"`)
	})

	t.Run("handles JSON with many custom fields", func(t *testing.T) {
		// Simulate a realistic Jira response with many custom fields
		fieldsJSON := `"key": "TEST-1", "fields": {`
		for i := 1; i <= 100; i++ {
			if i > 1 {
				fieldsJSON += ","
			}
			fieldsJSON += fmt.Sprintf(`"customfield_%d": "value%d"`, 10000+i, i)
		}
		fieldsJSON += `}`
		rawJSON := []byte("{" + fieldsJSON + "}")

		// Create mappings for all these fields
		mappings := make([]IssueTypeField, 100)
		for i := 0; i < 100; i++ {
			mappings[i] = IssueTypeField{
				Name: fmt.Sprintf("Custom Field %d", i+1),
				Key:  fmt.Sprintf("customfield_%d", 10001+i),
			}
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify some transformations occurred
		resultStr := string(result)
		assert.Contains(t, resultStr, `"customField1"`)
		assert.Contains(t, resultStr, `"customField50"`)
		assert.Contains(t, resultStr, `"customField100"`)
		// Original IDs should be gone
		assert.NotContains(t, resultStr, `"customfield_10001"`)
	})

	t.Run("handles extremely deep nesting without stack overflow", func(t *testing.T) {
		// Create deeply nested structure (reasonable depth)
		depth := 50
		var buildNested func(int) string
		buildNested = func(level int) string {
			if level == 0 {
				return `"customfield_10001": "deep value"`
			}
			return fmt.Sprintf(`"level%d": {%s}`, level, buildNested(level-1))
		}

		rawJSON := []byte(fmt.Sprintf(`{%s}`, buildNested(depth)))

		mappings := []IssueTypeField{
			{Name: "Deep Field", Key: "customfield_10001"},
		}

		// Should not panic or stack overflow
		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultStr := string(result)
		assert.Contains(t, resultStr, `"deepField"`)
	})

	t.Run("handles large array with many items", func(t *testing.T) {
		// Simulate array with many items (like a large sprint with many issues)
		items := make([]string, 200)
		for i := 0; i < 200; i++ {
			items[i] = fmt.Sprintf(`{"id": %d, "name": "Item %d"}`, i, i)
		}
		arrayJSON := strings.Join(items, ",")

		rawJSON := []byte(fmt.Sprintf(`{
			"key": "TEST-1",
			"fields": {
				"customfield_10001": [%s]
			}
		}`, arrayJSON))

		mappings := []IssueTypeField{
			{Name: "Items", Key: "customfield_10001"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify transformation happened
		resultStr := string(result)
		assert.Contains(t, resultStr, `"items"`)
		assert.Contains(t, resultStr, `"Item 0"`)
		assert.Contains(t, resultStr, `"Item 199"`)
	})

	t.Run("handles mixed custom fields", func(t *testing.T) {
		rawJSON := []byte(`{
			"key": "PROJ-1234",
			"fields": {
				"customfield_10001": "0.0",
				"fixVersions": [],
				"resolution": {
					"name": "Won't Do"
				},
				"customfield_10002": [
					{
						"displayName": "John Doe",
						"emailAddress": "jdoe@example.com"
					}
				],
				"customfield_10003": null,
				"priority": {
					"name": "Minor"
				},
				"labels": ["backend", "api"],
				"customfield_10004": {
					"value": "No",
					"id": "1001"
				},
				"customfield_10005": {
					"value": "No"
				},
				"customfield_10006": "None",
				"status": {
					"name": "Closed"
				},
				"customfield_10007": "9223372036854775807",
				"summary": "Test issue"
			}
		}`)

		mappings := []IssueTypeField{
			{Name: "Business Value", Key: "customfield_10001"},
			{Name: "Contributors", Key: "customfield_10002"},
			{Name: "Sprint", Key: "customfield_10003"},
			{Name: "Ready", Key: "customfield_10004"},
			{Name: "Blocked", Key: "customfield_10005"},
			{Name: "Blocked Reason", Key: "customfield_10006"},
			{Name: "Rank", Key: "customfield_10007"},
		}

		result, err := TransformIssueFields(rawJSON, mappings)
		assert.NoError(t, err)

		resultStr := string(result)

		// Verify custom fields are transformed
		assert.Contains(t, resultStr, `"businessValue"`)
		assert.Contains(t, resultStr, `"contributors"`)
		assert.Contains(t, resultStr, `"sprint"`)
		assert.Contains(t, resultStr, `"ready"`)
		assert.Contains(t, resultStr, `"blocked"`)
		assert.Contains(t, resultStr, `"blockedReason"`)
		assert.Contains(t, resultStr, `"rank"`)

		// Verify standard fields are NOT transformed
		assert.Contains(t, resultStr, `"fixVersions"`)
		assert.Contains(t, resultStr, `"resolution"`)
		assert.Contains(t, resultStr, `"priority"`)
		assert.Contains(t, resultStr, `"labels"`)
		assert.Contains(t, resultStr, `"status"`)
		assert.Contains(t, resultStr, `"summary"`)

		// Verify custom field IDs are removed
		assert.NotContains(t, resultStr, `"customfield_10001"`)
		assert.NotContains(t, resultStr, `"customfield_10002"`)
		assert.NotContains(t, resultStr, `"customfield_10003"`)
	})
}

func TestToFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Story Points", "storyPoints"},
		{"Epic Name", "epicName"},
		{"T-Shirt Size", "tShirtSize"},
		{"single", "single"},
		{"UPPERCASE", "uppercase"},
		{"Mixed Case Words", "mixedCaseWords"},
		{"With-Dashes", "withDashes"},
		{"With.Dots", "withDots"},
		{"With_Underscores", "withUnderscores"},
		{"Multiple   Spaces", "multipleSpaces"},
		{"Special!@#Chars", "specialChars"},
		{"Unicode Field Ñame", "unicodeFieldAme"},         // Non-ASCII chars stripped
		{"123 Starts With Number", "123StartsWithNumber"}, // Numbers preserved
		{"", ""},
		{"   ", ""},
		{"!!!", ""},
		{"Ação Completa", "aOCompleta"}, // Non-ASCII chars stripped
		{"Über Field", "berField"},      // Umlaut stripped
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toFieldName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformFieldsRecursion(t *testing.T) {
	fieldMap := map[string]string{
		"customfield_10001": "epicLink",
		"customfield_10002": "storyPoints",
	}

	t.Run("transforms array of objects", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"customfield_10001": "EPIC-1",
			},
			map[string]interface{}{
				"customfield_10002": 5,
			},
		}

		result := transformFields(data, fieldMap)
		resultArray := result.([]interface{})

		obj1 := resultArray[0].(map[string]interface{})
		assert.Contains(t, obj1, "epicLink")
		assert.NotContains(t, obj1, "customfield_10001")

		obj2 := resultArray[1].(map[string]interface{})
		assert.Contains(t, obj2, "storyPoints")
		assert.NotContains(t, obj2, "customfield_10002")
	})

	t.Run("preserves primitive values", func(t *testing.T) {
		tests := []interface{}{
			"string",
			123,
			123.456,
			true,
			false,
			nil,
		}

		for _, test := range tests {
			result := transformFields(test, fieldMap)
			assert.Equal(t, test, result)
		}
	})
}
