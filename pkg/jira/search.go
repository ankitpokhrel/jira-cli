package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	SearchLimitDefault uint = 100
)

// SearchResult struct holds response from /search endpoint.
type SearchResult struct {
	IsLast        bool     `json:"isLast"`
	NextPageToken string   `json:"nextPageToken"`
	Issues        []*Issue `json:"issues"`
}

func MergeSearchResults(results ...*SearchResult) *SearchResult {
	if len(results) == 0 {
		return nil
	}

	newResult := &SearchResult{}
	for _, result := range results {
		if result == nil {
			continue
		}
		newResult.Issues = append(newResult.Issues, result.Issues...)
		newResult.NextPageToken = result.NextPageToken
		newResult.IsLast = result.IsLast
	}

	return newResult
}

type SearchOptions struct {
	Limit         uint
	From          *uint
	NextPageToken string
	APIVersion    string
}

func WithAPIVersion2() SearchOption {
	return func(o *SearchOptions) {
		o.APIVersion = apiVersion2
	}
}

type SearchOption func(*SearchOptions)

// SearchWith searches for issues using v3 version of the Jira GET /search endpoint.
func (c *Client) SearchWith(jql string, options ...SearchOption) (*SearchResult, error) {
	opts := &SearchOptions{
		Limit:      SearchLimitDefault,
		APIVersion: apiVersion3,
	}
	for _, option := range options {
		option(opts)
	}

	params := []string{
		fmt.Sprintf("jql=%s", url.QueryEscape(jql)),
		fmt.Sprintf("maxResults=%d", opts.Limit),
	}

	if opts.NextPageToken != "" {
		params = append(params, fmt.Sprintf("nextPageToken=%s", opts.NextPageToken))
	} else if opts.From != nil && opts.APIVersion == apiVersion2 {
		params = append(params, fmt.Sprintf("startAt=%d", *opts.From))
	}

	urlPath := "/search"

	if opts.APIVersion == apiVersion3 {
		params = append(params, "fields=*all")
		urlPath = "/search/jql"
	}

	paramsString := strings.Join(params, "&")

	path := fmt.Sprintf("%s?%s", urlPath, paramsString)
	return c.search(path, opts.APIVersion)
}

// Search searches for issues using v3 version of the Jira GET /search endpoint.
func (c *Client) Search(jql string, limit uint) (*SearchResult, error) {
	return c.SearchWith(jql, func(options *SearchOptions) {
		options.Limit = limit
	})
}

// SearchV2 searches an issues using v2 version of the Jira GET /search endpoint.
func (c *Client) SearchV2(jql string, from, limit uint) (*SearchResult, error) {
	return c.SearchWith(jql, func(options *SearchOptions) {
		options.Limit = limit
		options.From = &from
		options.APIVersion = apiVersion2
	})
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
