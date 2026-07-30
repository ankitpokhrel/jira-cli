package jira

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

var (
	ErrMultipleCloudIDs = errors.New("multiple cloud IDs found, unable to determine which to use")
	ErrEmptyCloudID     = errors.New("empty cloud ID returned")
)

const (
	AccessibleResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"
)

type CloudIDResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Scopes    []string `json:"scopes"`
	AvatarURL string   `json:"avatarUrl"`
}

func (c *Client) GetCloudID() (string, error) {
	envCloudID := os.Getenv("JIRA_CLI_CLOUD_ID")
	if envCloudID != "" {
		return envCloudID, nil
	}

	res, err := c.request(context.Background(), http.MethodGet, AccessibleResourcesURL, nil, Header{
		"Accept": "application/json",
	})
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
	var out []CloudIDResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return "", err
	}
	if len(out) == 0 {
		return "", ErrEmptyResponse
	}
	// Return the first cloud ID found
	if len(out) > 1 {
		return "", ErrMultipleCloudIDs
	}
	if out[0].ID == "" {
		return "", ErrEmptyCloudID
	}
	// Return the account ID

	return out[0].ID, nil
}
