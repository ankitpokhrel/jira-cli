//nolint:dupl
package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
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

	actual, err := client.Search("project=TEST AND status=Done ORDER BY created DESC", 100)
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

func TestSearchAllSinglePage(t *testing.T) {
	// Verifies that SearchAll returns all issues when the first (and only) response
	// has isLast=true. This is the simplest pagination case: no follow-up requests needed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		qs := r.URL.Query()
		assert.Equal(t, "project=TEST ORDER BY created DESC", qs.Get("jql"))
		assert.Equal(t, "*all", qs.Get("fields"))
		// With totalLimit=100, maxResults should be 100 (capped at maxPageSize).
		assert.Equal(t, "100", qs.Get("maxResults"))
		// First request should NOT have a nextPageToken parameter.
		assert.Empty(t, qs.Get("nextPageToken"))

		resp, err := os.ReadFile("./testdata/search.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 100)
	assert.NoError(t, err)

	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 3)
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
	assert.Equal(t, "TEST-2", actual.Issues[1].Key)
	assert.Equal(t, "TEST-3", actual.Issues[2].Key)
}

func TestSearchAllTwoPages(t *testing.T) {
	// Verifies cursor-based pagination across two pages: the first response returns
	// isLast=false with a nextPageToken, and the second returns isLast=true.
	// Issues from both pages must be combined in order without duplicates.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		page := atomic.AddInt32(&requestCount, 1)
		qs := r.URL.Query()

		var fixture string
		switch page {
		case 1:
			// First page: no nextPageToken in request.
			assert.Empty(t, qs.Get("nextPageToken"))
			fixture = "./testdata/search_page1.json"
		case 2:
			// Second page: must include the nextPageToken from page 1.
			assert.Equal(t, "token-page-2", qs.Get("nextPageToken"))
			fixture = "./testdata/search_page2.json"
		default:
			t.Fatalf("unexpected request count: %d", page)
		}

		resp, err := os.ReadFile(fixture)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 100)
	assert.NoError(t, err)

	// Should have made exactly 2 requests.
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))

	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 3)
	// Issues should be combined from both pages in order.
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
	assert.Equal(t, "TEST-2", actual.Issues[1].Key)
	assert.Equal(t, "TEST-3", actual.Issues[2].Key)
}

func TestSearchAllThreePages(t *testing.T) {
	// Verifies pagination across three pages. The chain is:
	// page1 (isLast=false, token="token-page-2") ->
	// page2 (isLast=false, token="token-page-3") ->
	// page3 (isLast=true).
	// All issues from all three pages must be accumulated correctly.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		page := atomic.AddInt32(&requestCount, 1)
		qs := r.URL.Query()

		var fixture string
		switch page {
		case 1:
			assert.Empty(t, qs.Get("nextPageToken"))
			fixture = "./testdata/search_page1.json"
		case 2:
			assert.Equal(t, "token-page-2", qs.Get("nextPageToken"))
			fixture = "./testdata/search_page2_mid.json"
		case 3:
			assert.Equal(t, "token-page-3", qs.Get("nextPageToken"))
			fixture = "./testdata/search_page3.json"
		default:
			t.Fatalf("unexpected request count: %d", page)
		}

		resp, err := os.ReadFile(fixture)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 500)
	assert.NoError(t, err)

	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 4)
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
	assert.Equal(t, "TEST-2", actual.Issues[1].Key)
	assert.Equal(t, "TEST-3", actual.Issues[2].Key)
	assert.Equal(t, "TEST-4", actual.Issues[3].Key)
}

func TestSearchAllLimitEnforcement(t *testing.T) {
	// Verifies that SearchAll stops fetching additional pages once the totalLimit
	// is reached. With a limit of 2 and 2 issues on the first page, the second
	// page should not be fetched even though isLast is false.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		page := atomic.AddInt32(&requestCount, 1)

		var fixture string
		switch page {
		case 1:
			fixture = "./testdata/search_page1.json"
		default:
			// SearchAll should stop after the first page because we already have 2 issues == limit.
			t.Fatalf("should not have fetched page %d; limit was already reached", page)
			return
		}

		resp, err := os.ReadFile(fixture)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 2)
	assert.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 2)
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
	assert.Equal(t, "TEST-2", actual.Issues[1].Key)
}

