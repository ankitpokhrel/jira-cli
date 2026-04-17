package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ankitpokhrel/jira-cli/api"
)

// SearchIssuesInput is the input schema for the search_issues tool.
type SearchIssuesInput struct {
	JQL      string `json:"jql,omitempty" jsonschema:"raw JQL to execute. Passed through verbatim unless project is also set, in which case the JQL is wrapped as 'project = X AND (your JQL)'. If you set project alongside JQL, your JQL must not contain its own ORDER BY clause."`
	Project  string `json:"project,omitempty" jsonschema:"project key (defaults to the configured project when JQL is omitted; when JQL is provided, only set this if you want the JQL scoped to a specific project)"`
	Status   string `json:"status,omitempty" jsonschema:"filter by status name, e.g. \"To Do\""`
	Assignee string `json:"assignee,omitempty" jsonschema:"filter by assignee. Use \"me\" for the configured user, \"none\" for unassigned, or a username/account id."`
	Limit    int    `json:"limit,omitempty" jsonschema:"maximum number of issues to return (default 50, clamped to 100)"`
}

// SearchIssuesOutput is the structured result of the search_issues tool.
type SearchIssuesOutput struct {
	// Returned is the number of issues in this response page. The Jira v3
	// /search/jql endpoint does not return a total match count; callers that
	// need to know whether more results exist should rerun with a larger Limit.
	Returned int          `json:"returned"`
	Issues   []IssueBrief `json:"issues"`
}

// IssueBrief is a lean projection of jira.Issue used for list-style outputs.
type IssueBrief struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Assignee string `json:"assignee"`
	Reporter string `json:"reporter"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
	URL      string `json:"url"`
}

// SearchIssues runs the search_issues tool.
func SearchIssues(_ context.Context, d *Deps, in SearchIssuesInput) (SearchIssuesOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	jql := strings.TrimSpace(in.JQL)
	project := d.ResolveProject(in.Project)

	if jql == "" {
		jql = composeJQL(project, in.Status, in.Assignee)
	} else if in.Project != "" {
		jql = fmt.Sprintf(`project = %q AND (%s)`, in.Project, jql)
	}

	res, err := api.ProxySearch(d.Client, jql, 0, uint(limit))
	if err != nil {
		return SearchIssuesOutput{}, err
	}

	out := SearchIssuesOutput{Issues: make([]IssueBrief, 0, len(res.Issues))}
	for _, iss := range res.Issues {
		out.Issues = append(out.Issues, IssueBrief{
			Key:      iss.Key,
			Summary:  iss.Fields.Summary,
			Status:   iss.Fields.Status.Name,
			Type:     iss.Fields.IssueType.Name,
			Priority: iss.Fields.Priority.Name,
			Assignee: iss.Fields.Assignee.Name,
			Reporter: iss.Fields.Reporter.Name,
			Created:  iss.Fields.Created,
			Updated:  iss.Fields.Updated,
			URL:      d.IssueURL(iss.Key),
		})
	}
	out.Returned = len(out.Issues)
	return out, nil
}

func composeJQL(project, status, assignee string) string {
	var clauses []string
	if project != "" {
		clauses = append(clauses, fmt.Sprintf(`project = %q`, project))
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf(`status = %q`, status))
	}
	switch strings.ToLower(assignee) {
	case "":
	case "me":
		clauses = append(clauses, "assignee = currentUser()")
	case "none", "x":
		clauses = append(clauses, "assignee is EMPTY")
	default:
		clauses = append(clauses, fmt.Sprintf(`assignee = %q`, assignee))
	}
	if len(clauses) == 0 {
		return ""
	}
	return strings.Join(clauses, " AND ") + " ORDER BY created DESC"
}
