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

func TestCreateVersion(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/rest/api/2/version", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/version-create.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	req := &VersionCreateRequest{
		Name:        "Version 1.0",
		Description: "First release",
		Project:     "TEST",
		Archived:    false,
		Released:    false,
	}

	actual, err := client.CreateVersion(req)
	assert.NoError(t, err)
	assert.Equal(t, "10000", actual.ID)
	assert.Equal(t, "Version 1.0", actual.Name)
	assert.Equal(t, "First release", actual.Description)

	unexpectedStatusCode = true

	_, err = client.CreateVersion(req)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetVersion(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/rest/api/2/version/10000", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(404)
		} else {
			resp, err := os.ReadFile("./testdata/version-get.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetVersion("10000")
	assert.NoError(t, err)
	assert.Equal(t, "10000", actual.ID)
	assert.Equal(t, "Version 1.0", actual.Name)

	unexpectedStatusCode = true

	_, err = client.GetVersion("10000")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestUpdateVersion(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/rest/api/2/version/10000", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	released := true
	req := &VersionUpdateRequest{
		Name:     "Version 1.0 Updated",
		Released: &released,
	}

	err := client.UpdateVersion("10000", req)
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.UpdateVersion("10000", req)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
