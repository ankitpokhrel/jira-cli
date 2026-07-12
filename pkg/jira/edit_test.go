package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEditCustomFieldArray(t *testing.T) {
	expectedBody := `{"update":{"customfield_10001":[{"set":["a","b","c"]}]},"fields":{"parent":{}}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.JSONEq(t, expectedBody, actualBody.String())

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	req := &EditRequest{
		CustomFields: map[string]string{
			"multi-select": "a,b,c",
		},
	}
	req.WithCustomFields([]IssueTypeField{
		{
			Name: "Multi Select",
			Key:  "customfield_10001",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{DataType: customFieldFormatArray, Items: "string"},
		},
	})

	err := client.Edit("TEST-1", req)
	assert.NoError(t, err)
}

func TestConstructCustomFieldsForEditArrayOfStrings(t *testing.T) {
	data := &editRequest{}

	configuredFields := []IssueTypeField{
		{
			Name: "Multi Select",
			Key:  "customfield_10001",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{DataType: customFieldFormatArray, Items: "string"},
		},
	}
	fields := map[string]string{
		"multi-select": "a,b,c",
	}

	constructCustomFieldsForEdit(fields, configuredFields, data)

	expected := []customFieldTypeArraySet{{Set: []string{"a", "b", "c"}}}
	assert.Equal(t, expected, data.Update.M.customFields["customfield_10001"])
}

func TestEdit(t *testing.T) {
	var gotRawQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		gotRawQuery = r.URL.RawQuery

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.Edit("TEST-1", &EditRequest{ParentIssueKey: "EPIC-1"})
	assert.NoError(t, err)
	assert.Empty(t, gotRawQuery, "notifyUsers should not be set when SkipNotify is not requested")

	err = client.Edit("TEST-1", &EditRequest{ParentIssueKey: "EPIC-1", SkipNotify: true})
	assert.NoError(t, err)
	assert.Equal(t, "notifyUsers=false", gotRawQuery, "notifyUsers=false should only be set when SkipNotify is requested")
}
