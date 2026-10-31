package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// attachmentNoCheckTokenHeader disables Jira's XSRF check for attachment uploads.
const attachmentNoCheckTokenHeader = "X-Atlassian-Token"

// Attachment holds attachment info for an issue.
type Attachment struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mimeType"`
	Content   string `json:"content"`
	Author    User   `json:"author"`
	Created   string `json:"created"`
	Self      string `json:"self,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// AttachmentUploadResult holds the upload result for a single file.
type AttachmentUploadResult struct {
	Path       string
	Attachment Attachment
}

// UploadAttachment uploads one or more files to an issue using
// POST /rest/api/2/issue/{key}/attachments endpoint.
func (c *Client) UploadAttachment(key string, paths []string) ([]Attachment, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("jira: no files provided for upload")
	}

	files := make([]*os.File, 0, len(paths))
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			for _, opened := range files {
				_ = opened.Close()
			}
			return nil, fmt.Errorf("jira: cannot open file %q: %w", p, err)
		}
		files = append(files, f)
	}
	defer func() {
		for _, f := range files {
			_ = f.Close()
		}
	}()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for i, f := range files {
		part, err := w.CreateFormFile("file", filepath.Base(paths[i]))
		if err != nil {
			return nil, fmt.Errorf("jira: cannot create multipart form: %w", err)
		}
		if _, err := io.Copy(part, f); err != nil {
			return nil, fmt.Errorf("jira: cannot read file %q: %w", paths[i], err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("jira: cannot finalize multipart form: %w", err)
	}

	endpoint := c.server + baseURLv2 + fmt.Sprintf("/issue/%s/attachments", key)
	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodPost, endpoint, bytes.NewReader(buf.Bytes()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set(attachmentNoCheckTokenHeader, "no-check")
	c.setAuthHeader(req)

	httpClient := &http.Client{Transport: c.transport}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, formatUnexpectedResponse(res)
	}

	var out []Attachment
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("jira: cannot decode upload response: %w", err)
	}
	return out, nil
}

// setAuthHeader applies the configured auth type to a request.
func (c *Client) setAuthHeader(req *http.Request) {
	// Set default auth type to `basic`.
	if c.authType == nil {
		basic := AuthTypeBasic
		c.authType = &basic
	}

	switch c.authType.String() {
	case string(AuthTypeMTLS):
		if c.token != "" {
			req.Header.Add("Authorization", "Bearer "+c.token)
		}
	case string(AuthTypeBearer):
		req.Header.Add("Authorization", "Bearer "+c.token)
	case string(AuthTypeBasic):
		req.SetBasicAuth(c.login, c.token)
	}
}

// ListAttachments returns attachments of an issue using
// GET /rest/api/2/issue/{key}?fields=attachment endpoint.
func (c *Client) ListAttachments(key string) ([]Attachment, error) {
	res, err := c.GetV2(context.Background(), fmt.Sprintf("/issue/%s?fields=attachment", key), nil)
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

	var out struct {
		Fields struct {
			Attachment []Attachment `json:"attachment"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("jira: cannot decode attachments response: %w", err)
	}
	if out.Fields.Attachment == nil {
		return []Attachment{}, nil
	}
	return out.Fields.Attachment, nil
}

// GetAttachment fetches attachment metadata using GET /rest/api/2/attachment/{id} endpoint.
func (c *Client) GetAttachment(id string) (*Attachment, error) {
	res, err := c.GetV2(context.Background(), fmt.Sprintf("/attachment/%s", id), nil)
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

	var out Attachment
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("jira: cannot decode attachment response: %w", err)
	}
	return &out, nil
}

// DownloadAttachment downloads attachment content and returns raw bytes.
// It resolves the download URL from attachment metadata.
func (c *Client) DownloadAttachment(id string) ([]byte, string, error) {
	meta, err := c.GetAttachment(id)
	if err != nil {
		return nil, "", err
	}
	if meta.Content == "" {
		return nil, "", fmt.Errorf("jira: attachment %q has no download URL", id)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, meta.Content, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/octet-stream")
	c.setAuthHeader(req)

	httpClient := &http.Client{Transport: c.transport}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	if res == nil {
		return nil, "", ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", formatUnexpectedResponse(res)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("jira: cannot read attachment content: %w", err)
	}
	return data, attachmentFilename(meta), nil
}

// attachmentFilename resolves a safe filename for a downloaded attachment.
func attachmentFilename(meta *Attachment) string {
	name := filepath.Base(meta.Filename)
	if name == "" || name == "." || name == "/" {
		return fmt.Sprintf("attachment-%s", meta.ID)
	}
	if _, params, err := mime.ParseMediaType(fmt.Sprintf("attachment; filename=%q", name)); err == nil {
		if fn, ok := params["filename"]; ok && fn != "" {
			return filepath.Base(fn)
		}
	}
	return strings.ReplaceAll(name, "/", "_")
}

// DeleteAttachment deletes an attachment using DELETE /rest/api/2/attachment/{id} endpoint.
func (c *Client) DeleteAttachment(id string) error {
	res, err := c.DeleteV2(context.Background(), fmt.Sprintf("/attachment/%s", id), Header{
		attachmentNoCheckTokenHeader: "no-check",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return formatUnexpectedResponse(res)
	}
	return nil
}
