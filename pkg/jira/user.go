package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ErrInvalidSearchOption denotes invalid search option was given.
var ErrInvalidSearchOption = fmt.Errorf("invalid search option")

// UserSearchOptions holds options to search for user.
type UserSearchOptions struct {
	Query      string
	Username   string
	AccountID  string
	StartAt    int
	MaxResults int
}

// UserSearch search for user details using GET /user/search endpoint.
func (c *Client) UserSearch(opt *UserSearchOptions) ([]*User, error) {
	if opt == nil {
		return nil, ErrInvalidSearchOption
	}

	var opts []string

	if opt.Query != "" {
		opts = append(opts, fmt.Sprintf("query=%s", url.QueryEscape(opt.Query)))
	}
	if opt.Username != "" {
		opts = append(opts, fmt.Sprintf("username=%s", url.QueryEscape(opt.Username)))
	}
	if opt.AccountID != "" {
		opts = append(opts, fmt.Sprintf("accountId=%s", url.QueryEscape(opt.AccountID)))
	}
	if opt.StartAt != 0 {
		opts = append(opts, fmt.Sprintf("startAt=%d", opt.StartAt))
	}
	if opt.MaxResults != 0 {
		opts = append(opts, fmt.Sprintf("maxResults=%d", opt.MaxResults))
	}
	if len(opts) == 0 {
		return nil, ErrInvalidSearchOption
	}
	path := fmt.Sprintf("%s?%s", "/user/search", strings.Join(opts, "&"))

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

	var out []*User
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
