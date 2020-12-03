package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Epic struct holds epic response from /search endpoint.
type Epic struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// Epic fetches epics using the /search endpoint.
func (c *Client) Epic(jql string) (*Search, error) {
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

// EpicIssues fetches issues in the given epic.
func (c *Client) EpicIssues(key string) (*Search, error) {
	res, err := c.GetV1(context.Background(), fmt.Sprintf("/epic/%s/issue", key))
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
