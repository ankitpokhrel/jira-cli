package jira

import (
	"context"
	"fmt"
	"net/http"
)

// DeleteIssue deletes an issue using /issue/{key} endpoint.
func (c *Client) DeleteIssue(key string, cascade bool) error {
	path := fmt.Sprintf("/issue/%s", key)
	if cascade {
		path = fmt.Sprintf("%s?deleteSubtasks=true", path)
	}

	res, err := c.DeleteV2(context.Background(), path, nil)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent {
		return formatUnexpectedResponse(res)
	}
	return nil
}
