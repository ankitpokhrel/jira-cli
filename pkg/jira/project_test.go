package jira

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProjects(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/project", r.URL.Path)

		qs := r.URL.Query()

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"expand": []string{"lead"},
			}, qs)

			resp, err := ioutil.ReadFile("./testdata/projects.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.Project()
	assert.NoError(t, err)

	expected := []*Project{
		{
			Key:  "PRJ1",
			Name: "Project 1",
			Lead: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
		},
		{
			Key:  "PRJ2",
			Name: "Project 2",
			Lead: struct {
				Name string `json:"displayName"`
			}{Name: "Person B"},
		},
		{
			Key:  "PRJ3",
			Name: "Project 3",
			Lead: struct {
				Name string `json:"displayName"`
			}{Name: "Person C"},
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.Project()
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
