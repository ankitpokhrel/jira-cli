package jira

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestPaths(t *testing.T) {
	for _, tt := range []struct {
		name       string
		serverPath string
		endpoint   string
		wantURI    string
	}{
		{
			name:     "v3",
			endpoint: "/rest/api/3/project",
			wantURI:  "/rest/api/3/project",
		},
		{
			name:     "v2 without leading slash",
			endpoint: "rest/api/2/myself",
			wantURI:  "/rest/api/2/myself",
		},
		{
			name:       "agile with installation prefix",
			serverPath: "/jira/",
			endpoint:   "/rest/agile/1.0/board?maxResults=10",
			wantURI:    "/jira/rest/agile/1.0/board?maxResults=10",
		},
		{
			name:       "encoded path and query",
			serverPath: "/jira%20server/",
			endpoint:   "/rest/api/3/issue/PROJ%2F123?jql=project%20%3D%20TEST&value=a%2Bb&value=c%26d",
			wantURI:    "/jira%20server/rest/api/3/issue/PROJ%2F123?jql=project%20%3D%20TEST&value=a%2Bb&value=c%26d",
		},
		{
			name:       "encoded installation prefix",
			serverPath: "/jira%2Fteam/",
			endpoint:   "/rest/api/3/project/",
			wantURI:    "/jira%2Fteam/rest/api/3/project/",
		},
		{
			name:     "empty query",
			endpoint: "/rest/api/3/project?",
			wantURI:  "/rest/api/3/project?",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantURI, r.RequestURI)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient(Config{Server: server.URL + tt.serverPath})
			resp, err := client.Request(context.Background(), http.MethodGet, tt.endpoint, nil, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestRequestAuthorization(t *testing.T) {
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:token"))
	for _, tt := range []struct {
		name    string
		auth    AuthType
		token   string
		headers Header
		want    string
	}{
		{name: "default basic", token: "token", want: basic},
		{name: "basic", auth: AuthTypeBasic, token: "token", want: basic},
		{name: "bearer", auth: AuthTypeBearer, token: "token", want: "Bearer token"},
		{name: "mtls with token", auth: AuthTypeMTLS, token: "token", want: "Bearer token"},
		{name: "mtls without token", auth: AuthTypeMTLS},
		{name: "override basic", auth: AuthTypeBasic, token: "token", headers: Header{"Authorization": "Custom value"}, want: "Custom value"},
		{name: "override bearer", auth: AuthTypeBearer, token: "token", headers: Header{"authorization": "Custom value"}, want: "Custom value"},
		{name: "override mtls", auth: AuthTypeMTLS, token: "token", headers: Header{"Authorization": "Custom value"}, want: "Custom value"},
		{name: "empty override", auth: AuthTypeBearer, token: "token", headers: Header{"Authorization": ""}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.want, r.Header.Get("Authorization"))
				assert.LessOrEqual(t, len(r.Header.Values("Authorization")), 1)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client := NewClient(Config{Server: server.URL, Login: "user", APIToken: tt.token})
			// Exercise mTLS authorization independently of certificate loading.
			client.authType = &tt.auth
			resp, err := client.Request(context.Background(), http.MethodGet, "/rest/api/2/myself", nil, tt.headers)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
		})
	}
}

func TestRequestUsesConfiguredTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "access-token", r.Header.Get(cfAccessHeader))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL})
	client.transport = NewCFAccessRoundTripper("access-token", client.transport)
	resp, err := client.Request(context.Background(), http.MethodGet, "/rest/api/2/myself", nil, nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
}

func TestRequestRejectsInvalidURLs(t *testing.T) {
	for _, tt := range []struct {
		name     string
		server   string
		endpoint string
	}{
		{name: "absolute URL", server: "https://jira.example", endpoint: "https://other.example/rest/api/3/project"},
		{name: "protocol relative URL", server: "https://jira.example", endpoint: "//other.example/rest/api/3/project"},
		{name: "opaque URL", server: "https://jira.example", endpoint: "https:other.example"},
		{name: "empty endpoint", server: "https://jira.example"},
		{name: "query without path", server: "https://jira.example", endpoint: "?query=value"},
		{name: "fragment", server: "https://jira.example", endpoint: "/rest/api/3/project#fragment"},
		{name: "invalid encoding", server: "https://jira.example", endpoint: "/rest/api/3/%zz"},
		{name: "missing server", endpoint: "/rest/api/3/project"},
		{name: "invalid server encoding", server: "https://jira%zz.example", endpoint: "/rest/api/3/project"},
		{name: "missing scheme", server: "jira.example", endpoint: "/rest/api/3/project"},
		{name: "missing host", server: "https://", endpoint: "/rest/api/3/project"},
		{name: "unsupported scheme", server: "ftp://jira.example", endpoint: "/rest/api/3/project"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(Config{Server: tt.server})
			client.transport = mockRoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
				t.Error("invalid URL reached the transport")
				return nil, errors.New("unexpected request")
			})
			resp, err := client.Request(context.Background(), http.MethodGet, tt.endpoint, nil, nil)
			if resp != nil {
				defer func() { _ = resp.Body.Close() }()
			}
			assert.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}

func TestRequestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL})
	resp, err := client.Request(ctx, http.MethodGet, "/rest/api/3/project", nil, nil)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, resp)
}

func TestRequestNetworkError(t *testing.T) {
	wantErr := errors.New("connection failed")
	client := NewClient(Config{Server: "https://jira.example"})
	client.transport = mockRoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, wantErr
	})
	resp, err := client.Request(context.Background(), http.MethodGet, "/rest/api/3/project", nil, nil)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, resp)
}

func TestRequestDebugOutput(t *testing.T) {
	stdout, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
	require.NoError(t, err)
	defer func() { _ = stdout.Close() }()
	stderr, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	require.NoError(t, err)
	defer func() { _ = stderr.Close() }()
	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdout, stderr
	defer func() { os.Stdout, os.Stderr = originalOut, originalErr }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Response", "debug-response")
		_, _ = io.WriteString(w, "response body")
	}))
	defer server.Close()
	client := NewClient(Config{Server: server.URL, Debug: true})
	resp, err := client.Request(context.Background(), http.MethodGet, "/rest/api/3/project", nil, nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "response body", string(body))

	out, err := os.ReadFile(stdout.Name())
	require.NoError(t, err)
	assert.Empty(t, out)
	diagnostics, err := os.ReadFile(stderr.Name())
	require.NoError(t, err)
	assert.Contains(t, string(diagnostics), "REQUEST DETAILS")
	assert.Contains(t, string(diagnostics), "RESPONSE DETAILS")
	assert.Contains(t, string(diagnostics), "X-Response: debug-response")
}
