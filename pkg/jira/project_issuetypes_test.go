package jira

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProjectIssueTypes(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/project/TEST", r.URL.Path)

		qs := r.URL.Query()
		if unexpectedStatusCode {
			w.WriteHeader(401)
			return
		}

		assert.Equal(t, url.Values{
			"expand": []string{"issueTypes"},
		}, qs)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"issueTypes": [
				{"id": "1", "name": "Task", "subtask": false},
				{"id": "2", "name": "Sub-task", "subtask": true}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.ProjectIssueTypes("TEST")
	assert.NoError(t, err)
	assert.Equal(t, []*IssueType{
		{ID: "1", Name: "Task", Subtask: false},
		{ID: "2", Name: "Sub-task", Subtask: true},
	}, actual)

	unexpectedStatusCode = true
	_, err = client.ProjectIssueTypes("TEST")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

