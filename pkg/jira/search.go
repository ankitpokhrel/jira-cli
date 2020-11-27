package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Search struct holds response from /search endpoint.
type Search struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// Search fetches response from /search endpoint.
func (c *Client) Search(jql string) (*Search, error) {
	res, err := c.Get(context.Background(), "/search?jql="+url.QueryEscape(jql))
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errEmptyResponse
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, errUnexpectedStatusCode
	}

	var out Search

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
