package jira

import (
	"context"
	"encoding/json"
	"net/http"
)

// ServerInfo struct holds response from /serverInfo endpoint.
type ServerInfo struct {
	Version        string `json:"version"`
	VersionNumbers []int  `json:"versionNumbers"`
	DeploymentType string `json:"deploymentType"`
	BuildNumber    int    `json:"buildNumber"`
	DefaultLocale  struct {
		Locale string `json:"locale"`
	} `json:"defaultLocale"`
}

// ServerInfo fetches response from /serverInfo endpoint.
func (c *Client) ServerInfo() (*ServerInfo, error) {
	res, err := c.GetV2(context.Background(), "/serverInfo", nil)
	if err != nil {
		return nil, err
	}
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}
	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var info ServerInfo

	err = json.NewDecoder(res.Body).Decode(&info)

	return &info, err
}

// SupportsV9CreateMeta reports whether the installation should use the Jira
// Server v9 createmeta endpoint.
func SupportsV9CreateMeta(installation string, major, minor int) bool {
	if installation != InstallationTypeLocal {
		return false
	}
	//nolint:mnd
	return major >= 9 || (major == 8 && minor > 4)
}
