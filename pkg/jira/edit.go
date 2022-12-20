package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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
	// CustomFields holds all custom fields passed
	// while editing the issue.
	CustomFields map[string]string

	configuredCustomFields []IssueTypeField
}

// WithCustomFields sets valid custom fields for the issue.
func (er *EditRequest) WithCustomFields(cf []IssueTypeField) {
	er.configuredCustomFields = cf
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
		Add *struct {
			Name string `json:"name,omitempty"`
		} `json:"add,omitempty"`
		Remove *struct {
			Name string `json:"name,omitempty"`
		} `json:"remove,omitempty"`
	} `json:"fixVersions,omitempty"`

	customFields customField
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

	m, err := json.Marshal(cfm.M)
	if err != nil {
		return m, err
	}

	var temp interface{}
	if err := json.Unmarshal(m, &temp); err != nil {
		return nil, err
	}
	dm := temp.(map[string]interface{})

	for key, val := range cfm.M.customFields {
		dm[key] = val
	}

	return json.Marshal(dm)
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
		add, sub := splitAddAndRemove(req.FixVersions)

		versions := make([]struct {
			Add *struct {
				Name string `json:"name,omitempty"`
			} `json:"add,omitempty"`
			Remove *struct {
				Name string `json:"name,omitempty"`
			} `json:"remove,omitempty"`
		}, 0, len(req.FixVersions))

		for _, v := range sub {
			versions = append(versions, struct {
				Add *struct {
					Name string `json:"name,omitempty"`
				} `json:"add,omitempty"`
				Remove *struct {
					Name string `json:"name,omitempty"`
				} `json:"remove,omitempty"`
			}{Remove: &struct {
				Name string `json:"name,omitempty"`
			}{Name: v}})
		}
		for _, v := range add {
			versions = append(versions, struct {
				Add *struct {
					Name string `json:"name,omitempty"`
				} `json:"add,omitempty"`
				Remove *struct {
					Name string `json:"name,omitempty"`
				} `json:"remove,omitempty"`
			}{Add: &struct {
				Name string `json:"name,omitempty"`
			}{Name: v}})
		}

		update.M.FixVersions = versions
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
	constructCustomFieldsForEdit(req.CustomFields, req.configuredCustomFields, &data)

	return &data
}

func constructCustomFieldsForEdit(fields map[string]string, configuredFields []IssueTypeField, data *editRequest) {
	if len(fields) == 0 || len(configuredFields) == 0 {
		return
	}

	data.Update.M.customFields = make(customField)

	for key, val := range fields {
		for _, configured := range configuredFields {
			identifier := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(configured.Name)), " ", "-")
			if identifier != strings.ToLower(key) {
				continue
			}

			switch configured.Schema.DataType {
			case customFieldFormatOption:
				data.Update.M.customFields[configured.Key] = []customFieldTypeOptionSet{{Set: customFieldTypeOption{Value: val}}}
			case customFieldFormatProject:
				data.Update.M.customFields[configured.Key] = []customFieldTypeProjectSet{{Set: customFieldTypeProject{Value: val}}}
			case customFieldFormatArray:
				pieces := strings.Split(strings.TrimSpace(val), ",")
				if configured.Schema.Items == customFieldFormatOption {
					items := make([]customFieldTypeOptionAddRemove, 0)
					for _, p := range pieces {
						if strings.HasPrefix(p, separatorMinus) {
							items = append(items, customFieldTypeOptionAddRemove{Remove: &customFieldTypeOption{Value: strings.TrimPrefix(p, separatorMinus)}})
						} else {
							items = append(items, customFieldTypeOptionAddRemove{Add: &customFieldTypeOption{Value: p}})
						}
					}
					data.Update.M.customFields[configured.Key] = items
				} else {
					data.Update.M.customFields[configured.Key] = pieces
				}
			case customFieldFormatNumber:
				num, err := strconv.ParseFloat(val, 64) //nolint:gomnd
				if err != nil {
					// Let Jira API handle data type error for now.
					data.Update.M.customFields[configured.Key] = []customFieldTypeStringSet{{Set: val}}
				} else {
					data.Update.M.customFields[configured.Key] = []customFieldTypeNumberSet{{Set: customFieldTypeNumber(num)}}
				}
			default:
				data.Update.M.customFields[configured.Key] = []customFieldTypeStringSet{{Set: val}}
			}
		}
	}
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
