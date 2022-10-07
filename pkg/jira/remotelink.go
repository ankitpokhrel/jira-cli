package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type weblinkRequest struct {
	RemoteObject struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	} `json:"object"`
}

// WebLinkIssue adds a web link to an issue using POST /issue/{issueId}/remotelink endpoint.
func (c *Client) WebLinkIssue(issueID, title, url string) error {
	body, err := json.Marshal(weblinkRequest{
		RemoteObject: struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		}{Title: title, URL: url},
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/remotelink", issueID)

	res, err := c.PostV2(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return formatUnexpectedResponse(res)
	}
	return nil
}
