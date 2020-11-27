package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Me struct holds response from /myself endpoint.
type Me struct {
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
		return nil, fmt.Errorf("unauthorized")
	}

	var me Me

	err = json.NewDecoder(res.Body).Decode(&me)

	return &me, err
}
