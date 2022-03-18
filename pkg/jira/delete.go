package jira

import (
	"context"
	"fmt"
	"net/http"
)

// Delete deletes an issue using /issue/{key} endpoint.
func (c *Client) Delete(key string) error {
	res, err := c.DeleteV2(context.Background(), fmt.Sprintf("/issue/%s", key), nil)
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
