//nolint:dupl
package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSprints(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/board/2/sprint", r.URL.Path)

		qs := r.URL.Query()

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"state":      []string{"active,closed"},
				"startAt":    []string{"0"},
				"maxResults": []string{"10"},
			}, qs)

			resp, err := os.ReadFile("./testdata/sprints.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.Sprints(2, "state=active,closed", 0, 10)
	assert.NoError(t, err)

	expected := &SprintResult{
		MaxResults: 10,
		StartAt:    0,
		IsLast:     true,
		Sprints: []*Sprint{
			{
				ID:           1,
				Name:         "Sprint 1",
				Status:       "closed",
				StartDate:    "2020-11-15T05:39:24.463Z",
				EndDate:      "2020-11-29T05:39:24.463Z",
				CompleteDate: "2020-11-29T04:19:24.463Z",
				BoardID:      2,
			},
			{
				ID:           2,
				Name:         "Sprint 2",
				Status:       "closed",
				StartDate:    "2020-11-15T05:39:24.463Z",
				EndDate:      "2020-11-29T05:39:24.463Z",
				CompleteDate: "2020-11-29T04:19:24.463Z",
				BoardID:      2,
			},
			{
				ID:           3,
				Name:         "Sprint 3",
				Status:       "closed",
				StartDate:    "2020-11-15T05:39:24.463Z",
				EndDate:      "2020-11-29T05:39:24.463Z",
				CompleteDate: "2020-11-29T04:19:24.463Z",
				BoardID:      2,
			},
			{
				ID:           4,
				Name:         "Sprint 4",
				Status:       "closed",
				StartDate:    "2020-11-15T05:39:24.463Z",
				EndDate:      "2020-11-29T05:39:24.463Z",
				CompleteDate: "2020-11-29T04:19:24.463Z",
				BoardID:      2,
			},
			{
				ID:        5,
				Name:      "Sprint 5",
				Status:    "active",
				StartDate: "2020-11-29T06:49:24.463Z",
				EndDate:   "2020-12-13T07:09:24.463Z",
			},
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.Sprints(2, "state=active,closed", 0, 10)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestSprintsInBoards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/board/2/sprint", r.URL.Path)

		var (
			resp []byte
			err  error
		)

		qs := r.URL.Query()

		switch qs.Get("startAt") {
		case "0":
			assert.Equal(t, url.Values{
				"state":      []string{"active,closed"},
				"startAt":    []string{"0"},
				"maxResults": []string{"3"},
			}, qs)

			resp, err = os.ReadFile("./testdata/sprints-0.json")
			assert.NoError(t, err)

		case "2":
			assert.Equal(t, url.Values{
				"state":      []string{"active,closed"},
				"startAt":    []string{"2"},
				"maxResults": []string{"3"},
			}, qs)

			resp, err = os.ReadFile("./testdata/sprints-2.json")
			assert.NoError(t, err)

		case "3":
			assert.Equal(t, url.Values{
				"state":      []string{"active,closed"},
				"startAt":    []string{"3"},
				"maxResults": []string{"3"},
			}, qs)

			resp, err = os.ReadFile("./testdata/sprints-3.json")
			assert.NoError(t, err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	actual := client.SprintsInBoards([]int{2}, "state=active,closed", 3)
	expected := []*Sprint{
		{
			ID:        5,
			Name:      "Sprint 5",
			Status:    "active",
			StartDate: "2020-11-29T06:49:24.463Z",
			EndDate:   "2020-12-13T07:09:24.463Z",
			BoardID:   2,
		},
		{
			ID:           4,
			Name:         "Sprint 4",
			Status:       "closed",
			StartDate:    "2020-11-15T05:39:24.463Z",
			EndDate:      "2020-11-29T05:39:24.463Z",
			CompleteDate: "2020-11-29T04:19:24.463Z",
			BoardID:      2,
		},
		{
			ID:           3,
			Name:         "Sprint 3",
			Status:       "closed",
			StartDate:    "2020-11-15T05:39:24.463Z",
			EndDate:      "2020-11-29T05:39:24.463Z",
			CompleteDate: "2020-11-29T04:19:24.463Z",
			BoardID:      2,
		},
	}
	assert.Equal(t, expected, actual)
}

func TestSprintIssues(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/sprint/2/issue", r.URL.Path)

		qs := r.URL.Query()

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"jql":        []string{"project=TEST AND status=Done ORDER BY created DESC"},
				"startAt":    []string{"0"},
				"maxResults": []string{"100"},
			}, qs)

			resp, err := os.ReadFile("./testdata/search.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SprintIssues(2, "project=TEST AND status=Done ORDER BY created DESC", 0, 100)
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

	_, err = client.SprintIssues(2, "project=TEST", 0, 100)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestSprintIssuesAdd(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/sprint/5/issue", r.URL.Path)

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

	err := client.SprintIssuesAdd("5", "TEST-1", "TEST-2")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.SprintIssuesAdd("5", "TEST-1")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetSprint(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/sprint/5", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, "GET", r.Method)

			resp, err := os.ReadFile("./testdata/sprint-get.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	sprint, err := client.GetSprint(5)
	assert.NoError(t, err)
	assert.Equal(t, sprint.ID, 5)
	assert.Equal(t, sprint.Status, "active")

	unexpectedStatusCode = true

	_, err = client.GetSprint(5)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestEndSprint(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotNilf(t, r.Method, "invalid request method")

		if r.Method == "GET" {
			assert.Equal(t, "/rest/agile/1.0/sprint/5", r.URL.Path)

			resp, err := os.ReadFile("./testdata/sprint-get.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		} else {
			if unexpectedStatusCode {
				w.WriteHeader(400)
			} else if r.Method == "PUT" {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(200)
			}
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.EndSprint(5)
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.EndSprint(5)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
