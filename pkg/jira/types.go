package jira

import "encoding/json"

// IssueTypeEpic is an epic issue type.
const IssueTypeEpic = "Epic"

// Project holds project info.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Lead struct {
		Name string `json:"displayName"`
	} `json:"lead"`
}

// Board holds board info.
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Issue holds issue info.
type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// IssueFields holds issue fields.
type IssueFields struct {
	Summary    string   `json:"summary"`
	Labels     []string `json:"labels"`
	Resolution struct {
		Name string `json:"name"`
	} `json:"resolution"`
	IssueType struct {
		Name string `json:"name"`
	} `json:"issueType"`
	Assignee struct {
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
	Created string `json:"created"`
	Updated string `json:"updated"`
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

// ADF is an Atlassian document format.
// See https://developer.atlassian.com/cloud/jira/platform/apis/document/structure/
type ADF struct {
	Version int       `json:"version"`
	DocType string    `json:"type"`
	Content []ADFNode `json:"content"`
}

// ADFNode is an ADF node.
type ADFNode struct {
	NodeType string         `json:"type"`
	Content  []ADFNodeValue `json:"content"`
}

// ADFNodeValue is an ADF node value.
type ADFNodeValue struct {
	ValueType string `json:"type"`
	Text      string `json:"text"`
}
