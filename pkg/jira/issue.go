package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetIssue fetches issue details using GET /issue/{key} endpoint.
func (c *Client) GetIssue(key string) (*Issue, error) {
	path := fmt.Sprintf("/issue/%s", key)

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

	var out Issue

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
