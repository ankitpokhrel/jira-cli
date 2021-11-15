package jira

import (
	"context"
	"encoding/json"
	"net/http"
)

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
	Assignee       string
	Priority       string
	Labels         []string
	Components     []string
}

// Edit updates an issue using POST /issue endpoint.
func (c *Client) Edit(key string, req *EditRequest) error {
	data := c.getRequestDataForEdit(req)

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
	Assignee []struct {
		Set struct {
			AccountID string `json:"accountId,omitempty"`
		} `json:"set,omitempty"`
	} `json:"assignee,omitempty"`
	Priority []struct {
		Set struct {
			Name string `json:"name,omitempty"`
		} `json:"set,omitempty"`
	} `json:"priority,omitempty"`
	Labels []struct {
		Set []string `json:"set,omitempty"`
	} `json:"labels,omitempty"`
	Components []struct {
		Set []struct {
			Name string `json:"name,omitempty"`
		} `json:"set,omitempty"`
	} `json:"components,omitempty"`
}

type editFieldsMarshaler struct {
	M editFields
}

// MarshalJSON is a custom marshaler to handle empty fields.
func (cfm editFieldsMarshaler) MarshalJSON() ([]byte, error) {
	if len(cfm.M.Summary) == 0 || cfm.M.Summary[0].Set == "" {
		cfm.M.Summary = nil
	}
	if len(cfm.M.Description) == 0 || cfm.M.Description[0].Set == "" {
		cfm.M.Description = nil
	}
	if len(cfm.M.Assignee) == 0 || cfm.M.Assignee[0].Set.AccountID == "" {
		cfm.M.Assignee = nil
	}
	if len(cfm.M.Priority) == 0 || cfm.M.Priority[0].Set.Name == "" {
		cfm.M.Priority = nil
	}
	if len(cfm.M.Components) == 0 || len(cfm.M.Components[0].Set) == 0 {
		cfm.M.Components = nil
	}
	if len(cfm.M.Labels) == 0 || len(cfm.M.Labels[0].Set) == 0 {
		cfm.M.Labels = nil
	}

	return json.Marshal(cfm.M)
}

type editRequest struct {
	Update editFieldsMarshaler `json:"update"`
	Fields struct {
		Parent *struct {
			Key string `json:"key"`
		} `json:"parent,omitempty"`
	} `json:"fields"`
}

func (c *Client) getRequestDataForEdit(req *EditRequest) *editRequest {
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
		Assignee: []struct {
			Set struct {
				AccountID string `json:"accountId,omitempty"`
			} `json:"set,omitempty"`
		}{{Set: struct {
			AccountID string `json:"accountId,omitempty"`
		}{AccountID: req.Assignee}}},
		Priority: []struct {
			Set struct {
				Name string `json:"name,omitempty"`
			} `json:"set,omitempty"`
		}{{Set: struct {
			Name string `json:"name,omitempty"`
		}{Name: req.Priority}}},
		Labels: []struct {
			Set []string `json:"set,omitempty"`
		}{{Set: req.Labels}},
	}}

	if len(req.Components) > 0 {
		cmp := make([]struct {
			Name string `json:"name,omitempty"`
		}, 0, len(req.Components))

		for _, c := range req.Components {
			cmp = append(cmp, struct {
				Name string `json:"name,omitempty"`
			}{Name: c})
		}

		update.M.Components = []struct {
			Set []struct {
				Name string `json:"name,omitempty"`
			} `json:"set,omitempty"`
		}{{Set: cmp}}
	}

	fields := struct {
		Parent *struct {
			Key string `json:"key"`
		} `json:"parent,omitempty"`
	}{}
	if req.ParentIssueKey != "" {
		fields = struct {
			Parent *struct {
				Key string `json:"key"`
			} `json:"parent,omitempty"`
		}{Parent: &struct {
			Key string `json:"key"`
		}{Key: req.ParentIssueKey}}
	}

	data := editRequest{
		Update: update,
		Fields: fields,
	}

	return &data
}
