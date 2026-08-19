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

// CreateMetaResponseJiraServerV9 struct holds response from GET /issue/createmeta endpoint
// for jira server 9 and above.
type CreateMetaResponseJiraServerV9 struct {
	Values []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Subtask bool   `json:"subtask"`
	} `json:"values"`
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

// GetCreateMetaForJiraServerV9 gets create metadata using GET /issue/createmeta endpoint for jira server 9 and above.
func (c *Client) GetCreateMetaForJiraServerV9(req *CreateMetaRequest) (*CreateMetaResponseJiraServerV9, error) {
	path := fmt.Sprintf(
		"/issue/createmeta/%s/issuetypes?expand=%s",
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

	var out CreateMetaResponseJiraServerV9

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

// ProjectIssueTypes returns the issue types available for creating issues
// in the given project.
func (c *Client) ProjectIssueTypes(project string, serverV9 bool) ([]*IssueType, error) {
	req := &CreateMetaRequest{
		Projects: project,
		Expand:   "projects.issuetypes.fields",
	}

	if serverV9 {
		meta, err := c.GetCreateMetaForJiraServerV9(req)
		if err != nil {
			return nil, err
		}

		issueTypes := make([]*IssueType, 0, len(meta.Values))

		for _, it := range meta.Values {
			issueTypes = append(issueTypes, &IssueType{
				ID:      it.ID,
				Name:    it.Name,
				Subtask: it.Subtask,
			})
		}

		return issueTypes, nil
	}

	meta, err := c.GetCreateMeta(req)
	if err != nil {
		return nil, err
	}
	if len(meta.Projects) == 0 {
		return []*IssueType{}, nil
	}

	issueTypes := make([]*IssueType, 0, len(meta.Projects[0].IssueTypes))

	for _, it := range meta.Projects[0].IssueTypes {
		issueTypes = append(issueTypes, &IssueType{
			ID:      it.ID,
			Name:    it.Name,
			Handle:  it.Handle,
			Subtask: it.Subtask,
		})
	}

	return issueTypes, nil
}
