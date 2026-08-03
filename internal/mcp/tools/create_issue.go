package tools

import (
	"context"
	"errors"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// CreateIssueInput is the input schema for the create_issue tool.
//
// v1 intentionally omits parent/epic linking: the underlying pkg/jira.CreateRequest
// routes that through project-type-aware fields (EpicField, SubtaskField) that the
// MCP layer doesn't currently resolve, so exposing it here would silently drop the
// link for classic projects. Add back when pkg/jira grows a first-class linker.
type CreateIssueInput struct {
	Summary     string   `json:"summary" jsonschema:"issue summary (required)"`
	Type        string   `json:"type" jsonschema:"issue type, e.g. \"Task\", \"Bug\", \"Story\" (required)"`
	Project     string   `json:"project,omitempty" jsonschema:"project key (defaults to the configured project)"`
	Description string   `json:"description,omitempty" jsonschema:"issue description in markdown"`
	Priority    string   `json:"priority,omitempty" jsonschema:"priority name, e.g. \"High\""`
	Labels      []string `json:"labels,omitempty"`
	Components  []string `json:"components,omitempty"`
	Assignee    string   `json:"assignee,omitempty" jsonschema:"assignee account id (Cloud) or username (Local)"`
}

// CreateIssueOutput is the structured result of the create_issue tool.
type CreateIssueOutput struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// CreateIssue runs the create_issue tool.
func CreateIssue(_ context.Context, d *Deps, in CreateIssueInput) (CreateIssueOutput, error) {
	if in.Summary == "" {
		return CreateIssueOutput{}, errors.New("summary is required")
	}
	if in.Type == "" {
		return CreateIssueOutput{}, errors.New("type is required")
	}

	project := d.ResolveProject(in.Project)
	if project == "" {
		return CreateIssueOutput{}, errors.New("project is required (no default project configured)")
	}

	req := &jira.CreateRequest{
		Project:    project,
		IssueType:  in.Type,
		Summary:    in.Summary,
		Body:       in.Description,
		Priority:   in.Priority,
		Labels:     in.Labels,
		Components: in.Components,
		Assignee:   in.Assignee,
	}
	req.ForInstallationType(d.Installation)

	resp, err := api.ProxyCreate(d.Client, req)
	if err != nil {
		return CreateIssueOutput{}, err
	}
	return CreateIssueOutput{Key: resp.Key, URL: d.IssueURL(resp.Key)}, nil
}
