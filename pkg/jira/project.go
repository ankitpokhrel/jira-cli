package jira

import (
	"context"
	"encoding/json"
	"net/http"
)

// Project fetches response from /search endpoint.
func (c *Client) Project() ([]*Project, error) {
	res, err := c.Get(context.Background(), "/project?expand=lead")
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

	var out []*Project

	err = json.NewDecoder(res.Body).Decode(&out)

	return out, err
}
