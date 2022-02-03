package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
)

// SearchResult struct holds response from /search endpoint.
type SearchResult struct {
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Total      int      `json:"total"`
	Issues     []*Issue `json:"issues"`
}

// Search searches for issues using v3 version of the Jira GET /search endpoint.
func (c *Client) Search(jql string, limit uint) (*SearchResult, error) {
	return c.search(jql, limit, apiVersion3)
}

// SearchV2 searches an issues using v2 version of the Jira GET /search endpoint.
func (c *Client) SearchV2(jql string, limit uint) (*SearchResult, error) {
	return c.search(jql, limit, apiVersion2)
}

func (c *Client) search(jql string, limit uint, ver string) (*SearchResult, error) {
	var (
		res *http.Response
		err error
	)

	// maxResults is a int32
	// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/#api-rest-api-3-search-get
	if math.MaxInt32 < limit {
		limit = math.MaxInt32
	}
	path := fmt.Sprintf("/search?jql=%s&maxResults=%d", url.QueryEscape(jql), limit)

	switch ver {
	case apiVersion2:
		res, err = c.GetV2(context.Background(), path, nil)
	default:
		res, err = c.Get(context.Background(), path, nil)
	}

	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out SearchResult

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
