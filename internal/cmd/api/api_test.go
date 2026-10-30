package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestAPIRequests(t *testing.T) {
	payload := "{\"fields\": {\"summary\": \"hello\"}}\n"
	payloadFile := filepath.Join(t.TempDir(), "payload.json")
	require.NoError(t, os.WriteFile(payloadFile, []byte(payload), 0o600))
	emptyFile := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(emptyFile, nil, 0o600))

	for _, tt := range []struct {
		name        string
		args        []string
		stdin       string
		wantMethod  string
		wantBody    string
		wantHeaders map[string]string
	}{
		{
			name:        "GET",
			wantMethod:  http.MethodGet,
			wantHeaders: map[string]string{"Content-Type": ""},
		},
		{
			name:        "GET with input stays GET",
			args:        []string{"--input", payloadFile},
			wantMethod:  http.MethodGet,
			wantBody:    payload,
			wantHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:        "POST from stdin",
			args:        []string{"-X", "POST", "--input", "-"},
			stdin:       payload,
			wantMethod:  http.MethodPost,
			wantBody:    payload,
			wantHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:        "PUT from file",
			args:        []string{"--method", "PUT", "--input", payloadFile},
			wantMethod:  http.MethodPut,
			wantBody:    payload,
			wantHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:       "PATCH with non-JSON body and custom headers",
			args:       []string{"-X", "patch", "--input", "-", "-H", "content-type: text/plain", "-H", "ACCEPT: text/plain", "-H", "X-Trace: first", "-H", "x-trace: second, third", "-H", "X-URL: https://example.test:8080", "-H", "Authorization: Custom token"},
			stdin:      "plain\x00text\n",
			wantMethod: http.MethodPatch,
			wantBody:   "plain\x00text\n",
			wantHeaders: map[string]string{
				"Content-Type": "text/plain", "Accept": "text/plain", "X-Trace": "second, third",
				"X-URL": "https://example.test:8080", "Authorization": "Custom token",
			},
		},
		{
			name:       "DELETE",
			args:       []string{"-X", "DELETE"},
			wantMethod: http.MethodDelete,
		},
		{
			name:        "empty input file",
			args:        []string{"-X", "POST", "--input", emptyFile},
			wantMethod:  http.MethodPost,
			wantHeaders: map[string]string{"Content-Type": "application/json"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantMethod, r.Method)
				assert.Equal(t, "/rest/api/3/issue", r.URL.Path)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantBody, string(body))
				for key, value := range tt.wantHeaders {
					assert.Equal(t, value, r.Header.Get(key), "header %s", key)
				}
				if _, overridden := tt.wantHeaders["Accept"]; !overridden {
					assert.Equal(t, "application/json", r.Header.Get("Accept"))
				}
				if _, overridden := tt.wantHeaders["Authorization"]; !overridden {
					login, token, ok := r.BasicAuth()
					assert.True(t, ok)
					assert.Equal(t, "user", login)
					assert.Equal(t, "token", token)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			cmd, out := testAPICommand(server.URL)
			cmd.SetArgs(append([]string{"/rest/api/3/issue"}, tt.args...))
			cmd.SetIn(strings.NewReader(tt.stdin))
			require.NoError(t, cmd.Execute())
			assert.Empty(t, out.String())
		})
	}
}

func TestAPIResponses(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "JSON", status: http.StatusOK, body: "{\"key\":\"PROJ-1\"}"},
		{name: "text", status: http.StatusOK, body: "hello\n\n"},
		{name: "binary", status: http.StatusOK, body: "\x00\xff\x80"},
		{name: "created", status: http.StatusCreated, body: "{\"id\":\"1\"}\n"},
		{name: "no content", status: http.StatusNoContent},
		{name: "bad request", status: http.StatusBadRequest, body: "{\"errors\":{\"summary\":\"required\"}}"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: "unauthorized\n"},
		{name: "forbidden", status: http.StatusForbidden, body: "<html>forbidden</html>"},
		{name: "server error", status: http.StatusInternalServerError, body: "server error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					_, _ = io.WriteString(w, tt.body)
				}
			}))
			defer server.Close()

			cmd, out := testAPICommand(server.URL)
			cmd.SetArgs([]string{"/rest/api/3/project"})
			err := cmd.Execute()
			if tt.status >= http.StatusBadRequest {
				require.ErrorContains(t, err, http.StatusText(tt.status))
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.body, out.String())
		})
	}
}

func TestAPIInvalidInput(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	for _, tt := range []struct {
		name  string
		args  []string
		stdin io.Reader
		want  string
	}{
		{name: "missing endpoint", want: "accepts 1 arg"},
		{name: "extra endpoint", args: []string{"/rest/api/3/project", "extra"}, want: "accepts 1 arg"},
		{name: "empty endpoint", args: []string{""}, want: "relative to the configured Jira server"},
		{name: "absolute URL", args: []string{"https://other.example/rest/api/3/project"}, want: "relative to the configured Jira server"},
		{name: "missing header colon", args: []string{"/rest/api/3/project", "-H", "Accept"}, want: "invalid header"},
		{name: "empty header name", args: []string{"/rest/api/3/project", "-H", ": value"}, want: "invalid header"},
		{name: "invalid header name", args: []string{"/rest/api/3/project", "-H", "Bad Header: value"}, want: "invalid header field name"},
		{name: "invalid header value", args: []string{"/rest/api/3/project", "-H", "X-Value: one\ntwo"}, want: "invalid header field value"},
		{name: "invalid method", args: []string{"/rest/api/3/project", "-X", "BAD METHOD"}, want: "invalid method"},
		{name: "empty method", args: []string{"/rest/api/3/project", "-X", ""}, want: "method must not be empty"},
		{name: "missing file", args: []string{"/rest/api/3/project", "--input", filepath.Join(t.TempDir(), "missing")}, want: "read request body"},
		{name: "empty file name", args: []string{"/rest/api/3/project", "--input", ""}, want: "read request body"},
		{name: "stdin read error", args: []string{"/rest/api/3/project", "--input", "-"}, stdin: failingIO{}, want: "read request body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, out := testAPICommand(server.URL)
			cmd.SetArgs(tt.args)
			if tt.stdin != nil {
				cmd.SetIn(tt.stdin)
			}
			err := cmd.Execute()
			require.ErrorContains(t, err, tt.want)
			assert.Empty(t, out.String())
		})
	}
	assert.Zero(t, calls.Load())
}

func TestAPICommandContext(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd, out := testAPICommand(server.URL)
	cmd.SetArgs([]string{"/rest/api/3/project"})
	err := cmd.ExecuteContext(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
	assert.Zero(t, calls.Load())
}

func TestAPIResponseReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = io.WriteString(w, "short")
	}))
	defer server.Close()

	cmd, out := testAPICommand(server.URL)
	cmd.SetArgs([]string{"/rest/api/3/project"})
	err := cmd.Execute()
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, "short", out.String())
}

func TestAPIResponseWriteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "response")
	}))
	defer server.Close()

	cmd, _ := testAPICommand(server.URL)
	cmd.SetOut(failingIO{})
	cmd.SetArgs([]string{"/rest/api/3/project"})
	require.ErrorContains(t, cmd.Execute(), "copy response body")
}

func testAPICommand(server string) (*cobra.Command, *bytes.Buffer) {
	client := jira.NewClient(jira.Config{Server: server, Login: "user", APIToken: "token"})
	cmd := newCmdAPI(func(_ bool) *jira.Client { return client })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	return cmd, &out
}

type failingIO struct{}

func (failingIO) Read(_ []byte) (int, error) {
	return 0, errors.New("read failed")
}

func (failingIO) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}
