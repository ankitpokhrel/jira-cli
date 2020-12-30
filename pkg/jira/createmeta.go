package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CreateMetaRequest struct holds request data for createmeta request.
type CreateMetaRequest struct {
	Projects       string
	IssueTypeNames string
	Expand         string
}

// CreateMetaResponse struct holds response from POST /issue/createmeta endpoint.
type CreateMetaResponse struct {
	Projects []struct {
		Key        string `json:"key"`
		Name       string `json:"name"`
		IssueTypes []struct {
			Name   string                 `json:"name"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"issuetypes"`
	} `json:"projects"`
}

// GetCreateMeta gets create metadata using GET /issue/createmeta endpoint.
func (c *Client) GetCreateMeta(req *CreateMetaRequest) (*CreateMetaResponse, error) {
	path := fmt.Sprintf(
		"/issue/createmeta?projectKeys=%s&issuetypeNames=%s&expand=%s",
		req.Projects, req.IssueTypeNames, req.Expand,
	)

	res, err := c.Get(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, ErrUnexpectedStatusCode
	}

	var out CreateMetaResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
