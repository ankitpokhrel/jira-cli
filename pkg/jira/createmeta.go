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

// CreateMetaResponse struct holds response from GET /issue/createmeta endpoint.
type CreateMetaResponse struct {
	Projects []struct {
		Key        string                 `json:"key"`
		Name       string                 `json:"name"`
		IssueTypes []*CreateMetaIssueType `json:"issuetypes"`
	} `json:"projects"`
}

// CreateMetaIssueType struct holds issue types from GET /issue/createmeta endpoint.
type CreateMetaIssueType struct {
	IssueType
	Fields map[string]IssueTypeField `json:"fields"`
}

// GetCreateMeta gets create metadata using GET /issue/createmeta endpoint.
func (c *Client) GetCreateMeta(req *CreateMetaRequest) (*CreateMetaResponse, error) {
	path := fmt.Sprintf(
		"/issue/createmeta?projectKeys=%s&expand=%s",
		req.Projects, req.Expand,
	)
	if req.IssueTypeNames != "" {
		path += fmt.Sprintf("&issuetypeNames=%s", req.IssueTypeNames)
	}

	res, err := c.GetV2(context.Background(), path, nil)
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

	var out CreateMetaResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}
