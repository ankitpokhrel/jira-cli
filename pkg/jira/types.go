package jira

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
