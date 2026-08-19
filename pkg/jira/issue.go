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

	total := iss.Fields.Comment.Total
	limit := filter.Collection(opts).GetInt(issue.KeyIssueNumComments)
	if limit > total {
		limit = total
	}
	for i := total - 1; i >= total-limit; i-- {
		body := iss.Fields.Comment.Comments[i].Body
		iss.Fields.Comment.Comments[i].Body = ifaceToADF(body)
	}
	return iss, nil
}

// GetIssueV2 fetches issue details using v2 version of Jira GET /issue/{key} endpoint.
func (c *Client) GetIssueV2(key string, _ ...filter.Filter) (*Issue, error) {
	return c.getIssue(key, apiVersion2)
}

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

type issueCommentPropertyValue struct {
	Internal bool `json:"internal"`
}

type issueCommentProperty struct {
	Key   string                    `json:"key"`
	Value issueCommentPropertyValue `json:"value"`
}
type issueCommentRequest struct {
	Body       string                 `json:"body"`
	Properties []issueCommentProperty `json:"properties"`
}

type commentVisibility struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type issueCommentRequestV3 struct {
	Body       *adf.ADF           `json:"body"`
	Visibility *commentVisibility `json:"visibility,omitempty"`
}

// IssueCommentOptions configures AddIssueCommentOpts.
type IssueCommentOptions struct {
	Internal        bool
	VisibilityRole  string
	VisibilityGroup string
}

func (o IssueCommentOptions) visibility() *commentVisibility {
	role := strings.TrimSpace(o.VisibilityRole)
	group := strings.TrimSpace(o.VisibilityGroup)
	switch {
	case role != "":
		return &commentVisibility{Type: "role", Value: role}
	case group != "":
		return &commentVisibility{Type: "group", Value: group}
	default:
		return nil
	}
}

// Validate checks flag combinations for issue comments.
func (o IssueCommentOptions) Validate() error {
	role := strings.TrimSpace(o.VisibilityRole)
	group := strings.TrimSpace(o.VisibilityGroup)
	if role != "" && group != "" {
		return fmt.Errorf("jira: visibility role and visibility group are mutually exclusive")
	}
	if o.Internal && (role != "" || group != "") {
		return fmt.Errorf("jira: internal comments cannot be combined with role or group visibility")
	}
	return nil
}

// commentPlainToADF builds a minimal Atlassian Document body from plain text (paragraphs split on
// blank lines; single newlines become hard breaks). Used for REST v3 comments when setting visibility.
func commentPlainToADF(text string) *adf.ADF {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return &adf.ADF{
			Version: 1,
			DocType: "doc",
			Content: []*adf.Node{
				{NodeType: adf.NodeParagraph, Content: []*adf.Node{}},
			},
		}
	}
	blocks := strings.Split(text, "\n\n")
	var content []*adf.Node
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		inline := make([]*adf.Node, 0, len(lines)*2-1)
		for i, line := range lines {
			if i > 0 {
				inline = append(inline, &adf.Node{NodeType: adf.InlineNodeHardBreak})
			}
			inline = append(inline, &adf.Node{
				NodeType:  adf.ChildNodeText,
				NodeValue: adf.NodeValue{Text: line},
			})
		}
		content = append(content, &adf.Node{
			NodeType: adf.NodeParagraph,
			Content:  inline,
		})
	}
	if len(content) == 0 {
		content = []*adf.Node{{NodeType: adf.NodeParagraph, Content: []*adf.Node{}}}
	}
	return &adf.ADF{Version: 1, DocType: "doc", Content: content}
}

// AddIssueComment adds comment to an issue using POST /issue/{key}/comment endpoint.
func (c *Client) AddIssueComment(key, comment string, internal bool) error {
	return c.AddIssueCommentOpts(key, comment, IssueCommentOptions{Internal: internal})
}

// AddIssueCommentOpts adds a comment like AddIssueComment. When VisibilityRole or VisibilityGroup is set,
// it uses REST API v3 with an ADF body and a visibility restriction (mutually exclusive with Internal).
func (c *Client) AddIssueCommentOpts(key, comment string, opts IssueCommentOptions) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	vis := opts.visibility()
	if vis != nil {
		req := issueCommentRequestV3{
			Body:       commentPlainToADF(comment),
			Visibility: vis,
		}
		body, err := json.Marshal(&req)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("/issue/%s/comment", key)
		res, err := c.Post(context.Background(), path, body, Header{
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

	body, err := json.Marshal(&issueCommentRequest{Body: md.ToJiraMD(comment), Properties: []issueCommentProperty{{Key: "sd.public.comment", Value: issueCommentPropertyValue{Internal: opts.Internal}}}})
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
