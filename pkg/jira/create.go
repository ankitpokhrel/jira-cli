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
	Name      string
	IssueType string
	Summary   string
	Body      string
	Priority  string
	Labels    []string
	// EpicFieldName is the dynamic epic field name
	// that changes per jira installation.
	EpicFieldName string
}

// Create creates an issue using POST /issue endpoint.
func (c *Client) Create(req *CreateRequest) (*CreateResponse, error) {
	data := c.getRequestData(req)

	body, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	res, err := c.Post(context.Background(), "/issue", body, Header{
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

type createFields struct {
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
	IssueType struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Name        string `json:"name,omitempty"`
	Summary     string `json:"summary"`
	Description *ADF   `json:"description,omitempty"`
	Priority    *struct {
		Name string `json:"name,omitempty"`
	} `json:"priority,omitempty"`
	Labels []string `json:"labels,omitempty"`

	epicFieldName string
}

type createFieldsMarshaler struct {
	M createFields
}

// MarshalJSON is a custom marshaler to handle dynamic field.
func (cfm createFieldsMarshaler) MarshalJSON() ([]byte, error) {
	m, err := json.Marshal(cfm.M)
	if err != nil || cfm.M.Name == "" || cfm.M.epicFieldName == "" {
		return m, err
	}

	var temp interface{}
	if err := json.Unmarshal(m, &temp); err != nil {
		return nil, err
	}
	dm := temp.(map[string]interface{})

	dm[cfm.M.epicFieldName] = dm["name"]
	delete(dm, "name")

	return json.Marshal(dm)
}

type createRequest struct {
	Update struct{}              `json:"update"`
	Fields createFieldsMarshaler `json:"fields"`
}

func (c *Client) getRequestData(req *CreateRequest) *createRequest {
	if req.Labels == nil {
		req.Labels = []string{}
	}

	data := createRequest{
		Update: struct{}{},
		Fields: createFieldsMarshaler{createFields{
			Project: struct {
				Key string `json:"key"`
			}{Key: req.Project},
			IssueType: struct {
				Name string `json:"name"`
			}{Name: req.IssueType},
			Name:          req.Name,
			Summary:       req.Summary,
			Labels:        req.Labels,
			epicFieldName: req.EpicFieldName,
		}},
	}

	if req.Priority != "" {
		data.Fields.M.Priority = &struct {
			Name string `json:"name,omitempty"`
		}{Name: req.Priority}
	}
	if req.Body != "" {
		data.Fields.M.Description = &ADF{
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
