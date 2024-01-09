package jira

import (
	"context"
	"encoding/json"
	"net/http"
)

// Me struct holds response from /myself endpoint.
type Me struct {
	Login    string `json:"name"`
	Name     string `json:"displayName"`
	Email    string `json:"emailAddress"`
	Timezone string `json:"timeZone"`
}

// Me fetches response from /myself endpoint.
func (c *Client) Me() (*Me, error) {
	res, err := c.GetV2(context.Background(), "/myself", nil)
	if err != nil {
		return nil, err
	}
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}
	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var me Me

	err = json.NewDecoder(res.Body).Decode(&me)

	return &me, err
}
