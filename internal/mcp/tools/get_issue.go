package tools

import (
	"context"
	"errors"

	"github.com/ankitpokhrel/jira-cli/api"
	issuefilter "github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

// GetIssueInput is the input schema for the get_issue tool.
type GetIssueInput struct {
	Key             string `json:"key" jsonschema:"issue key, e.g. \"PROJ-123\" (required)"`
	IncludeComments *bool  `json:"include_comments,omitempty" jsonschema:"include recent comments in the response (default true)"`
	CommentLimit    int    `json:"comment_limit,omitempty" jsonschema:"maximum number of recent comments to include (default 10)"`
}

// GetIssueOutput is the structured result of the get_issue tool.
type GetIssueOutput struct {
	Key         string         `json:"key"`
	Summary     string         `json:"summary"`
	Status      string         `json:"status"`
	Type        string         `json:"type"`
	Priority    string         `json:"priority"`
	Assignee    string         `json:"assignee"`
	Reporter    string         `json:"reporter"`
	Labels      []string       `json:"labels"`
	Components  []string       `json:"components"`
	FixVersions []string       `json:"fix_versions"`
	Parent      string         `json:"parent,omitempty"`
	Created     string         `json:"created"`
	Updated     string         `json:"updated"`
	Description string         `json:"description"`
	Comments    []CommentBrief `json:"comments,omitempty"`
	URL         string         `json:"url"`
}

// CommentBrief is a lean projection of an issue comment.
type CommentBrief struct {
	ID      string `json:"id"`
	Author  string `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
}

// GetIssue runs the get_issue tool.
func GetIssue(_ context.Context, d *Deps, in GetIssueInput) (GetIssueOutput, error) {
	if in.Key == "" {
		return GetIssueOutput{}, errors.New("key is required")
	}

	includeComments := true
	if in.IncludeComments != nil {
		includeComments = *in.IncludeComments
	}
	commentLimit := in.CommentLimit
	if commentLimit <= 0 {
		commentLimit = 10
	}

	iss, err := api.ProxyGetIssue(d.Client, in.Key, issuefilter.NewNumCommentsFilter(uint(commentLimit)))
	if err != nil {
		return GetIssueOutput{}, err
	}

	out := GetIssueOutput{
		Key:         iss.Key,
		Summary:     iss.Fields.Summary,
		Status:      iss.Fields.Status.Name,
		Type:        iss.Fields.IssueType.Name,
		Priority:    iss.Fields.Priority.Name,
		Assignee:    iss.Fields.Assignee.Name,
		Reporter:    iss.Fields.Reporter.Name,
		Labels:      iss.Fields.Labels,
		Created:     iss.Fields.Created,
		Updated:     iss.Fields.Updated,
		Description: bodyToMarkdown(iss.Fields.Description),
		URL:         d.IssueURL(iss.Key),
	}
	if iss.Fields.Parent != nil {
		out.Parent = iss.Fields.Parent.Key
	}
	for _, c := range iss.Fields.Components {
		out.Components = append(out.Components, c.Name)
	}
	for _, v := range iss.Fields.FixVersions {
		out.FixVersions = append(out.FixVersions, v.Name)
	}

	if includeComments && iss.Fields.Comment.Total > 0 {
		comments := iss.Fields.Comment.Comments
		// Take the last commentLimit comments (newest), preserving chronological order.
		start := 0
		if len(comments) > commentLimit {
			start = len(comments) - commentLimit
		}
		for i := start; i < len(comments); i++ {
			c := comments[i]
			out.Comments = append(out.Comments, CommentBrief{
				ID:      c.ID,
				Author:  c.Author.DisplayName,
				Body:    bodyToMarkdown(c.Body),
				Created: c.Created,
			})
		}
	}

	return out, nil
}
