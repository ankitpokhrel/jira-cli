//nolint:dupl
package jira

import (
	"context"
	"errors"
	"net"
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

type mockRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f mockRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestCFAccessRoundTripper provides a **very** narrow unit test to ensure the
// CF Access Header token value is set by the custom round-tripper.
func TestCFAccessRoundTripper(t *testing.T) {
	token := "tmp-access-token"
	wantResponse := &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	next := mockRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, token, req.Header.Get(cfAccessHeader))
		return wantResponse, nil
	})

	req := httptest.NewRequest(http.MethodGet,
		"https://jira.example.com", nil)
	req.Header.Set(cfAccessHeader, "stale-token")

	resp, err := NewCFAccessRoundTripper(token, next).RoundTrip(req)

	assert.NoError(t, err)
	assert.Same(t, wantResponse, resp)

	_ = resp.Body.Close()
}

// isTimeout reports whether err is a network timeout.
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func TestResponseTimeout(t *testing.T) {
	// The server accepts the request and then never answers.
	stalled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-stalled
	}))
	defer server.Close()
	defer close(stalled)

	client := NewClient(
		Config{Server: server.URL},
		WithTimeout(3*time.Second),
		WithResponseTimeout(200*time.Millisecond),
	)

	start := time.Now()
	resp, err := client.Get(context.Background(), "/myself", nil)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Error(t, err)
	assert.True(t, isTimeout(err), "expected a timeout error, got: %v", err)
	assert.Less(t, time.Since(start), 3*time.Second)
}

func TestTLSHandshakeTimeout(t *testing.T) {
	// The listener accepts TCP connections and never starts a TLS handshake.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	var conns []net.Conn
	accepted := make(chan struct{})
	go func() {
		defer close(accepted)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conns = append(conns, conn)
		}
	}()
	defer func() {
		_ = listener.Close()
		<-accepted
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()

	client := NewClient(
		Config{Server: "https://" + listener.Addr().String()},
		WithTimeout(200*time.Millisecond),
		WithInsecureTLS(true),
	)

	start := time.Now()
	resp, err := client.Get(context.Background(), "/myself", nil)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Error(t, err)
	assert.True(t, isTimeout(err), "expected a timeout error, got: %v", err)
	assert.Less(t, time.Since(start), 3*time.Second)
}
