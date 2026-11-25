package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
)

const (
	// IssueTypeEpic is an epic issue type.
	IssueTypeEpic = "Epic"
	// IssueTypeSubTask is a sub-task issue type.
	IssueTypeSubTask = "Sub-task"
	// AssigneeNone is an empty assignee.
	AssigneeNone = "none"
	// AssigneeDefault is a default assignee.
	AssigneeDefault = "default"
)

// GetIssue fetches issue details using GET /issue/{key} endpoint.
func (c *Client) GetIssue(key string, opts ...filter.Filter) (*Issue, error) {
	iss, err := c.getIssue(key, apiVersion3)
	if err != nil {
		return nil, err
	}

	iss.Fields.Description = ifaceToADF(iss.Fields.Description)

	available := len(iss.Fields.Comment.Comments)
	limit := filter.Collection(opts).GetInt(issue.KeyIssueNumComments)
	if limit > available {
		limit = available
	}
	if limit < 0 {
		limit = 0
	}
	for i := available - 1; i >= available-limit; i-- {
		body := iss.Fields.Comment.Comments[i].Body
		iss.Fields.Comment.Comments[i].Body = ifaceToADF(body)
	}
	return iss, nil
}

// GetIssueV2 fetches issue details using v2 version of Jira GET /issue/{key} endpoint.
func (c *Client) GetIssueV2(key string, _ ...filter.Filter) (*Issue, error) {
	return c.getIssue(key, apiVersion2)
}

// getIssue decodes version-specific JSON without normalizing rich-text fields;
// v3 callers must convert the description and any comments they intend to render.
func (c *Client) getIssue(key, ver string) (*Issue, error) {
	rawOut, err := c.getIssueRaw(key, ver)
	if err != nil {
		return nil, err
	}

	var iss Issue
	err = json.Unmarshal([]byte(rawOut), &iss)
	if err != nil {
		return nil, err
	}
	return &iss, nil
}

// GetIssueRaw fetches issue details same as GetIssue but returns the raw API response body string.
func (c *Client) GetIssueRaw(key string) (string, error) {
	return c.getIssueRaw(key, apiVersion3)
}

// GetIssueV2Raw fetches issue details same as GetIssueV2 but returns the raw API response body string.
func (c *Client) GetIssueV2Raw(key string) (string, error) {
	return c.getIssueRaw(key, apiVersion2)
}

// getIssueRaw accepts only a successful issue response and leaves its JSON intact
// for raw output or version-specific decoding; unknown versions use v3.
func (c *Client) getIssueRaw(key, ver string) (string, error) {
	path := fmt.Sprintf("/issue/%s", key)

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
		return "", err
	}
	if res == nil {
		return "", ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return "", formatUnexpectedResponse(res)
	}

	var b strings.Builder
	_, err = io.Copy(&b, res.Body)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// AssignIssue assigns issue to the user using v3 version of the PUT /issue/{key}/assignee endpoint.
func (c *Client) AssignIssue(key, assignee string) error {
	return c.assignIssue(key, assignee, apiVersion3)
}

// AssignIssueV2 assigns issue to the user using v2 version of the PUT /issue/{key}/assignee endpoint.
func (c *Client) AssignIssueV2(key, assignee string) error {
	return c.assignIssue(key, assignee, apiVersion2)
}

