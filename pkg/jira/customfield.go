package jira

import "strings"

const (
	customFieldFormatOption  = "option"
	customFieldFormatArray   = "array"
	customFieldFormatNumber  = "number"
	customFieldFormatProject = "project"
)

type customField map[string]interface{}

type customFieldTypeNumber float64

type customFieldTypeNumberSet struct {
	Set customFieldTypeNumber `json:"set"`
}

type customFieldTypeStringSet struct {
	Set string `json:"set"`
}

type customFieldTypeOption struct {
	Value string `json:"value"`
}

type customFieldTypeOptionSet struct {
	Set customFieldTypeOption `json:"set"`
}

type customFieldTypeOptionAddRemove struct {
	Add    *customFieldTypeOption `json:"add,omitempty"`
	Remove *customFieldTypeOption `json:"remove,omitempty"`
}

type customFieldTypeProject struct {
	Value string `json:"key"`
}

type customFieldTypeProjectSet struct {
	Set customFieldTypeProject `json:"set"`
}

// splitUnescapedCommas splits a string on commas that are not escaped with backslash.
// Escaped commas (\,) are unescaped in the resulting strings.
func splitUnescapedCommas(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}

	var result []string
	var current strings.Builder
	escaped := false

	for i := 0; i < len(s); i++ {
		if escaped {
			if s[i] == ',' {
				current.WriteByte(',')
			} else {
				current.WriteByte('\\')
				current.WriteByte(s[i])
			}
			escaped = false
		} else if s[i] == '\\' {
			escaped = true
		} else if s[i] == ',' {
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(s[i])
		}
	}

	if escaped {
		current.WriteByte('\\')
	}

	result = append(result, strings.TrimSpace(current.String()))
	return result
}
