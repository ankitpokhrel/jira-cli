package jira

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
	ID     string      `json:"id"`
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
		Name        string `json:"name"`
		Description string `json:"description"`
		IconURL     string `json:"iconUrl"`
	} `json:"issueType"`
	Assignee struct {
		Name string `json:"displayName"`
	} `json:"assignee"`
	Priority struct {
		Name    string `json:"name"`
		IconURL string `json:"iconUrl"`
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