// assignIssue translates none/default sentinels and switches between Server user
// names and Cloud account IDs; only HTTP 204 confirms the assignment.
func (c *Client) assignIssue(key, assignee, ver string) error {
	path := fmt.Sprintf("/issue/%s/assignee", key)

	aid := new(string)
	switch assignee {
	case AssigneeNone:
		*aid = "-1"
	case AssigneeDefault:
		aid = nil
	default:
		*aid = assignee
	}

	var (
		res  *http.Response
		err  error
		body []byte
	)

	switch ver {
	case apiVersion2:
		type assignRequest struct {
			Name *string `json:"name"`
		}

		body, err = json.Marshal(assignRequest{Name: aid})
		if err != nil {
			return err
		}
		res, err = c.PutV2(context.Background(), path, body, Header{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
	default:
		type assignRequest struct {
			AccountID *string `json:"accountId"`
		}

		body, err = json.Marshal(assignRequest{AccountID: aid})
		if err != nil {
			return err
		}
		res, err = c.Put(context.Background(), path, body, Header{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		})
	}

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
	res, err := c.GetV2(context.Background(), "/issueLinkType", nil)
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

// linkRequest keeps inward and outward keys distinct because swapping them
// reverses the meaning of directional link types.
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

	res, err := c.PostV2(context.Background(), "/issueLink", body, Header{
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

// UnlinkIssue disconnects two issues using DELETE /issueLink/{linkId} endpoint.
func (c *Client) UnlinkIssue(linkID string) error {
	deleteLinkURL := fmt.Sprintf("/issueLink/%s", linkID)
	res, err := c.DeleteV2(context.Background(), deleteLinkURL, Header{
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

// GetLinkID gets linkID between two issues.
func (c *Client) GetLinkID(inwardIssue, outwardIssue string) (string, error) {
	i, err := c.GetIssueV2(inwardIssue)
	if err != nil {
		return "", err
	}

	for _, link := range i.Fields.IssueLinks {
		if link.InwardIssue != nil && link.InwardIssue.Key == outwardIssue {
			return link.ID, nil
		}

		if link.OutwardIssue != nil && link.OutwardIssue.Key == outwardIssue {
			return link.ID, nil
		}
	}
	return "", fmt.Errorf("no link found between provided issues")
}

// issueCommentPropertyValue expresses Service Management privacy explicitly:
// Internal false denotes a public comment rather than an omitted privacy choice.
type issueCommentPropertyValue struct {
	Internal bool `json:"internal"`
}

// issueCommentProperty associates privacy metadata with its Jira property key,
// independently of the comment body sent to the v2 endpoint.
type issueCommentProperty struct {
	Key   string                    `json:"key"`
	Value issueCommentPropertyValue `json:"value"`
}

// issueCommentRequest uses v2 Jira markup and explicit privacy properties for a
// top-level comment; native replies use a separate GraphQL mutation.
type issueCommentRequest struct {
	Body       string                 `json:"body"`
	Properties []issueCommentProperty `json:"properties"`
}

// AddIssueComment adds comment to an issue using POST /issue/{key}/comment endpoint.
func (c *Client) AddIssueComment(key, comment string, internal bool) error {
	body, err := json.Marshal(&issueCommentRequest{Body: md.ToJiraMD(comment), Properties: []issueCommentProperty{{Key: "sd.public.comment", Value: issueCommentPropertyValue{Internal: internal}}}})
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

// issueWorklogRequest leaves Started absent to let Jira choose the current time;
// Comment is already converted to Jira markup before encoding.
type issueWorklogRequest struct {
	Started   string `json:"started,omitempty"`
	TimeSpent string `json:"timeSpent"`
	Comment   string `json:"comment"`
}

// AddIssueWorklog adds worklog to an issue using POST /issue/{key}/worklog endpoint.
// Leave param `started` empty to use the server's current datetime as start date.
func (c *Client) AddIssueWorklog(key, started, timeSpent, comment, newEstimate string) error {
	worklogReq := issueWorklogRequest{
		TimeSpent: timeSpent,
		Comment:   md.ToJiraMD(comment),
	}
	if started != "" {
		worklogReq.Started = started
	}
	body, err := json.Marshal(&worklogReq)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/worklog", key)
	if newEstimate != "" {
		path = fmt.Sprintf("%s?adjustEstimate=new&newEstimate=%s", path, newEstimate)
	}
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

// GetField gets all fields configured for a Jira instance using GET /field endpiont.
func (c *Client) GetField() ([]*Field, error) {
	res, err := c.GetV2(context.Background(), "/field", Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
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

	var out []*Field

	err = json.NewDecoder(res.Body).Decode(&out)

	return out, err
}

// ifaceToADF normalizes generic JSON values through the ADF decoder, returning
// nil for absent or undecodable input. It does not validate document structure.
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

// remotelinkRequest identifies an external resource by URL and display title,
// unlike linkRequest, whose endpoints are both Jira issue keys.
type remotelinkRequest struct {
	RemoteObject struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	} `json:"object"`
}

// RemoteLinkIssue adds a remote link to an issue using POST /issue/{issueId}/remotelink endpoint.
func (c *Client) RemoteLinkIssue(issueID, title, url string) error {
	body, err := json.Marshal(remotelinkRequest{
		RemoteObject: struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		}{Title: title, URL: url},
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/issue/%s/remotelink", issueID)

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

// WatchIssue adds user as a watcher using v2 version of the POST /issue/{key}/watchers endpoint.
func (c *Client) WatchIssue(key, watcher string) error {
	return c.watchIssue(key, watcher, apiVersion3)
}

// WatchIssueV2 adds user as a watcher using using v2 version of the POST /issue/{key}/watchers endpoint.
func (c *Client) WatchIssueV2(key, watcher string) error {
	return c.watchIssue(key, watcher, apiVersion2)
}

// watchIssue sends the watcher as a JSON string, not an object, to the selected
// API version; only HTTP 204 confirms that Jira accepted the change.
func (c *Client) watchIssue(key, watcher, ver string) error {
	path := fmt.Sprintf("/issue/%s/watchers", key)

	var (
		res  *http.Response
		err  error
		body []byte
	)

	body, err = json.Marshal(watcher)
	if err != nil {
		return err
	}

	header := Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	switch ver {
	case apiVersion2:
		res, err = c.PostV2(context.Background(), path, body, header)
	default:
		res, err = c.Post(context.Background(), path, body, header)
	}

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
