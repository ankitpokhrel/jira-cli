package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Epic fetches epics using the /search endpoint.
func (c *Client) Epic(jql string) (*SearchResult, error) {
	res, err := c.Get(context.Background(), "/search?jql="+url.QueryEscape(jql), nil)
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

// EpicIssues fetches issues in the given epic.
func (c *Client) EpicIssues(key, jql string, limit uint) (*SearchResult, error) {
	path := fmt.Sprintf("/epic/%s/issue?maxResults=%d", key, limit)
	if jql != "" {
		path += fmt.Sprintf("&jql=%s", url.QueryEscape(jql))
	}

	res, err := c.GetV1(context.Background(), path, nil)
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
