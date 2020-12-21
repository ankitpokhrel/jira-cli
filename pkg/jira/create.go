package jira

import (
	"context"
	"encoding/json"
	"net/http"
)

// CreateResponse struct holds response from POST /issue endpoint.
type CreateResponse struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// CreateRequest struct holds request data for create request.
type CreateRequest struct {
	Project   string
	IssueType string
	Summary   string
	Body      string
	Priority  string
	Labels    []string
}

type createFields struct {
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
	IssueType struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Summary     string `json:"summary"`
	Description *ADF   `json:"description,omitempty"`
	Priority    *struct {
		Name string `json:"name,omitempty"`
	} `json:"priority,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

type createRequest struct {
	Update struct{}     `json:"update"`
	Fields createFields `json:"fields"`
}

// Create creates an issue using POST /issue endpoint.
func (c *Client) Create(req *CreateRequest) (*CreateResponse, error) {
	data := c.getRequestData(req)

	b, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	res, err := c.Post(context.Background(), "/issue", b, Header{
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

	if res.StatusCode != http.StatusCreated {
		return nil, ErrUnexpectedStatusCode
	}

	var out CreateResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

func (c *Client) getRequestData(req *CreateRequest) *createRequest {
	if req.Labels == nil {
		req.Labels = []string{}
	}

	data := createRequest{
		Update: struct{}{},
		Fields: createFields{
			Project: struct {
				Key string `json:"key"`
			}{Key: req.Project},
			IssueType: struct {
				Name string `json:"name"`
			}{Name: req.IssueType},
			Summary: req.Summary,
			Labels:  req.Labels,
		},
	}

	if req.Priority != "" {
		data.Fields.Priority = &struct {
			Name string `json:"name,omitempty"`
		}{Name: req.Priority}
	}

	if req.Body != "" {
		data.Fields.Description = &ADF{
			Version: 1,
			DocType: "doc",
			Content: []ADFNode{
				{
					NodeType: "paragraph",
					Content:  []ADFNodeValue{{ValueType: "text", Text: req.Body}},
				},
			},
		}
	}

	return &data
}
