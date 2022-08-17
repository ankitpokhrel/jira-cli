package jira

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServerInfo(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/serverInfo", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/serverinfo.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.ServerInfo()
	assert.NoError(t, err)

	expected := &ServerInfo{
		Version:        "1001.0.0-SNAPSHOT",
		VersionNumbers: []int{1001, 0, 0},
		DeploymentType: "Cloud",
		BuildNumber:    100204,
		DefaultLocale: struct {
			Locale string `json:"locale"`
		}{Locale: "en_US"},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.ServerInfo()
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