func TestSearchAllLimitTrimsOvershoot(t *testing.T) {
	// Verifies that if a page returns more issues than needed to reach totalLimit,
	// the result is trimmed to exactly totalLimit. Here we request limit=1 but the
	// first page returns 2 issues; the result must be trimmed to 1.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		qs := r.URL.Query()
		// With totalLimit=1, maxResults should be 1 (less than maxPageSize).
		assert.Equal(t, "1", qs.Get("maxResults"))

		// Return 2 issues even though maxResults=1 was requested. The API might
		// not always honor maxResults exactly, so SearchAll must trim to the limit.
		resp, err := os.ReadFile("./testdata/search_page1.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 1)
	assert.NoError(t, err)

	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 1)
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
}

func TestSearchAllLimitLessThanPageSize(t *testing.T) {
	// Verifies that when totalLimit < maxPageSize (100), the maxResults parameter
	// is set to totalLimit rather than the full page size. This avoids fetching
	// more data than needed from the API.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		qs := r.URL.Query()
		assert.Equal(t, "50", qs.Get("maxResults"))

		resp, err := os.ReadFile("./testdata/search.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 50)
	assert.NoError(t, err)

	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 3)
}

func TestSearchAllLimitExceedsPageSize(t *testing.T) {
	// Verifies that when totalLimit > maxPageSize (100), the maxResults parameter
	// is capped at 100 per request, and pagination is used to collect up to totalLimit.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		page := atomic.AddInt32(&requestCount, 1)
		qs := r.URL.Query()

		// maxResults should be capped at 100 for the first request.
		assert.Equal(t, "100", qs.Get("maxResults"))

		var fixture string
		switch page {
		case 1:
			fixture = "./testdata/search_page1.json"
		case 2:
			fixture = "./testdata/search_page2.json"
		default:
			t.Fatalf("unexpected request count: %d", page)
		}

		resp, err := os.ReadFile(fixture)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 200)
	assert.NoError(t, err)

	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 3)
}

func TestSearchAllEmptyResults(t *testing.T) {
	// Verifies that SearchAll handles the case where the JQL matches no issues.
	// The API returns isLast=true with an empty issues array on the first request.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		resp, err := os.ReadFile("./testdata/search_empty.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=EMPTY ORDER BY created DESC", 100)
	assert.NoError(t, err)

	assert.True(t, actual.IsLast)
	assert.Empty(t, actual.Issues)
}

func TestSearchAllEmptyTokenStopsPagination(t *testing.T) {
	// Defensive check: if the API returns isLast=false but an empty nextPageToken,
	// SearchAll must stop to avoid infinite loops. This guards against malformed
	// API responses.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		// Return isLast=false but no nextPageToken — a contradictory response.
		result := SearchResult{
			IsLast:        false,
			NextPageToken: "",
			Issues: []*Issue{
				{Key: "TEST-1", Fields: IssueFields{Summary: "issue 1", Labels: []string{}}},
			},
		}
		resp, err := json.Marshal(result)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 100)
	assert.NoError(t, err)

	// Should return what we got rather than looping forever.
	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 1)
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
}

func TestSearchAllSingleIssue(t *testing.T) {
	// Verifies SearchAll works correctly when there is exactly one issue total.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		result := SearchResult{
			IsLast: true,
			Issues: []*Issue{
				{Key: "TEST-99", Fields: IssueFields{Summary: "only issue", Labels: []string{}}},
			},
		}
		resp, err := json.Marshal(result)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 100)
	assert.NoError(t, err)

	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 1)
	assert.Equal(t, "TEST-99", actual.Issues[0].Key)
}

func TestSearchAllErrorOnFirstPage(t *testing.T) {
	// Verifies that if the API returns an error on the first request,
	// SearchAll propagates the error immediately with no partial results.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 100)
	assert.Error(t, err)
	assert.Nil(t, actual)
}

