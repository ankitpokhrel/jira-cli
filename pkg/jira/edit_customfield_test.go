package jira

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequestDataForEditCustomFields(t *testing.T) {
	field := func(name, key, datatype, items string) IssueTypeField {
		f := IssueTypeField{Name: name, Key: key}
		f.Schema.DataType = datatype
		f.Schema.Items = items

		return f
	}

	configured := []IssueTypeField{
		field("Involved Services", "customfield_1", "array", "string"),
		field("Platform", "customfield_2", "array", "option"),
		field("Quarter", "customfield_3", "option", ""),
		field("Story Points", "customfield_4", "number", ""),
		field("Note", "customfield_5", "string", ""),
	}

	cases := []struct {
		name     string
		fields   map[string]string
		expected string
	}{
		{
			name:     "array of strings is set with an explicit verb",
			fields:   map[string]string{"involved-services": "group:one,group:two"},
			expected: `{"customfield_1":[{"set":["group:one","group:two"]}]}`,
		},
		{
			name:     "array of options is added and removed",
			fields:   map[string]string{"platform": "iOS,-Android"},
			expected: `{"customfield_2":[{"add":{"value":"iOS"}},{"remove":{"value":"Android"}}]}`,
		},
		{
			name:     "option is set",
			fields:   map[string]string{"quarter": "Q3"},
			expected: `{"customfield_3":[{"set":{"value":"Q3"}}]}`,
		},
		{
			name:     "number is set",
			fields:   map[string]string{"story-points": "3"},
			expected: `{"customfield_4":[{"set":3}]}`,
		},
		{
			name:     "string is set",
			fields:   map[string]string{"note": "a custom note"},
			expected: `{"customfield_5":[{"set":"a custom note"}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &EditRequest{CustomFields: tc.fields}
			req.WithCustomFields(configured)

			body, err := json.Marshal(getRequestDataForEdit(req))
			assert.NoError(t, err)

			var got struct {
				Update json.RawMessage `json:"update"`
			}
			assert.NoError(t, json.Unmarshal(body, &got))
			assert.JSONEq(t, tc.expected, string(got.Update))
		})
	}
}
