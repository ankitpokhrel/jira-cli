//nolint:dupl
package jira

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// captureStdout redirects os.Stdout to a pipe, runs fn, then returns
// whatever was written. This is used to verify debug output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String()
}

func TestDumpRedactsAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		authType AuthType
		login    string
		token    string
	}{
		{
			name:     "bearer token is redacted",
			authType: AuthTypeBearer,
			token:    "super-secret-pat-token",
		},
		{
			name:     "basic auth is redacted",
			authType: AuthTypeBasic,
			login:    "user@example.com",
			token:    "super-secret-api-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(Config{
				Server:   server.URL,
				Login:    tt.login,
				APIToken: tt.token,
				AuthType: &tt.authType,
				Debug:    true,
			}, WithTimeout(3*time.Second))

			output := captureStdout(t, func() {
				resp, err := client.GetV2(context.Background(), "/search", nil)
				require.NoError(t, err)
				_ = resp.Body.Close()
			})

			assert.Contains(t, output, "[REDACTED]",
				"debug output should contain redacted placeholder")
			assert.NotContains(t, output, tt.token,
				"debug output must not contain the raw token")
		})
	}

	// MTLS with a bearer token is tested via dump() directly because
	// NewClient requires real certificate files for MTLS initialization.
	t.Run("mtls bearer token is redacted", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/test", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer super-secret-mtls-token")

		output := captureStdout(t, func() {
			dump(req, nil)
		})

		assert.Contains(t, output, "[REDACTED]",
			"debug output should contain redacted placeholder")
		assert.NotContains(t, output, "super-secret-mtls-token",
			"debug output must not contain the raw MTLS token")
	})
}

func TestDumpPreservesOriginalHeaders(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer server.Close()

	authType := AuthTypeBearer
	client := NewClient(Config{
		Server:   server.URL,
		APIToken: "my-secret-token",
		AuthType: &authType,
		Debug:    true,
	}, WithTimeout(3*time.Second))

	_ = captureStdout(t, func() {
		resp, err := client.GetV2(context.Background(), "/search", nil)
		require.NoError(t, err)
		_ = resp.Body.Close()
	})

	assert.Equal(t, "Bearer my-secret-token", receivedAuth,
		"server must receive the real Authorization header, not the redacted one")
}

func TestDumpWithNoAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	// Create a request without any auth to verify dump does not panic
	// or inject a spurious [REDACTED] header.
	req, err := http.NewRequest(http.MethodGet, server.URL+"/test", http.NoBody)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		dump(req, nil)
	})

	assert.NotContains(t, output, "Authorization",
		"dump should not add an Authorization header when none exists")
	assert.NotContains(t, output, "[REDACTED]",
		"dump should not show redacted placeholder when no auth header is present")
	assert.Contains(t, output, "REQUEST DETAILS",
		"dump should still produce request output")
}

func TestDumpWithMultipleAuthorizationValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/test", http.NoBody)
	require.NoError(t, err)

	// Simulate multiple Authorization header values (unlikely but possible).
	req.Header.Add("Authorization", "Bearer token-one")
	req.Header.Add("Authorization", "Bearer token-two")

	output := captureStdout(t, func() {
		dump(req, nil)
	})

	assert.NotContains(t, output, "token-one",
		"first authorization value must be redacted")
	assert.NotContains(t, output, "token-two",
		"second authorization value must be redacted")
	assert.Contains(t, output, "[REDACTED]",
		"redacted placeholder should appear in output")

	// Verify original request headers are untouched.
	vals := req.Header.Values("Authorization")
	assert.Len(t, vals, 2, "original request should still have both Authorization values")
	assert.Equal(t, "Bearer token-one", vals[0])
	assert.Equal(t, "Bearer token-two", vals[1])
}
