package jira

import (
	"encoding/json"
	"strings"
)

func parseCustomFieldJSONContainer(v string) (any, bool) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil, false
	}

	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return nil, false
	}

	var out any
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil, false
	}

	return out, true
}
