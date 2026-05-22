//nolint:dupl
package jira

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// captureStdout runs fn while os.Stdout is redirected to a pipe and returns
// what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		_, _ = io.Copy(&sb, r)
		done <- sb.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func TestDumpScrubsBearerAuthorization(t *testing.T) {
	const secret = "SECRET-TOKEN-XYZ"

	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/rest/api/3/issue", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Cookie", "session=COOKIESECRET")
	req.Header.Set("Proxy-Authorization", "Basic UFJPWFlTRUNSRVQ=")

	out := captureStdout(t, func() { dump(req, nil) })

	assert.NotContains(t, out, secret, "Bearer token must not appear in dump")
	assert.NotContains(t, out, "Bearer "+secret)
	assert.NotContains(t, out, "COOKIESECRET")
	assert.NotContains(t, out, "PROXYSECRET")
	assert.Contains(t, out, "Authorization: REDACTED")
	assert.Contains(t, out, "Cookie: REDACTED")
	assert.Contains(t, out, "Proxy-Authorization: REDACTED")

	// And — equally important — the original request must still carry its
	// real headers so downstream Do() works.
	assert.Equal(t, "Bearer "+secret, req.Header.Get("Authorization"))
	assert.Equal(t, "session=COOKIESECRET", req.Header.Get("Cookie"))
}

func TestDumpScrubsBasicAuthorization(t *testing.T) {
	const user, pass = "alice", "SECRET-PASS-123"

	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/rest/api/3/issue", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.SetBasicAuth(user, pass)

	encoded := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))

	out := captureStdout(t, func() { dump(req, nil) })

	assert.NotContains(t, out, pass, "basic-auth password must not appear in dump")
	assert.NotContains(t, out, encoded, "base64 of user:pass must not appear in dump")
	assert.NotContains(t, out, "Basic "+encoded)
	assert.Contains(t, out, "Authorization: REDACTED")

	// Original request preserved.
	gotUser, gotPass, ok := req.BasicAuth()
	assert.True(t, ok)
	assert.Equal(t, user, gotUser)
	assert.Equal(t, pass, gotPass)
}

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

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second), WithInsecureTLS(true))
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

func TestDeleteV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "jira-cli", r.Header.Get("X-Requested-By"))

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	resp, err := client.DeleteV2(context.Background(), "/issue/TEST-1", Header{
		"X-Requested-By": "jira-cli",
	})

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	_ = resp.Body.Close()
}
