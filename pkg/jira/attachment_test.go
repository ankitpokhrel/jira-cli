package jira

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetIssueAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "attachment", r.URL.Query().Get("fields"))

		resp, err := os.ReadFile("./testdata/attachments.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	attachments, err := client.GetIssueAttachments("TEST-1")
	assert.NoError(t, err)
	assert.Len(t, attachments, 2)
	assert.Equal(t, "screenshot.png", attachments[0].Filename)
	assert.Equal(t, "document.pdf", attachments[1].Filename)
	assert.Equal(t, 12345, attachments[0].Size)
}

func TestGetIssueAttachmentsV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "attachment", r.URL.Query().Get("fields"))

		resp, err := os.ReadFile("./testdata/attachments.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	attachments, err := client.GetIssueAttachmentsV2("TEST-1")
	assert.NoError(t, err)
	assert.Len(t, attachments, 2)
}

func TestGetIssueAttachments_NoAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"key": "TEST-1", "fields": {"attachment": []}}`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	attachments, err := client.GetIssueAttachments("TEST-1")
	assert.NoError(t, err)
	assert.Len(t, attachments, 0)
}
