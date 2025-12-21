//nolint:dupl
package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	var (
		apiVersion2          bool
		unexpectedStatusCode bool
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiVersion2 {
			assert.Equal(t, "/rest/api/2/search", r.URL.Path)
		} else {
			assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
		}

		qs := r.URL.Query()

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"jql":        []string{"project=TEST AND status=Done ORDER BY created DESC"},
				"fields":     []string{"*all"},
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

	actual, err := client.Search("project=TEST AND status=Done ORDER BY created DESC", 0, 100)
	assert.NoError(t, err)

	expected := &SearchResult{
		IsLast:        true,
		NextPageToken: "",
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

	apiVersion2 = true
	unexpectedStatusCode = true

	_, err = client.SearchV2("project=TEST", 0, 100)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestSearchPagination(t *testing.T) {
	// Helper to create a minimal issue with just a key
	makeIssue := func(key string) *Issue {
		return &Issue{
			Key: key,
			Fields: IssueFields{
				Summary: key + " summary",
			},
		}
	}

	// Helper to create a search response
	makeResponse := func(issues []*Issue, nextPageToken string, isLast bool) []byte {
		resp := struct {
			Issues        []*Issue `json:"issues"`
			NextPageToken string   `json:"nextPageToken,omitempty"`
			IsLast        bool     `json:"isLast"`
		}{
			Issues:        issues,
			NextPageToken: nextPageToken,
			IsLast:        isLast,
		}
		data, _ := json.Marshal(resp)
		return data
	}

	t.Run("offset skipping within single page", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			// Return 5 issues, client requests from=2, limit=2
			// Should skip first 2 and return TEST-3, TEST-4
			resp := makeResponse([]*Issue{
				makeIssue("TEST-1"),
				makeIssue("TEST-2"),
				makeIssue("TEST-3"),
				makeIssue("TEST-4"),
				makeIssue("TEST-5"),
			}, "", true)
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 2, 2)

		assert.NoError(t, err)
		assert.Len(t, result.Issues, 2)
		assert.Equal(t, "TEST-3", result.Issues[0].Key)
		assert.Equal(t, "TEST-4", result.Issues[1].Key)
	})

	t.Run("multi-page fetching with nextPageToken", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)

			token := r.URL.Query().Get("nextPageToken")

			var resp []byte
			switch token {
			case "":
				// First page
				resp = makeResponse([]*Issue{
					makeIssue("TEST-1"),
					makeIssue("TEST-2"),
				}, "page2token", false)
			case "page2token":
				// Second page
				resp = makeResponse([]*Issue{
					makeIssue("TEST-3"),
					makeIssue("TEST-4"),
				}, "page3token", false)
			case "page3token":
				// Third/last page
				resp = makeResponse([]*Issue{
					makeIssue("TEST-5"),
				}, "", true)
			}
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 0, 100)

		assert.NoError(t, err)
		assert.Equal(t, 3, requestCount, "should make 3 requests to fetch all pages")
		assert.Len(t, result.Issues, 5)
		assert.Equal(t, "TEST-1", result.Issues[0].Key)
		assert.Equal(t, "TEST-5", result.Issues[4].Key)
	})

	t.Run("limit stops fetching early", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)

			// Each page has 3 items, but client only wants 2
			resp := makeResponse([]*Issue{
				makeIssue("TEST-1"),
				makeIssue("TEST-2"),
				makeIssue("TEST-3"),
			}, "nexttoken", false)
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 0, 2)

		assert.NoError(t, err)
		assert.Equal(t, 1, requestCount, "should stop after first page since limit reached")
		assert.Len(t, result.Issues, 2)
		assert.Equal(t, "TEST-1", result.Issues[0].Key)
		assert.Equal(t, "TEST-2", result.Issues[1].Key)
		assert.False(t, result.IsLast, "IsLast should be false when we stop early due to limit")
	})

	t.Run("offset spanning multiple pages", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)

			token := r.URL.Query().Get("nextPageToken")

			var resp []byte
			switch token {
			case "":
				// First page: 2 items
				resp = makeResponse([]*Issue{
					makeIssue("TEST-1"),
					makeIssue("TEST-2"),
				}, "page2token", false)
			case "page2token":
				// Second page: 2 items
				resp = makeResponse([]*Issue{
					makeIssue("TEST-3"),
					makeIssue("TEST-4"),
				}, "page3token", false)
			case "page3token":
				// Third page: 2 items
				resp = makeResponse([]*Issue{
					makeIssue("TEST-5"),
					makeIssue("TEST-6"),
				}, "", true)
			}
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		// Skip first 3 items, get next 2
		// Should skip TEST-1, TEST-2, TEST-3 and return TEST-4, TEST-5
		result, err := client.Search("project=TEST", 3, 2)

		assert.NoError(t, err)
		assert.Len(t, result.Issues, 2)
		assert.Equal(t, "TEST-4", result.Issues[0].Key)
		assert.Equal(t, "TEST-5", result.Issues[1].Key)
	})

	t.Run("offset beyond available items returns empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			resp := makeResponse([]*Issue{
				makeIssue("TEST-1"),
				makeIssue("TEST-2"),
			}, "", true)
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		// Offset 10, but only 2 items exist
		result, err := client.Search("project=TEST", 10, 5)

		assert.NoError(t, err)
		assert.Len(t, result.Issues, 0)
		assert.True(t, result.IsLast)
	})

	t.Run("empty result set", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			resp := makeResponse([]*Issue{}, "", true)
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 0, 10)

		assert.NoError(t, err)
		assert.Len(t, result.Issues, 0)
		assert.True(t, result.IsLast)
	})

	t.Run("error mid-pagination returns error", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")

			if requestCount == 1 {
				// First request succeeds
				w.WriteHeader(200)
				resp := makeResponse([]*Issue{
					makeIssue("TEST-1"),
					makeIssue("TEST-2"),
				}, "page2token", false)
				_, _ = w.Write(resp)
			} else {
				// Second request fails
				w.WriteHeader(500)
				_, _ = w.Write([]byte(`{"errorMessages":["Internal server error"]}`))
			}
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 0, 10)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, 2, requestCount, "should have made 2 requests before error")
	})

	t.Run("nextPageToken is passed in subsequent requests", func(t *testing.T) {
		var receivedTokens []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedTokens = append(receivedTokens, r.URL.Query().Get("nextPageToken"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)

			token := r.URL.Query().Get("nextPageToken")
			var resp []byte
			switch token {
			case "":
				resp = makeResponse([]*Issue{makeIssue("TEST-1")}, "token-page-2", false)
			case "token-page-2":
				resp = makeResponse([]*Issue{makeIssue("TEST-2")}, "token-page-3", false)
			case "token-page-3":
				resp = makeResponse([]*Issue{makeIssue("TEST-3")}, "", true)
			}
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		_, err := client.Search("project=TEST", 0, 10)

		assert.NoError(t, err)
		assert.Equal(t, []string{"", "token-page-2", "token-page-3"}, receivedTokens,
			"should pass correct nextPageToken in each request")
	})

	t.Run("IsLast true when all pages exhausted", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			// Single page with all results, isLast=true
			resp := makeResponse([]*Issue{
				makeIssue("TEST-1"),
				makeIssue("TEST-2"),
			}, "", true)
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 0, 10)

		assert.NoError(t, err)
		assert.Len(t, result.Issues, 2)
		assert.True(t, result.IsLast, "IsLast should be true when all pages exhausted")
	})

	t.Run("IsLast false when limit reached before end", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			// Page has more items than requested limit
			resp := makeResponse([]*Issue{
				makeIssue("TEST-1"),
				makeIssue("TEST-2"),
				makeIssue("TEST-3"),
				makeIssue("TEST-4"),
			}, "more-pages", false)
			_, _ = w.Write(resp)
		}))
		defer server.Close()

		client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
		result, err := client.Search("project=TEST", 0, 2)

		assert.NoError(t, err)
		assert.Len(t, result.Issues, 2)
		assert.False(t, result.IsLast, "IsLast should be false when we stopped due to limit")
	})
}
