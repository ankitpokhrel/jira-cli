package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const maxResults = 100

// SearchResult struct holds response from /search endpoint.
type SearchResult struct {
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Total      int      `json:"total"`
	Issues     []*Issue `json:"issues"`
}

// Search fetches response from /search endpoint.
func (c *Client) Search(jql string) (*SearchResult, error) {
	path := fmt.Sprintf("/search?jql=%s&maxResults=%d", url.QueryEscape(jql), maxResults)

	res, err := c.Get(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrEmptyResponse
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, ErrUnexpectedStatusCode
	}

	var out SearchResult

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
