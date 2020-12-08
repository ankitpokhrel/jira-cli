package jira

import (
	"context"
	"encoding/json"
	"net/http"
)

// Me struct holds response from /myself endpoint.
type Me struct {
	Name  string `json:"displayName"`
	Email string `json:"emailAddress"`
}

// Me fetches response from /myself endpoint.
func (c *Client) Me() (*Me, error) {
	res, err := c.Get(context.Background(), "/myself")
	if err != nil {
		return nil, err
	}

	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}

	if res.StatusCode != http.StatusOK {
		return nil, errUnexpectedStatusCode
	}

	var me Me

	err = json.NewDecoder(res.Body).Decode(&me)

	return &me, err
}
