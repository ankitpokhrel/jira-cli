package jira

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProjectComponents(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project/PRJ/components", r.URL.Path)

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

	actual, err := client.ProjectComponents("PRJ")
	assert.NoError(t, err)

	expected := []*ProjectComponent{
		{ID: "10000", Name: "Backend", Description: "Backend component"},
		{ID: "10001", Name: "Frontend", Description: "Frontend component"},
		{ID: "10002", Name: "Mobile", Description: "Mobile component"},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.ProjectComponents("PRJ")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestProjectComponentsV2(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/project/PRJ/components", r.URL.Path)

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

	actual, err := client.ProjectComponentsV2("PRJ")
	assert.NoError(t, err)

	expected := []*ProjectComponent{
		{ID: "10000", Name: "Backend", Description: "Backend component"},
		{ID: "10001", Name: "Frontend", Description: "Frontend component"},
		{ID: "10002", Name: "Mobile", Description: "Mobile component"},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.ProjectComponentsV2("PRJ")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
