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

type editTestServer struct{ code int }

func (e *editTestServer) serve(t *testing.T, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-123", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.JSONEq(t, expectedBody, actualBody.String())

		w.WriteHeader(e.code)
	}))
}

func TestEditWithCustomFieldArrayEscapedComma(t *testing.T) {
	expectedBody := `{"update":{"customfield_10050":["WL: Tools, Development and Support"]},"fields":{"parent":{}}}`
	testServer := editTestServer{code: 204}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	customFields := []IssueTypeField{
		{
			Name: "Work Category",
			Key:  "customfield_10050",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{
				DataType: "array",
				Items:    "string",
			},
		},
	}

	requestData := EditRequest{
		CustomFields: map[string]string{
			"work-category": `WL: Tools\, Development and Support`,
		},
	}
	requestData.WithCustomFields(customFields)

	err := client.Edit("TEST-123", &requestData)
	assert.NoError(t, err)
}

func TestEditWithCustomFieldArrayMultipleValues(t *testing.T) {
	expectedBody := `{"update":{"customfield_10051":["Value 1","Value 2, with comma","Value 3"]},"fields":{"parent":{}}}`
	testServer := editTestServer{code: 204}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	customFields := []IssueTypeField{
		{
			Name: "Multi Value Field",
			Key:  "customfield_10051",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{
				DataType: "array",
				Items:    "string",
			},
		},
	}

	requestData := EditRequest{
		CustomFields: map[string]string{
			"multi-value-field": `Value 1,Value 2\, with comma,Value 3`,
		},
	}
	requestData.WithCustomFields(customFields)

	err := client.Edit("TEST-123", &requestData)
	assert.NoError(t, err)
}
