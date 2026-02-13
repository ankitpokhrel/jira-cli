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
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.JSONEq(t, expectedBody, actualBody.String())

		w.WriteHeader(e.code)
	}))
}

func TestEditWithCascadingSelect(t *testing.T) {
	expectedBody := `{"update":{"customfield_10318":[{"set":{"value":"Engineering","child":{"value":"Backend"}}}]},"fields":{"parent":{}}}`
	server := (&editTestServer{code: http.StatusNoContent}).serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	requestData := EditRequest{
		CustomFields: map[string]string{"dev-team": "Engineering->Backend"},
	}
	requestData.WithCustomFields([]IssueTypeField{
		{Name: "Dev Team", Key: "customfield_10318", Schema: struct {
			DataType string `json:"type"`
			Items    string `json:"items,omitempty"`
		}{DataType: "option-with-child"}},
	})

	err := client.Edit("TEST-1", &requestData)
	assert.NoError(t, err)
}

func TestEditWithCascadingSelectParentOnly(t *testing.T) {
	expectedBody := `{"update":{"customfield_10318":[{"set":{"value":"Engineering"}}]},"fields":{"parent":{}}}`
	server := (&editTestServer{code: http.StatusNoContent}).serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	requestData := EditRequest{
		CustomFields: map[string]string{"dev-team": "Engineering"},
	}
	requestData.WithCustomFields([]IssueTypeField{
		{Name: "Dev Team", Key: "customfield_10318", Schema: struct {
			DataType string `json:"type"`
			Items    string `json:"items,omitempty"`
		}{DataType: "option-with-child"}},
	})

	err := client.Edit("TEST-1", &requestData)
	assert.NoError(t, err)
}
