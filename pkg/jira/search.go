package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SearchResult struct holds response from /search endpoint.
type SearchResult struct {
	IsLast        bool     `json:"isLast"`
	NextPageToken string   `json:"nextPageToken"`
	Issues        []*Issue `json:"issues"`
}

// Search searches for issues using v3 version of the Jira GET /search endpoint.
// If fields is empty, defaults to "*all" which includes most fields.
// Specific fields can be requested by comma-separated list (e.g., "key,summary,customfield_10001").
func (c *Client) Search(jql string, limit uint, fields string) (*SearchResult, error) {
	if fields == "" {
		fields = "*all"
	}
	path := fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=%s", url.QueryEscape(jql), limit, url.QueryEscape(fields))
	return c.search(path, apiVersion3)
}

// SearchV2 searches an issues using v2 version of the Jira GET /search endpoint.
// If fields is empty, no fields parameter is added (Jira returns defaults).
func (c *Client) SearchV2(jql string, from, limit uint, fields string) (*SearchResult, error) {
	path := fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d", url.QueryEscape(jql), from, limit)
	if fields != "" {
		path = fmt.Sprintf("%s&fields=%s", path, url.QueryEscape(fields))
	}
	return c.search(path, apiVersion2)
}

func (c *Client) search(path, ver string) (*SearchResult, error) {
	var (
		res *http.Response
		err error
	)

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
