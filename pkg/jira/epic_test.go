//nolint:dupl
package jira

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEpicIssues(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/epic/TEST-0/issue", r.URL.Path)

		qs := r.URL.Query()

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"jql":        []string{"project=TEST AND status=Done ORDER BY created DESC"},
				"maxResults": []string{"100"},
			}, qs)

			resp, err := ioutil.ReadFile("./testdata/search.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.EpicIssues("TEST-0", "project=TEST AND status=Done ORDER BY created DESC", 100)
	assert.NoError(t, err)

	expected := &SearchResult{
		StartAt:    0,
		MaxResults: 50,
		Total:      3,
		Issues: []*Issue{
			{
				Key: "TEST-1",
				Fields: IssueFields{
					Summary:   "Bug summary",
					Labels:    []string{},
					IssueType: IssueType{Name: "Bug"},
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
			},
			{
				Key: "TEST-2",
				Fields: IssueFields{
					Summary:   "Story summary",
					Labels:    []string{"critical", "feature"},
					IssueType: IssueType{Name: "Story"},
					Priority: struct {
						Name string `json:"name"`
					}{Name: "High"},
					Reporter: struct {
						Name string `json:"displayName"`
					}{Name: "Person B"},
					Watches: struct {
						IsWatching bool `json:"isWatching"`
						WatchCount int  `json:"watchCount"`
					}{IsWatching: true, WatchCount: 12},
					Status: struct {
						Name string `json:"name"`
					}{Name: "In Progress"},
					Created: "2020-12-03T14:05:20.974+0100",
					Updated: "2020-12-03T14:05:20.974+0100",
				},
			},
			{
				Key: "TEST-3",
				Fields: IssueFields{
					Summary: "Task summary",
					Labels:  []string{},
					Resolution: struct {
						Name string `json:"name"`
					}{Name: "Done"},
					IssueType: IssueType{Name: "Task"},
					Assignee: struct {
						Name string `json:"displayName"`
					}{Name: "Person A"},
					Priority: struct {
						Name string `json:"name"`
					}{Name: "Low"},
					Reporter: struct {
						Name string `json:"displayName"`
					}{Name: "Person C"},
					Watches: struct {
						IsWatching bool `json:"isWatching"`
						WatchCount int  `json:"watchCount"`
					}{IsWatching: false, WatchCount: 3},
					Status: struct {
						Name string `json:"name"`
					}{Name: "Done"},
					Created: "2020-12-03T14:05:20.974+0100",
					Updated: "2020-12-03T14:05:20.974+0100",
				},
			},
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.EpicIssues("TEST-0", "project=TEST", 100)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestEpicIssuesAdd(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/epic/TEST-0/issue", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			expectedBody := `{"issues":["TEST-1","TEST-2"]}`
			actualBody := new(strings.Builder)
			_, _ = io.Copy(actualBody, r.Body)

			assert.Equal(t, expectedBody, actualBody.String())

			w.WriteHeader(204)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.EpicIssuesAdd("TEST-0", "TEST-1", "TEST-2")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.EpicIssuesAdd("TEST-0", "TEST-1")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestEpicIssuesRemove(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/epic/none/issue", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			expectedBody := `{"issues":["TEST-1","TEST-2"]}`
			actualBody := new(strings.Builder)
			_, _ = io.Copy(actualBody, r.Body)

			assert.Equal(t, expectedBody, actualBody.String())

			w.WriteHeader(204)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.EpicIssuesRemove("TEST-1", "TEST-2")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.EpicIssuesRemove("TEST-1", "TEST-2")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
