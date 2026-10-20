package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEdit(t *testing.T) {
	var expectedQueryParams string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, expectedQueryParams, r.URL.RawQuery)

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.Edit("TEST-1", &EditRequest{ParentIssueKey: "EPIC-1"})
	assert.NoError(t, err)

	expectedQueryParams = "notifyUsers=false"
	err = client.Edit("TEST-1", &EditRequest{ParentIssueKey: "EPIC-1", SkipNotify: true})
	assert.NoError(t, err)
}
