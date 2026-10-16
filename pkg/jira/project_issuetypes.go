package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type projectIssueTypesResponse struct {
	IssueTypes []*IssueType `json:"issueTypes"`
}

// ProjectIssueTypes fetches issue types for a given project using
// GET /rest/api/2/project/{projectKey}?expand=issueTypes.
//
// This endpoint does not provide create-screen field requirements (unlike createmeta),
// but it can be used as a fallback when createmeta is not available.
func (c *Client) ProjectIssueTypes(projectKey string) ([]*IssueType, error) {
	path := fmt.Sprintf("/project/%s?expand=%s", url.PathEscape(projectKey), url.QueryEscape("issueTypes"))

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

	var out projectIssueTypesResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out.IssueTypes, nil
}
