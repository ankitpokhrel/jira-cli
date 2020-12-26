package jira

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

func TestGetIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := ioutil.ReadFile("./testdata/issue.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3))

	actual, err := client.GetIssue("TEST-1")
	assert.NoError(t, err)

	expected := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary: "Bug summary",
			Description: &adf.ADF{
				Version: 1,
				DocType: "doc",
				Content: []*adf.Node{
					{
						NodeType: "paragraph",
						Content: []*adf.Node{
							{NodeType: "text", NodeValue: adf.NodeValue{Text: "Test description"}},
						},
					},
				},
			},
			Labels: []string{},
			IssueType: struct {
				Name string `json:"name"`
			}{Name: "Bug"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: true, WatchCount: 1},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Created: "2020-12-03T14:05:20.974+0100",
			Updated: "2020-12-03T14:05:20.974+0100",
		},
	}

	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetIssue("TEST-1")
	assert.Error(t, ErrUnexpectedStatusCode, err)
}

func TestGetIssueWithoutDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

		resp, err := ioutil.ReadFile("./testdata/issue-1.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3))

	actual, err := client.GetIssue("TEST-1")
	assert.NoError(t, err)

	var nilADF *adf.ADF
	expected := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary:     "Bug summary",
			Description: nilADF,
			Labels:      []string{},
			IssueType: struct {
				Name string `json:"name"`
			}{Name: "Bug"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: true, WatchCount: 1},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Created: "2020-12-03T14:05:20.974+0100",
			Updated: "2020-12-03T14:05:20.974+0100",
		},
	}
	assert.Equal(t, expected, actual)
}
