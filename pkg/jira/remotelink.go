package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type weblinkRequest struct {
	RemoteObject struct {
		Url   string `json:"url"`
		Title string `json:"title"`
	} `json:"object"`
}

// Add a web link to an issue using POST /issue/{issueId}/remotelink endpoint.
func (c *Client) WebLinkIssue(issueId, title, url string) error {
	body, err := json.Marshal(weblinkRequest{
		RemoteObject: struct {
			Url   string `json:"url"`
			Title string `json:"title"`
		}{Title: title, Url: url},
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/transitions", issueId)

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
