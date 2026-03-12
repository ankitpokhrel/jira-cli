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

const maxPageSize uint = 100

// Search searches for issues using v3 version of the Jira GET /search endpoint.
// This performs a single request and returns up to `limit` results.
func (c *Client) Search(jql string, limit uint) (*SearchResult, error) {
	path := fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=*all", url.QueryEscape(jql), limit)
	return c.search(path, apiVersion3)
}

// SearchAll searches for issues using v3 version of the Jira GET /search endpoint
// with automatic cursor-based pagination via nextPageToken. It fetches pages of up
// to 100 issues at a time until `totalLimit` issues are collected or all results
// are exhausted (isLast == true).
func (c *Client) SearchAll(jql string, totalLimit uint) (*SearchResult, error) {
	var allIssues []*Issue

	pageSize := totalLimit
	if pageSize == 0 || pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	nextPageToken := ""
	for {
		path := fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=*all", url.QueryEscape(jql), pageSize)
		if nextPageToken != "" {
			path += fmt.Sprintf("&nextPageToken=%s", url.QueryEscape(nextPageToken))
		}

		result, err := c.search(path, apiVersion3)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, result.Issues...)

		if result.IsLast || result.NextPageToken == "" {
			break
		}
		if totalLimit > 0 && uint(len(allIssues)) >= totalLimit {
			break
		}

		nextPageToken = result.NextPageToken

		// Adjust page size for the last page if needed.
		if totalLimit > 0 {
			remaining := totalLimit - uint(len(allIssues))
			if remaining < pageSize {
				pageSize = remaining
			}
		}
	}

	// Trim to totalLimit if we overshot.
	if totalLimit > 0 && uint(len(allIssues)) > totalLimit {
		allIssues = allIssues[:totalLimit]
	}

	return &SearchResult{
		IsLast: true,
		Issues: allIssues,
	}, nil
}

// SearchV2 searches an issues using v2 version of the Jira GET /search endpoint.
func (c *Client) SearchV2(jql string, from, limit uint) (*SearchResult, error) {
	path := fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d", url.QueryEscape(jql), from, limit)
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
