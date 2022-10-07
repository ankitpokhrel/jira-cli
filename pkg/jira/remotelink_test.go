package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebLinkIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1/remotelink/", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(201)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.WebLinkIssue("TEST-1", "weblink title", "http://weblink.com")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.WebLinkIssue("TEST-2", "weblink title", "https://weblink.com")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
