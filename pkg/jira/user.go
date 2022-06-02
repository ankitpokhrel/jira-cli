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
	Project    string
	Query      string
	Username   string
	AccountID  string
	StartAt    int
	MaxResults int
}

// UserSearch search for user details using v3 version of the GET /user/assignable/search endpoint.
func (c *Client) UserSearch(opt *UserSearchOptions) ([]*User, error) {
	return c.userSearch(opt, apiVersion3)
}

// UserSearchV2 search for user details using v2 version of the GET /user/assignable/search endpoint.
func (c *Client) UserSearchV2(opt *UserSearchOptions) ([]*User, error) {
	// The `username` query param is deprecated since Jira API v2 and is not available in v3.
	// Since the` query` parameter doesn't seem to return expected results, we will use the
	// `username` param in call to v2. Chances are the `query` param may stop working in
	// later v2 updates so we might have to revisit this in the future. Note that the
	// `username` param is not as flexible as the `query` param available in v3.
	//
	// See https://github.com/ankitpokhrel/jira-cli/issues/198
	if opt.Query != "" && opt.Username == "" {
		opt.Username = opt.Query
		opt.Query = ""
	}
	return c.userSearch(opt, apiVersion2)
}

func (c *Client) userSearch(opt *UserSearchOptions, ver string) ([]*User, error) {
	if opt == nil {
		return nil, ErrInvalidSearchOption
	}

	var (
		opts []string
		res  *http.Response
		err  error
	)

	if opt.Project != "" {
		opts = append(opts, fmt.Sprintf("project=%s", opt.Project))
	}
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

	path := fmt.Sprintf("%s?%s", "/user/assignable/search", strings.Join(opts, "&"))

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

	var out []*User
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
