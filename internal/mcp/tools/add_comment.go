package tools

import (
	"context"
	"errors"
)

// AddCommentInput is the input schema for the add_comment tool.
type AddCommentInput struct {
	Key      string `json:"key" jsonschema:"issue key, e.g. \"PROJ-123\" (required)"`
	Body     string `json:"body" jsonschema:"comment body in markdown (required)"`
	Internal bool   `json:"internal,omitempty" jsonschema:"mark as an internal (service-desk) comment"`
}

// AddCommentOutput is the structured result of the add_comment tool.
type AddCommentOutput struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// AddComment runs the add_comment tool.
func AddComment(_ context.Context, d *Deps, in AddCommentInput) (AddCommentOutput, error) {
	if in.Key == "" {
		return AddCommentOutput{}, errors.New("key is required")
	}
	if in.Body == "" {
		return AddCommentOutput{}, errors.New("body is required")
	}
	if err := d.Client.AddIssueComment(in.Key, in.Body, in.Internal); err != nil {
		return AddCommentOutput{}, err
	}
	return AddCommentOutput{Key: in.Key, URL: d.IssueURL(in.Key)}, nil
}
