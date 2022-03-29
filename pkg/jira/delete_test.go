package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeleteIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if unexpectedStatusCode {
			assert.Equal(t, "/rest/api/2/issue/BAD", r.URL.RequestURI())
			w.WriteHeader(400)
		} else {
			assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.RequestURI())
			w.WriteHeader(204)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.DeleteIssue("TEST-1", false)
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.DeleteIssue("BAD", false)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestDeleteIssueWithCascade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1?deleteSubtasks=true", r.URL.RequestURI())
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.DeleteIssue("TEST-1", true)
	assert.NoError(t, err)
}
