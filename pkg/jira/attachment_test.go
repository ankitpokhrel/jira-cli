package jira

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUploadAttachment(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notes.txt")
	assert.NoError(t, os.WriteFile(filePath, []byte("hello jira"), 0o600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1/attachments", r.URL.Path)
		assert.Equal(t, "no-check", r.Header.Get("X-Atlassian-Token"))

		mr, err := r.MultipartReader()
		assert.NoError(t, err)
		part, err := mr.NextPart()
		assert.NoError(t, err)
		assert.Equal(t, "notes.txt", part.FileName())
		body, err := io.ReadAll(part)
		assert.NoError(t, err)
		assert.Equal(t, "hello jira", string(body))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[{"id":"10000","filename":"notes.txt","size":10,"mimeType":"text/plain","content":"http://x/attachment/10000","author":{"displayName":"Ada","emailAddress":"a@x.io","active":true},"created":"2024-01-01T00:00:00.000+0000"}]`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	out, err := client.UploadAttachment("TEST-1", []string{filePath})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "10000", out[0].ID)
	assert.Equal(t, "notes.txt", out[0].Filename)
}

func TestUploadAttachmentMissingFile(t *testing.T) {
	client := NewClient(Config{Server: "http://localhost"}, WithTimeout(3*time.Second))

	_, err := client.UploadAttachment("TEST-1", []string{"/nonexistent/file.txt"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot open file")
}

func TestUploadAttachmentNoFiles(t *testing.T) {
	client := NewClient(Config{Server: "http://localhost"}, WithTimeout(3*time.Second))

	_, err := client.UploadAttachment("TEST-1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files provided")
}

func TestUploadAttachmentServerError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notes.txt")
	assert.NoError(t, os.WriteFile(filePath, []byte("x"), 0o600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"errorMessages":["You do not have permission."]}`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	_, err := client.UploadAttachment("TEST-1", []string{filePath})
	assert.Error(t, err)
}

func TestListAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "attachment", r.URL.Query().Get("fields"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"key":"TEST-1","fields":{"attachment":[{"id":"10000","filename":"a.txt","size":1,"mimeType":"text/plain","content":"http://x/1","author":{"displayName":"Ada","emailAddress":"a@x.io","active":true},"created":"2024-01-01T00:00:00.000+0000"}]}}`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	out, err := client.ListAttachments("TEST-1")
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "a.txt", out[0].Filename)
}

func TestListAttachmentsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"key":"TEST-1","fields":{}}`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	out, err := client.ListAttachments("TEST-1")
	assert.NoError(t, err)
	assert.Empty(t, out)
}

func TestDownloadAttachment(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/2/attachment/10000":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"id":"10000","filename":"a.txt","size":3,"mimeType":"text/plain","content":"%s/content/10000","author":{"displayName":"Ada","emailAddress":"a@x.io","active":true},"created":"2024-01-01T00:00:00.000+0000"}`,
				server.URL,
			)))
		case "/content/10000":
			assert.Equal(t, "application/octet-stream", r.Header.Get("Accept"))
			w.WriteHeader(200)
			_, _ = w.Write([]byte("abc"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	data, name, err := client.DownloadAttachment("10000")
	assert.NoError(t, err)
	assert.Equal(t, "a.txt", name)
	assert.Equal(t, "abc", string(data))
}

func TestDownloadAttachmentMissingURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"10000","filename":"a.txt","size":0,"mimeType":"text/plain","author":{"displayName":"Ada","emailAddress":"a@x.io","active":true},"created":"2024-01-01T00:00:00.000+0000"}`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	_, _, err := client.DownloadAttachment("10000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no download URL")
}

func TestDeleteAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/attachment/10000", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	assert.NoError(t, client.DeleteAttachment("10000"))
}

func TestMultipartFieldName(t *testing.T) {
	// Guards the Jira-required multipart field name "file".
	dir := t.TempDir()
	filePath := filepath.Join(dir, "a.txt")
	assert.NoError(t, os.WriteFile(filePath, []byte("x"), 0o600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseMultipartForm(1<<20))
		files := r.MultipartForm.File["file"]
		assert.Len(t, files, 1)
		fh := files[0]
		assert.Equal(t, "a.txt", fh.Filename)
		f, err := fh.Open()
		assert.NoError(t, err)
		defer func() { _ = f.Close() }()
		_, _ = io.ReadAll(f)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	_, err := client.UploadAttachment("TEST-1", []string{filePath})
	assert.NoError(t, err)
}
