package jira

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructCustomFieldsParsesNestedJSONObjectForCreate(t *testing.T) {
	req := CreateRequest{
		Project:   "TEST",
		IssueType: "Task",
		Summary:   "Test issue",
		CustomFields: map[string]string{
			"reviewers": `{"accountId":"abc","profile":{"team":{"id":"42"}}}`,
		},
	}
	req.WithCustomFields([]IssueTypeField{
		{
			Name: "reviewers",
			Key:  "customfield_10042",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{
				DataType: "user",
			},
		},
	})

	body, err := json.Marshal((&Client{}).getRequestData(&req))
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"update": {},
		"fields": {
			"project": {"key": "TEST"},
			"issuetype": {"name": "Task"},
			"summary": "Test issue",
			"customfield_10042": {
				"accountId": "abc",
				"profile": {"team": {"id": "42"}}
			}
		}
	}`, string(body))
}

func TestConstructCustomFieldsParsesNestedJSONObjectForEdit(t *testing.T) {
	req := EditRequest{
		CustomFields: map[string]string{
			"reviewers": `{"accountId":"abc","profile":{"team":{"members":[{"id":"1"},{"id":"2"}]}}}`,
		},
	}
	req.WithCustomFields([]IssueTypeField{
		{
			Name: "reviewers",
			Key:  "customfield_10042",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{
				DataType: "user",
			},
		},
	})

	body, err := json.Marshal(getRequestDataForEdit(&req))
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"update": {
			"customfield_10042": [{
				"set": {
					"accountId": "abc",
					"profile": {"team": {"members": [{"id":"1"},{"id":"2"}]}}
				}
			}]
		},
		"fields": {"parent": {}}
	}`, string(body))
}
