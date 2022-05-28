package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

const separatorMinus = "-"

// EditResponse struct holds response from POST /issue endpoint.
type EditResponse struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// EditRequest struct holds request data for edit request.
// Setting an Assignee requires an account ID.
type EditRequest struct {
	IssueType      string
	ParentIssueKey string
	Summary        string
	Body           string
	Priority       string
	Labels         []string
	Components     []string
	FixVersions    []string
}

// Edit updates an issue using POST /issue endpoint.
func (c *Client) Edit(key string, req *EditRequest) error {
	data := getRequestDataForEdit(req)

	body, err := json.Marshal(&data)
	if err != nil {
		return err
	}

	res, err := c.PutV2(context.Background(), "/issue/"+key, body, Header{
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

type editFields struct {
	Summary []struct {
		Set string `json:"set,omitempty"`
	} `json:"summary,omitempty"`
	Description []struct {
		Set string `json:"set,omitempty"`
	} `json:"description,omitempty"`
	Priority []struct {
		Set struct {
			Name string `json:"name,omitempty"`
		} `json:"set,omitempty"`
	} `json:"priority,omitempty"`
	Labels []struct {
		Add    string `json:"add,omitempty"`
		Remove string `json:"remove,omitempty"`
	} `json:"labels,omitempty"`
	Components []struct {
		Add *struct {
			Name string `json:"name,omitempty"`
		} `json:"add,omitempty"`
		Remove *struct {
			Name string `json:"name,omitempty"`
		} `json:"remove,omitempty"`
	} `json:"components,omitempty"`
	FixVersions []struct {
		Set []struct {
			Name string `json:"name,omitempty"`
		} `json:"set,omitempty"`
	} `json:"fixVersions,omitempty"`
}

type editFieldsMarshaler struct {
	M editFields
}

// MarshalJSON is a custom marshaler to handle empty fields.
func (cfm *editFieldsMarshaler) MarshalJSON() ([]byte, error) {
	if len(cfm.M.Summary) == 0 || cfm.M.Summary[0].Set == "" {
		cfm.M.Summary = nil
	}
	if len(cfm.M.Description) == 0 || cfm.M.Description[0].Set == "" {
		cfm.M.Description = nil
	}
	if len(cfm.M.Priority) == 0 || cfm.M.Priority[0].Set.Name == "" {
		cfm.M.Priority = nil
	}
	if len(cfm.M.Components) == 0 || (cfm.M.Components[0].Add != nil && cfm.M.Components[0].Remove != nil) {
		cfm.M.Components = nil
	}
	if len(cfm.M.Labels) == 0 || (cfm.M.Labels[0].Add == "" && cfm.M.Labels[0].Remove == "") {
		cfm.M.Labels = nil
	}

	return json.Marshal(cfm.M)
}

type editRequest struct {
	Update editFieldsMarshaler `json:"update"`
	Fields struct {
		Parent *struct {
			Key string `json:"key,omitempty"`
			Set string `json:"set,omitempty"`
		} `json:"parent,omitempty"`
	} `json:"fields"`
}

func getRequestDataForEdit(req *EditRequest) *editRequest {
	if req.Labels == nil {
		req.Labels = []string{}
	}

	update := editFieldsMarshaler{editFields{
		Summary: []struct {
			Set string `json:"set,omitempty"`
		}{{Set: req.Summary}},
		Description: []struct {
			Set string `json:"set,omitempty"`
		}{{Set: req.Body}},
		Priority: []struct {
			Set struct {
				Name string `json:"name,omitempty"`
			} `json:"set,omitempty"`
		}{{Set: struct {
			Name string `json:"name,omitempty"`
		}{Name: req.Priority}}},
	}}

	if len(req.Labels) > 0 {
		add, sub := splitAddAndRemove(req.Labels)

		labels := make([]struct {
			Add    string `json:"add,omitempty"`
			Remove string `json:"remove,omitempty"`
		}, 0, len(req.Labels))

		for _, l := range sub {
			labels = append(labels, struct {
				Add    string `json:"add,omitempty"`
				Remove string `json:"remove,omitempty"`
			}{Remove: l})
		}
		for _, l := range add {
			labels = append(labels, struct {
				Add    string `json:"add,omitempty"`
				Remove string `json:"remove,omitempty"`
			}{Add: l})
		}

		update.M.Labels = labels
	}
	if len(req.Components) > 0 {
		add, sub := splitAddAndRemove(req.Components)

		cmp := make([]struct {
			Add *struct {
				Name string `json:"name,omitempty"`
			} `json:"add,omitempty"`
			Remove *struct {
				Name string `json:"name,omitempty"`
			} `json:"remove,omitempty"`
		}, 0, len(req.Components))

		for _, c := range sub {
			cmp = append(cmp, struct {
				Add *struct {
					Name string `json:"name,omitempty"`
				} `json:"add,omitempty"`
				Remove *struct {
					Name string `json:"name,omitempty"`
				} `json:"remove,omitempty"`
			}{Remove: &struct {
				Name string `json:"name,omitempty"`
			}{Name: c}})
		}
		for _, c := range add {
			cmp = append(cmp, struct {
				Add *struct {
					Name string `json:"name,omitempty"`
				} `json:"add,omitempty"`
				Remove *struct {
					Name string `json:"name,omitempty"`
				} `json:"remove,omitempty"`
			}{Add: &struct {
				Name string `json:"name,omitempty"`
			}{Name: c}})
		}

		update.M.Components = cmp
	}
	if len(req.FixVersions) > 0 {
		versions := make([]struct {
			Name string `json:"name,omitempty"`
		}, 0, len(req.FixVersions))

		for _, v := range req.FixVersions {
			versions = append(versions, struct {
				Name string `json:"name,omitempty"`
			}{Name: v})
		}
		update.M.FixVersions = []struct {
			Set []struct {
				Name string `json:"name,omitempty"`
			} `json:"set,omitempty"`
		}{{Set: versions}}
	}

	fields := struct {
		Parent *struct {
			Key string `json:"key,omitempty"`
			Set string `json:"set,omitempty"`
		} `json:"parent,omitempty"`
	}{
		Parent: &struct {
			Key string `json:"key,omitempty"`
			Set string `json:"set,omitempty"`
		}{},
	}
	if req.ParentIssueKey != "" {
		if req.ParentIssueKey == AssigneeNone {
			fields.Parent.Set = AssigneeNone
		} else {
			fields.Parent.Key = req.ParentIssueKey
		}
	}

	data := editRequest{
		Update: update,
		Fields: fields,
	}

	return &data
}

func splitAddAndRemove(input []string) ([]string, []string) {
	add := make([]string, 0, len(input))
	sub := make([]string, 0, len(input))

	for _, inp := range input {
		if strings.HasPrefix(inp, separatorMinus) {
			sub = append(sub, strings.TrimPrefix(inp, separatorMinus))
		}
	}
	for _, inp := range input {
		if !strings.HasPrefix(inp, separatorMinus) && !inArray(sub, inp) {
			add = append(add, inp)
		}
	}

	return add, sub
}

func inArray(array []string, item string) bool {
	for _, i := range array {
		if i == item {
			return true
		}
	}
	return false
}
