package jira

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCreateMeta(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/createmeta", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"projectKeys":    []string{"TEST"},
				"issuetypeNames": []string{"Epic"},
				"expand":         []string{"projects.issuetypes.fields"},
			}, r.URL.Query())

			resp, err := os.ReadFile("./testdata/createmeta.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetCreateMeta(&CreateMetaRequest{
		Projects:       "TEST",
		IssueTypeNames: "Epic",
		Expand:         "projects.issuetypes.fields",
	})
	assert.NoError(t, err)

	expected := &CreateMetaResponse{[]struct {
		Key        string                 `json:"key"`
		Name       string                 `json:"name"`
		IssueTypes []*CreateMetaIssueType `json:"issuetypes"`
	}{
		{
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []*CreateMetaIssueType{
				{
					IssueType: IssueType{
						ID:      "10001",
						Name:    "Epic",
						Subtask: false,
					},
					Fields: map[string]IssueTypeField{
						"customfield_10011": {
							Name: "Epic Name",
							Key:  "customfield_10011",
						},
						"priority": {
							Name: "Priority",
							Key:  "priority",
						},
						"customfield_10014": {
							Name: "Epic Link",
							Key:  "customfield_10014",
						},
					},
				},
			},
		},
	}}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetCreateMeta(&CreateMetaRequest{
		Projects:       "TEST",
		IssueTypeNames: "Epic",
		Expand:         "projects.issuetypes.fields",
	})
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetCreateMetaForJiraServerV9(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/createmeta/TEST/issuetypes", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"issuetypeNames": []string{"Epic"},
				"expand":         []string{"projects.issuetypes.fields"},
			}, r.URL.Query())

			resp, err := os.ReadFile("./testdata/createmetav9.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetCreateMetaForJiraServerV9(&CreateMetaRequest{
		Projects:       "TEST",
		IssueTypeNames: "Epic",
		Expand:         "projects.issuetypes.fields",
	})
	assert.NoError(t, err)

	expected := &CreateMetaResponseJiraServerV9{[]struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Subtask bool   `json:"subtask"`
	}{
		{
			ID:      "10001",
			Name:    "Epic",
			Subtask: false,
		},
		{
			ID:      "10002",
			Name:    "Task",
			Subtask: false,
		},
		{
			ID:      "10003",
			Name:    "Sub-task",
			Subtask: true,
		},
	}}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetCreateMetaForJiraServerV9(&CreateMetaRequest{
		Projects: "TEST",
		Expand:   "projects.issuetypes.fields",
	})
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
