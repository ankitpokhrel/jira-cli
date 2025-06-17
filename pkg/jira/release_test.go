package jira

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReleases(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project/1000/versions", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {

			resp, err := os.ReadFile("./testdata/releases.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.Release("1000")
	assert.NoError(t, err)

	expected := []*ProjectVersion{
		{
			ID:          "1000",
			Description: "Release A",
			Name:        "First",
			Archived:    false,
			Released:    true,
			ProjectID:   1000,
		},
		{
			ID:          "1001",
			Description: "Release B",
			Name:        "Second",
			Archived:    false,
			Released:    false,
			ProjectID:   1000,
		},
		{
			ID:          "1002",
			Description: "Release C",
			Name:        "Third",
			Archived:    true,
			Released:    false,
			ProjectID:   1000,
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.Release("1000")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
