package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// TransitionIssueInput is the input schema for the transition_issue tool.
//
// v1 intentionally omits assignee: pkg/jira.TransitionRequestFields.Assignee only
// supports the v2-style {"name": "..."} shape, which Jira Cloud ignores for
// account-id-style users. Reassignment should go through a dedicated tool (or wait
// until pkg/jira grows accountId-aware transition field support).
type TransitionIssueInput struct {
	Key        string `json:"key" jsonschema:"issue key (required)"`
	Transition string `json:"transition" jsonschema:"target transition name, e.g. \"In Progress\" (required, case-insensitive)"`
	Comment    string `json:"comment,omitempty" jsonschema:"optional comment to add as part of the transition (workflow must allow it)"`
	Resolution string `json:"resolution,omitempty" jsonschema:"optional resolution name to set, e.g. \"Fixed\""`
}

// TransitionIssueOutput is the structured result of the transition_issue tool.
type TransitionIssueOutput struct {
	Key      string `json:"key"`
	ToStatus string `json:"to_status"`
	URL      string `json:"url"`
}

// TransitionIssue runs the transition_issue tool.
func TransitionIssue(_ context.Context, d *Deps, in TransitionIssueInput) (TransitionIssueOutput, error) {
	if in.Key == "" {
		return TransitionIssueOutput{}, errors.New("key is required")
	}
	if in.Transition == "" {
		return TransitionIssueOutput{}, errors.New("transition is required")
	}

	transitions, err := api.ProxyTransitions(d.Client, in.Key)
	if err != nil {
		return TransitionIssueOutput{}, err
	}

	var match *jira.Transition
	target := strings.ToLower(strings.TrimSpace(in.Transition))
	available := make([]string, 0, len(transitions))
	for _, t := range transitions {
		available = append(available, t.Name)
		if strings.ToLower(t.Name) == target {
			match = t
		}
	}
	if match == nil {
		return TransitionIssueOutput{}, fmt.Errorf(
			"unknown transition %q for %s. Valid transitions: %s",
			in.Transition, in.Key, strings.Join(available, ", "),
		)
	}

	req := &jira.TransitionRequest{
		Transition: &jira.TransitionRequestData{
			ID:   match.ID.String(),
			Name: match.Name,
		},
	}
	if in.Comment != "" {
		req.Update = &jira.TransitionRequestUpdate{}
		req.Update.Comment = append(req.Update.Comment, struct {
			Add struct {
				Body string `json:"body"`
			} `json:"add"`
		}{
			Add: struct {
				Body string `json:"body"`
			}{Body: in.Comment},
		})
	}
	if in.Resolution != "" {
		req.Fields = &jira.TransitionRequestFields{
			Resolution: &struct {
				Name string `json:"name"`
			}{Name: in.Resolution},
		}
	}

	if _, err := d.Client.Transition(in.Key, req); err != nil {
		return TransitionIssueOutput{}, err
	}
	return TransitionIssueOutput{
		Key:      in.Key,
		ToStatus: match.Name,
		URL:      d.IssueURL(in.Key),
	}, nil
}
