package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
)

const (
	// AssigneeNone is a empty assignee.
	AssigneeNone = "none"
	// AssigneeDefault is a default assignee.
	AssigneeDefault = "default"
)

type assignRequest struct {
	AccountID *string `json:"accountId"`
}

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
		return nil, formatUnexpectedResponse(res)
	}

	var out Issue
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	out.Fields.Description = ifaceToADF(out.Fields.Description)

	return &out, nil
}

// AssignIssue assigns issue to the user using PUT /issue/{key}/assignee endpoint.
func (c *Client) AssignIssue(key, accountID string) error {
	aid := new(string)
	switch accountID {
	case AssigneeNone:
		*aid = "-1"
	case AssigneeDefault:
		aid = nil
	default:
		*aid = accountID
	}
	body, err := json.Marshal(assignRequest{AccountID: aid})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/assignee", key)
	res, err := c.Put(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
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

// GetIssueLinkTypes fetches issue link types using GET /issueLinkType endpoint.
func (c *Client) GetIssueLinkTypes() ([]*IssueLinkType, error) {
	res, err := c.Get(context.Background(), "/issueLinkType", nil)
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
		IssueLinkTypes []*IssueLinkType `json:"issueLinkTypes"`
	}

	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out.IssueLinkTypes, nil
}

type linkRequest struct {
	InwardIssue struct {
		Key string `json:"key"`
	} `json:"inwardIssue"`
	OutwardIssue struct {
		Key string `json:"key"`
	} `json:"outwardIssue"`
	LinkType struct {
		Name string `json:"name"`
	} `json:"type"`
}

// LinkIssue connects issues to the given link type using POST /issueLink endpoint.
func (c *Client) LinkIssue(inwardIssue, outwardIssue, linkType string) error {
	body, err := json.Marshal(linkRequest{
		InwardIssue: struct {
			Key string `json:"key"`
		}{Key: inwardIssue},
		OutwardIssue: struct {
			Key string `json:"key"`
		}{Key: outwardIssue},
		LinkType: struct {
			Name string `json:"name"`
		}{Name: linkType},
	})
	if err != nil {
		return err
	}

	res, err := c.Post(context.Background(), "/issueLink", body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return formatUnexpectedResponse(res)
	}
	return nil
}

type issueCommentRequest struct {
	Body string `json:"body"`
}

// AddIssueComment adds comment to an issue using POST /issue/{key}/comment endpoint.
// It only supports plain text comments at the moment.
func (c *Client) AddIssueComment(key, comment string) error {
	body, err := json.Marshal(&issueCommentRequest{Body: md.JiraToGithubFlavored(comment)})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/comment", key)
	res, err := c.PostV2(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}
	if res == nil {
		return ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return formatUnexpectedResponse(res)
	}
	return nil
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
