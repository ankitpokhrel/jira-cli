package jira

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
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
	// ParentIssueKey is required when creating a sub-task for classic project.
	// This can also be used to attach epic for next-gen project.
	ParentIssueKey string
	Summary        string
	Body           interface{} // string in v1/v2 and adf.ADF in v3
	Priority       string
	Labels         []string
	Components     []string
	FixVersions    []string
	// EpicField is the dynamic epic field name
	// that changes per jira installation.
	EpicField string
	// SubtaskField is usually called "Sub-task" but is
	// case-sensitive in Jira and can differ slightly
	// in different Jira versions.
	SubtaskField string

	projectType string
}

// ForProjectType set jira project type.
func (cr *CreateRequest) ForProjectType(pt string) {
	cr.projectType = pt
}

// Create creates an issue using v3 version of the POST /issue endpoint.
func (c *Client) Create(req *CreateRequest) (*CreateResponse, error) {
	return c.create(req, apiVersion3)
}

// CreateV2 creates an issue using v2 version of the POST /issue endpoint.
func (c *Client) CreateV2(req *CreateRequest) (*CreateResponse, error) {
	return c.create(req, apiVersion2)
}

func (c *Client) create(req *CreateRequest, ver string) (*CreateResponse, error) {
	data := c.getRequestData(req)

	body, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	header := Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	var res *http.Response

	switch ver {
	case apiVersion2:
		res, err = c.PostV2(context.Background(), "/issue", body, header)
	default:
		res, err = c.Post(context.Background(), "/issue", body, header)
	}

	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return nil, formatUnexpectedResponse(res)
	}

	var out CreateResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

func (*Client) getRequestData(req *CreateRequest) *createRequest {
	if req.Labels == nil {
		req.Labels = []string{}
	}

	cf := createFields{
		Project: struct {
			Key string `json:"key"`
		}{Key: req.Project},
		IssueType: struct {
			Name string `json:"name"`
		}{Name: req.IssueType},
		Name:      req.Name,
		Summary:   req.Summary,
		Labels:    req.Labels,
		epicField: req.EpicField,
	}

	switch v := req.Body.(type) {
	case string:
		cf.Description = md.ToJiraMD(v)
	case *adf.ADF:
		cf.Description = v
	}

	data := createRequest{
		Update: struct{}{},
		Fields: createFieldsMarshaler{cf},
	}

	if req.ParentIssueKey != "" {
		subtaskField := IssueTypeSubTask
		if req.SubtaskField != "" {
			subtaskField = req.SubtaskField
		}

		if req.projectType == ProjectTypeNextGen || data.Fields.M.IssueType.Name == subtaskField {
			data.Fields.M.Parent = &struct {
				Key string `json:"key"`
			}{Key: req.ParentIssueKey}
		} else {
			data.Fields.M.Name = req.ParentIssueKey
		}
	}
	if req.Priority != "" {
		data.Fields.M.Priority = &struct {
			Name string `json:"name,omitempty"`
		}{Name: req.Priority}
	}
	if len(req.Components) > 0 {
		comps := make([]struct {
			Name string `json:"name,omitempty"`
		}, 0, len(req.Components))

		for _, c := range req.Components {
			comps = append(comps, struct {
				Name string `json:"name,omitempty"`
			}{c})
		}
		data.Fields.M.Components = comps
	}
	if len(req.FixVersions) > 0 {
		versions := make([]struct {
			Name string `json:"name,omitempty"`
		}, 0, len(req.FixVersions))

		for _, v := range req.FixVersions {
			versions = append(versions, struct {
				Name string `json:"name,omitempty"`
			}{v})
		}
		data.Fields.M.FixVersions = versions
	}

	return &data
}

type createRequest struct {
	Update struct{}              `json:"update"`
	Fields createFieldsMarshaler `json:"fields"`
}

type createFields struct {
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
	IssueType struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Parent *struct {
		Key string `json:"key"`
	} `json:"parent,omitempty"`
	Name        string      `json:"name,omitempty"`
	Summary     string      `json:"summary"`
	Description interface{} `json:"description,omitempty"`
	Priority    *struct {
		Name string `json:"name,omitempty"`
	} `json:"priority,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	Components []struct {
		Name string `json:"name,omitempty"`
	} `json:"components,omitempty"`
	FixVersions []struct {
		Name string `json:"name,omitempty"`
	} `json:"fixVersions,omitempty"`

	epicField string
}

type createFieldsMarshaler struct {
	M createFields
}

// MarshalJSON is a custom marshaler to handle dynamic field.
func (cfm createFieldsMarshaler) MarshalJSON() ([]byte, error) {
	m, err := json.Marshal(cfm.M)
	if err != nil {
		return m, err
	}

	var temp interface{}
	if err := json.Unmarshal(m, &temp); err != nil {
		return nil, err
	}
	dm := temp.(map[string]interface{})

	if cfm.M.epicField != "" {
		dm[cfm.M.epicField] = dm["name"]
	}
	delete(dm, "name")

	return json.Marshal(dm)
}
