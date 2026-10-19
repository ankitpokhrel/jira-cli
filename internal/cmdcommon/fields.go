package cmdcommon

import (
	"strings"
)

// TranslateFieldNames converts human-readable field names to field IDs.
// For example: "Story Points,summary" -> "customfield_10001,summary"
// Leaves unknown names and field IDs unchanged.
// Also normalizes the input by trimming whitespace and removing empty fields.
func TranslateFieldNames(fieldsStr string) string {
	if fieldsStr == "" {
		return ""
	}

	// Get field mappings from config
	fieldMappings, err := GetConfiguredCustomFields()

	// Build name -> ID map (case-insensitive for user convenience)
	nameToID := make(map[string]string)
	if err == nil {
		for _, field := range fieldMappings {
			nameToID[strings.ToLower(field.Name)] = field.Key
		}
	}

	// Process each field in the comma-separated list
	fields := strings.Split(fieldsStr, ",")
	translatedFields := make([]string, 0, len(fields))

	for _, field := range fields {
		field = strings.TrimSpace(field)

		// Skip empty fields (important for cases like "key,,summary")
		if field == "" {
			continue
		}

		// If it's already a customfield ID or special value, keep as-is
		if strings.HasPrefix(field, "customfield_") || strings.HasPrefix(field, "*") {
			translatedFields = append(translatedFields, field)
			continue
		}

		// Try to translate from config (case-insensitive)
		if fieldID, ok := nameToID[strings.ToLower(field)]; ok {
			translatedFields = append(translatedFields, fieldID)
		} else {
			// Unknown field name, keep as-is (might be a standard field like "key", "summary")
			translatedFields = append(translatedFields, field)
		}
	}

	return strings.Join(translatedFields, ",")
}
