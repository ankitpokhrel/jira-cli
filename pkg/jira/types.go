package jira

import (
	"encoding/json"
)

const (
	// AuthTypeBasic is a basic auth.
	AuthTypeBasic AuthType = "basic"
	// AuthTypeBearer is a bearer auth.
	AuthTypeBearer AuthType = "bearer"
)

// AuthType is a jira authentication type.
// Currently supports basic and bearer (PAT).
// Defaults to basic for empty or invalid value.
type AuthType string

// String implements stringer interface.
func (at AuthType) String() string {
	if at == "" {
		return string(AuthTypeBasic)
	}
	return string(at)
}

// Project holds project info.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Lead struct {
		Name string `json:"displayName"`
	} `json:"lead"`
	Type string `json:"style"`
}

// Board holds board info.
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Epic holds epic info.
type Epic struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

// Issue holds issue info.
type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// IssueFields holds issue fields.
type IssueFields struct {
	Summary     string      `json:"summary"`
	Description interface{} `json:"description"` // string in v1/v2, adf.ADF in v3
	Labels      []string    `json:"labels"`
	Resolution  struct {
		Name string `json:"name"`
	} `json:"resolution"`
	IssueType IssueType `json:"issueType"`
	Assignee  struct {
		Name string `json:"displayName"`
	} `json:"assignee"`
	Priority struct {
		Name string `json:"name"`
	} `json:"priority"`
	Reporter struct {
		Name string `json:"displayName"`
	} `json:"reporter"`
	Watches struct {
		IsWatching bool `json:"isWatching"`
		WatchCount int  `json:"watchCount"`
	} `json:"watches"`
	Status struct {
		Name string `json:"name"`
	} `json:"status"`
	Components []struct {
		Name string `json:"name"`
	} `json:"components"`
	Comment struct {
		Comments []struct {
			ID      string      `json:"id"`
			Author  User        `json:"author"`
			Body    interface{} `json:"body"` // string in v1/v2, adf.ADF in v3
			Created string      `json:"created"`
		} `json:"comments"`
		Total int `json:"total"`
	} `json:"comment"`
	IssueLinks []struct {
		LinkType struct {
			Name    string `json:"name"`
			Inward  string `json:"inward"`
			Outward string `json:"outward"`
		} `json:"type"`
		InwardIssue  *Issue `json:"inwardIssue,omitempty"`
		OutwardIssue *Issue `json:"outwardIssue,omitempty"`
	} `json:"issueLinks"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

// IssueTypeField holds issue field info.
type IssueTypeField struct {
	Name   string `json:"name"`
	Key    string `json:"key"`
	Schema struct {
		DataType string `json:"type"`
		Items    string `json:"items,omitempty"`
	} `json:"schema"`
	FieldID string `json:"fieldId,omitempty"`
}

// IssueType holds issue type info.
type IssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Handle  string `json:"untranslatedName,omitempty"` // This field may not exist in older version of the API.
	Subtask bool   `json:"subtask"`
}

// IssueLinkType holds issue link type info.
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// Sprint holds sprint info.
type Sprint struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"state"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	CompleteDate string `json:"completeDate,omitempty"`
	BoardID      int    `json:"originBoardId,omitempty"`
}

// Transition holds issue transition info.
type Transition struct {
	ID          json.Number `json:"id"`
	Name        string      `json:"name"`
	IsAvailable bool        `json:"isAvailable"`
}

// User holds user info.
type User struct {
	AccountID string `json:"accountId"`
	Email     string `json:"emailAddress"`
	Name      string `json:"displayName"`
	Active    bool   `json:"active"`
}
