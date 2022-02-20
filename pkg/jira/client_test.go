//nolint:dupl
package jira

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search", r.URL.Path)
		assert.Equal(t, url.Values{
			"jql": []string{"project=TEST AND status=Done"},
		}, r.URL.Query())
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.Get(context.Background(), "/search?jql=project=TEST%20AND%20status=Done", Header{
		"Content-Type": "text/plain",
	})

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestGetV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/epic/TEST-1/issue", r.URL.Path)
		assert.Equal(t, url.Values{
			"jql": []string{"project=TEST AND status=Done"},
		}, r.URL.Query())

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.GetV1(context.Background(), "/epic/TEST-1/issue?jql=project=TEST%20AND%20status=Done", nil)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestGetV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/search", r.URL.Path)
		assert.Equal(t, url.Values{
			"jql": []string{"project=TEST AND status=Done"},
		}, r.URL.Query())
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.GetV2(context.Background(), "/search?jql=project=TEST%20AND%20status=Done", Header{
		"Content-Type": "text/plain",
	})

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "jira-cli", r.Header.Get("X-Requested-By"))

		w.WriteHeader(201)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.Post(context.Background(), "/issue", []byte("hello"), Header{
		"Content-Type":   "application/json",
		"X-Requested-By": "jira-cli",
	})

	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestPostV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "jira-cli", r.Header.Get("X-Requested-By"))

		w.WriteHeader(201)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.PostV2(context.Background(), "/issue", []byte("hello"), Header{
		"Content-Type":   "application/json",
		"X-Requested-By": "jira-cli",
	})

	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestPostV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/issue", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "jira-cli", r.Header.Get("X-Requested-By"))

		w.WriteHeader(201)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.PostV1(context.Background(), "/issue", []byte("hello"), Header{
		"Content-Type":   "application/json",
		"X-Requested-By": "jira-cli",
	})

	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1/assignee", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "jira-cli", r.Header.Get("X-Requested-By"))

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.Put(context.Background(), "/issue/TEST-1/assignee", []byte("jon"), Header{
		"Content-Type":   "application/json",
		"X-Requested-By": "jira-cli",
	})

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	_ = resp.Body.Close()
}

func TestPutV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1/assignee", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "jira-cli", r.Header.Get("X-Requested-By"))

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.PutV2(context.Background(), "/issue/TEST-1/assignee", []byte("jon"), Header{
		"Content-Type":   "application/json",
		"X-Requested-By": "jira-cli",
	})

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	_ = resp.Body.Close()
}
