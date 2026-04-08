package jira

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testProjectComponentsHelper(t *testing.T, expectedPath string, fn func(*Client, string) ([]*ProjectComponent, error)) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedPath, r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/components.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := fn(client, "PRJ")
	assert.NoError(t, err)

	expected := []*ProjectComponent{
		{ID: "10000", Name: "Backend", Description: "Backend component"},
		{ID: "10001", Name: "Frontend", Description: "Frontend component"},
		{ID: "10002", Name: "Mobile", Description: "Mobile component"},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = fn(client, "PRJ")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestProjectComponents(t *testing.T) {
	testProjectComponentsHelper(t, "/rest/api/3/project/PRJ/components", func(c *Client, p string) ([]*ProjectComponent, error) {
		return c.ProjectComponents(p)
	})
}

func TestProjectComponentsV2(t *testing.T) {
	testProjectComponentsHelper(t, "/rest/api/2/project/PRJ/components", func(c *Client, p string) ([]*ProjectComponent, error) {
		return c.ProjectComponentsV2(p)
	})
}
