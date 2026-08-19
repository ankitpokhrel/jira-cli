package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ProjectComponents fetches response from /project/{projectIdOrKey}/components endpoint.
func (c *Client) ProjectComponents(project string) ([]*ProjectComponent, error) {
	return c.projectComponents(project, apiVersion3)
}

// ProjectComponentsV2 fetches response from /project/{projectIdOrKey}/components endpoint.
func (c *Client) ProjectComponentsV2(project string) ([]*ProjectComponent, error) {
	return c.projectComponents(project, apiVersion2)
}

func (c *Client) projectComponents(project, ver string) ([]*ProjectComponent, error) {
	path := fmt.Sprintf("/project/%s/components", project)

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

	var out []*ProjectComponent

	err = json.NewDecoder(res.Body).Decode(&out)

	return out, err
}