func TestSearchAllErrorMidPagination(t *testing.T) {
	// Verifies that if the first page succeeds but the second page returns an error,
	// SearchAll returns an error. The implementation does not return partial results;
	// it propagates the error from the failing page.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&requestCount, 1)

		switch page {
		case 1:
			resp, err := os.ReadFile("./testdata/search_page1.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		case 2:
			// Second page returns an HTTP error.
			w.WriteHeader(500)
		default:
			t.Fatalf("unexpected request count: %d", page)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 200)
	assert.Error(t, err)
	assert.Nil(t, actual)
	// Verify both requests were actually made.
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
}

func TestSearchAllInvalidJSON(t *testing.T) {
	// Verifies that a malformed JSON response is surfaced as an error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"isLast": true, "issues": [`))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 100)
	assert.Error(t, err)
	assert.Nil(t, actual)
}

func TestSearchAllPageSizeAdjustment(t *testing.T) {
	// Verifies that SearchAll adjusts the maxResults for the last page when the
	// remaining count is less than the page size. With totalLimit=3 and 2 issues
	// on page 1, the second page request should have maxResults=1.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		page := atomic.AddInt32(&requestCount, 1)
		qs := r.URL.Query()

		switch page {
		case 1:
			assert.Equal(t, "3", qs.Get("maxResults"))

			resp, err := os.ReadFile("./testdata/search_page1.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		case 2:
			// Remaining = 3 - 2 = 1, so maxResults should be adjusted to 1.
			assert.Equal(t, "1", qs.Get("maxResults"))

			resp, err := os.ReadFile("./testdata/search_page2.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		default:
			t.Fatalf("unexpected request count: %d", page)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 3)
	assert.NoError(t, err)

	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 3)
}

func TestSearchAllZeroLimit(t *testing.T) {
	// Verifies that SearchAll handles totalLimit=0 gracefully by treating it as
	// "fetch all available results". The pageSize should default to maxPageSize (100)
	// and pagination should continue until isLast=true.
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		page := atomic.AddInt32(&requestCount, 1)
		qs := r.URL.Query()

		// With totalLimit=0, maxResults should default to maxPageSize (100).
		assert.Equal(t, "100", qs.Get("maxResults"))

		var fixture string
		switch page {
		case 1:
			assert.Empty(t, qs.Get("nextPageToken"))
			fixture = "./testdata/search_page1.json"
		case 2:
			assert.Equal(t, "token-page-2", qs.Get("nextPageToken"))
			fixture = "./testdata/search_page2.json"
		default:
			t.Fatalf("unexpected request count: %d", page)
		}

		resp, err := os.ReadFile(fixture)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST ORDER BY created DESC", 0)
	assert.NoError(t, err)

	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
	assert.True(t, actual.IsLast)
	assert.Len(t, actual.Issues, 3)
	assert.Equal(t, "TEST-1", actual.Issues[0].Key)
	assert.Equal(t, "TEST-2", actual.Issues[1].Key)
	assert.Equal(t, "TEST-3", actual.Issues[2].Key)
}

func TestSearchAllPassesNextPageToken(t *testing.T) {
	// Verifies that SearchAll correctly URL-encodes and passes the nextPageToken
	// from each response to the subsequent request.
	var requestCount int32
	// Use tokens that require URL encoding to verify proper handling.
	tokens := []string{"", "abc+def/123=", "xyz+uvw/456="}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&requestCount, 1)
		qs := r.URL.Query()

		// Verify the token sent in this request matches what we expect.
		expectedToken := tokens[page-1]
		assert.Equal(t, expectedToken, qs.Get("nextPageToken"),
			fmt.Sprintf("page %d: unexpected nextPageToken", page))

		var result SearchResult
		switch page {
		case 1:
			result = SearchResult{
				IsLast:        false,
				NextPageToken: tokens[1],
				Issues:        []*Issue{{Key: "TEST-1", Fields: IssueFields{Labels: []string{}}}},
			}
		case 2:
			result = SearchResult{
				IsLast:        false,
				NextPageToken: tokens[2],
				Issues:        []*Issue{{Key: "TEST-2", Fields: IssueFields{Labels: []string{}}}},
			}
		case 3:
			result = SearchResult{
				IsLast: true,
				Issues: []*Issue{{Key: "TEST-3", Fields: IssueFields{Labels: []string{}}}},
			}
		default:
			t.Fatalf("unexpected request count: %d", page)
		}

		resp, err := json.Marshal(result)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.SearchAll("project=TEST", 500)
	assert.NoError(t, err)

	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
	assert.Len(t, actual.Issues, 3)
}
