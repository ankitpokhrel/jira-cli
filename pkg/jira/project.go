package jira

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	// ProjectTypeClassic is a classic project type.
	ProjectTypeClassic = "classic"
	// ProjectTypeNextGen is a next gen project type.
	ProjectTypeNextGen = "next-gen"
)

// Project fetches response from /project endpoint.
func (c *Client) Project() ([]*Project, error) {
	raw, err := c.ProjectRaw()
	if err != nil {
		return nil, err
	}

	var out []*Project

	err = json.Unmarshal([]byte(raw), &out)

	return out, err
}

// ProjectRaw fetches response from /project endpoint same as Project but returns the raw API response body string.
func (c *Client) ProjectRaw() (string, error) {
	res, err := c.GetV2(context.Background(), "/project?expand=lead", nil)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return "", formatUnexpectedResponse(res)
	}

	var b strings.Builder

	_, err = io.Copy(&b, res.Body)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
