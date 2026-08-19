package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	// maxSearchPageSize is the maximum number of results per page for the Jira search API.
	maxSearchPageSize = 100
)

// SearchResult struct holds response from /search endpoint.
type SearchResult struct {
	IsLast        bool     `json:"isLast"`
	NextPageToken string   `json:"nextPageToken"`
	Issues        []*Issue `json:"issues"`
}

// Search searches for issues using v3 version of the Jira GET /search/jql endpoint.
//
// It supports cursor-based pagination using nextPageToken from the API response.
// The from parameter specifies how many items to skip, and limit specifies the
// maximum number of items to return. For large offsets, multiple API requests
// may be made internally to reach the requested position.
func (c *Client) Search(jql string, from, limit uint) (*SearchResult, error) {
	var (
		allIssues []*Issue
		skipped   uint
		nextToken string
	)

	// Determine API page size: use requested limit if no offset, otherwise use max for efficiency.
	// When we need to skip items (from > 0), larger pages reduce the number of API calls needed.
	pageSize := limit
	if from > 0 && pageSize < maxSearchPageSize {
		pageSize = maxSearchPageSize
	}

	for {
		path := fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=*all", url.QueryEscape(jql), pageSize)
		if nextToken != "" {
			path += "&nextPageToken=" + url.QueryEscape(nextToken)
		}

		result, err := c.search(path, apiVersion3)
		if err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			// Skip items until we reach the 'from' offset
			if skipped < from {
				skipped++
				continue
			}

			allIssues = append(allIssues, issue)

			// Stop if we've collected enough items
			if uint(len(allIssues)) >= limit {
				return &SearchResult{
					Issues: allIssues,
					IsLast: false, // We stopped early, so there may be more results
				}, nil
			}
		}

		// If this is the last page, we're done
		if result.IsLast || result.NextPageToken == "" {
			break
		}

		nextToken = result.NextPageToken
	}

	return &SearchResult{
		Issues: allIssues,
		IsLast: true,
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
