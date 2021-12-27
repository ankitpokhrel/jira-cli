package jira

import (
	"encoding/json"
)

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

// IssueType holds issue type info.
type IssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Handle  string `json:"untranslatedName,omitempty"` // This field doesn't exist in v2.
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
