package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// GetIssueAttachments fetches attachments for an issue using v3 API.
func (c *Client) GetIssueAttachments(key string) ([]Attachment, error) {
	return c.getIssueAttachments(key, apiVersion3)
}

// GetIssueAttachmentsV2 fetches attachments for an issue using v2 API.
func (c *Client) GetIssueAttachmentsV2(key string) ([]Attachment, error) {
	return c.getIssueAttachments(key, apiVersion2)
}

func (c *Client) getIssueAttachments(key, ver string) ([]Attachment, error) {
	path := fmt.Sprintf("/issue/%s?fields=attachment", key)

	var (
		res *http.Response
		err error
	)

	switch ver {
	case apiVersion2:
		res, err = c.GetV2(context.Background(), path, nil)
	default:
		res, err = c.Get(context.Background(), path, nil)
	}

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

	var issue Issue
	if err := json.NewDecoder(res.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return issue.Fields.Attachment, nil
}

// DownloadAttachment downloads an attachment from the given URL to the target path.
func (c *Client) DownloadAttachment(contentURL, targetPath string) error {
	res, err := c.request(context.Background(), http.MethodGet, contentURL, nil, nil)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return formatUnexpectedResponse(res)
	}

	// Ensure directory exists
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Copy content
	_, err = io.Copy(out, res.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
