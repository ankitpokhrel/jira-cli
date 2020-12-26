package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

// GetIssue fetches issue details using GET /issue/{key} endpoint.
func (c *Client) GetIssue(key string) (*Issue, error) {
	path := fmt.Sprintf("/issue/%s", key)

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

	var out Issue

	err = json.NewDecoder(res.Body).Decode(&out)

	out.Fields.Description = ifaceToADF(out.Fields.Description)

	return &out, err
}

func ifaceToADF(v interface{}) *adf.ADF {
	if v == nil {
		return nil
	}
	var doc *adf.ADF
	js, err := json.Marshal(v)
	if err != nil {
		return nil // ignore invalid data
	}
	if err = json.Unmarshal(js, &doc); err != nil {
		return nil // ignore invalid data
	}
	return doc
}
