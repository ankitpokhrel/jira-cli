package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// BoardTypeScrum represents a scrum board type.
	BoardTypeScrum = "scrum"
	// BoardTypeAll represents all board types.
	BoardTypeAll = ""
)

// BoardResult holds response from /board endpoint.
type BoardResult struct {
	MaxResults int      `json:"maxResults"`
	Total      int      `json:"total"`
	Boards     []*Board `json:"values"`
}

// Boards gets all boards of a given type in a project.
func (c *Client) Boards(project, boardType string) (*BoardResult, error) {
	path := fmt.Sprintf("/board?projectKeyOrId=%s", project)
	if boardType != "" {
		path += fmt.Sprintf("&type=%s", boardType)
	}

	return c.board(path)
}

// BoardSearch fetches boards with the given name in a project.
func (c *Client) BoardSearch(project, name string) (*BoardResult, error) {
	path := fmt.Sprintf("/board?projectKeyOrId=%s&name=%s", project, name)

	return c.board(path)
}

func (c *Client) board(path string) (*BoardResult, error) {
	res, err := c.GetV1(context.Background(), path, nil)
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

	var out BoardResult

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
