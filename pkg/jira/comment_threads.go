package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	// These UI-internal persisted operations are versioned together. They are not
	// the public GraphQL schema and may stop working when Jira changes its UI.
	threadRootsRouteV1   = "/gateway/api/graphql/pq/e7440492f5fce0c34a11de0fab72b7389a9e73932eac35b1c9cff9b79c3c9894"
	threadRepliesRouteV1 = "/gateway/api/graphql/pq/9fb5324cd9a9667b60957bc4f961d754452096fb9ecf43bb6fca81007452f8bc"
	threadRootsQueryV1   = "CommentsViewRelayAdapter_RefetchQuery"
	threadRepliesQueryV1 = "CommentRepliesListRelay_RefetchQuery"
	threadMentionVarV1   = "__relay_internal__pv__jiraplatform__graphqlrelayprovidersrcPlatformEditorMentionNodeGraphqlProvider"
	threadCommentsVarV1  = "__relay_internal__pv__jiraplatform__graphqlrelayprovidersrcjivRelayCommentsRelayProvider"
	threadResponseLimit  = 16 << 20
	threadRecordLimit    = 10000
	threadRequestLimit   = 1000
	threadTotalTimeout   = 2 * time.Minute
	threadHTTPTimeout    = 30 * time.Second
	threadRootPageSize   = 50
	threadReplyPageSize  = 100
	threadTargetWindow   = 5
)

var (
	threadIssueKey = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*-[0-9]+$`)
	threadCloudID  = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)
	threadIssueID  = regexp.MustCompile(`^[0-9]+$`)
)

// GetIssueCommentThreads reads all visible Cloud comments with verified native
// parentage and REST v3 ADF bodies. It is experimental and supports Basic API-token
// authentication only. It never falls back to inferred or flat relationships.
func (c *Client) GetIssueCommentThreads(ctx context.Context, key string) ([]IssueComment, error) {
	issue, err := c.getIssueWithCommentThreads(ctx, key, true)
	if err != nil {
		return nil, fmt.Errorf("jira: native comment threads: %w", err)
	}
	return issue.Fields.Comment.Comments, nil
}

// GetIssueWithCommentThreads reads a Cloud issue over HTTPS without redirects,
// including the initial issue fetch in the experimental adapter's safety bounds.
// When includeComments is false, it preserves the embedded comments and total
// without fetching native relationships or additional comment pages.
func (c *Client) GetIssueWithCommentThreads(ctx context.Context, key string, includeComments bool) (*Issue, error) {
	issue, err := c.getIssueWithCommentThreads(ctx, key, includeComments)
	if err != nil {
		return nil, fmt.Errorf("jira: issue with native comment threads: %w", err)
	}
	return issue, nil
}

// getIssueWithCommentThreads keeps base details and optional thread retrieval
// within one bounded HTTPS session, replacing embedded comments only on success.
func (c *Client) getIssueWithCommentThreads(ctx context.Context, key string, includeComments bool) (*Issue, error) {
	if !threadIssueKey.MatchString(key) {
		return nil, errors.New("invalid issue key")
	}
	r, err := c.newThreadReader()
	if err != nil {
		return nil, err
	}
	totalTimeout := threadTotalTimeout
	if c.timeout > 0 {
		totalTimeout = min(totalTimeout, c.timeout)
	}
	ctx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()
	body, err := r.get(ctx, baseURLv3+"/issue/"+key, false)
	if err != nil {
		return nil, fmt.Errorf("read base issue: %w", err)
	}
	var response struct {
		ID     string       `json:"id"`
		Key    string       `json:"key"`
		Fields *IssueFields `json:"fields"`
	}
	if err := threadDecode(body, &response); err != nil {
		return nil, fmt.Errorf("decode base issue: %w", err)
	}
	if !threadIssueID.MatchString(response.ID) || !strings.EqualFold(response.Key, key) || response.Fields == nil {
		return nil, errors.New("missing or inconsistent issue identity or fields")
	}
	description := ifaceToADF(response.Fields.Description)
	if response.Fields.Description != nil && (description == nil || description.DocType != "doc" || description.Version != 1 || description.Content == nil) {
		return nil, errors.New("invalid issue description ADF")
	}
	response.Fields.Description = description
	if response.Fields.Comment.Total < 0 || len(response.Fields.Comment.Comments) > response.Fields.Comment.Total {
		return nil, errors.New("invalid embedded comment count")
	}
	issue := &Issue{Key: response.Key, Fields: *response.Fields}
	if !includeComments {
		return issue, nil
	}
	issueID, err := r.bootstrap(ctx, key, response.ID)
	if err != nil {
		return nil, fmt.Errorf("bootstrap issue identity: %w", err)
	}
	comments, err := r.commentThreads(ctx, issueID, key)
	if err != nil {
		return nil, err
	}
	issue.Fields.Comment.Comments = comments
	issue.Fields.Comment.Total = len(comments)
	return issue, nil
}

// commentThreads requires all native and REST pages with matching identities
// before assigning ParentID; partial coverage cannot distinguish a root from a reply.
func (r *threadReader) commentThreads(ctx context.Context, issueID, key string) ([]IssueComment, error) {
	parents, err := r.parents(ctx, issueID, key)
	if err != nil {
		return nil, fmt.Errorf("read native relationships: %w", err)
	}
	comments, err := r.comments(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read REST comments: %w", err)
	}
	if len(comments) != len(parents) {
		return nil, errors.New("native and REST comment coverage differs; collection may have changed")
	}
	for i := range comments {
		parent, ok := parents[comments[i].ID]
		if !ok {
			return nil, errors.New("REST comment has no verified native relationship")
		}
		comments[i].ParentID = parent.parentID
	}
	return comments, nil
}

// threadReader shares one request budget across bootstrap, pagination, and reply
// confirmation while keeping credentials confined to the configured HTTPS origin.
type threadReader struct {
	client               *http.Client
	origin, login, token string
	requests             int
}

// newThreadReader rejects unsupported authentication and non-origin URLs before
// I/O, and disables redirects so configured credentials cannot follow a new target.
func (c *Client) newThreadReader() (*threadReader, error) {
	if c.authType != nil && c.authType.String() != string(AuthTypeBasic) {
		return nil, errors.New("only Basic API-token authentication is supported")
	}
	if c.login == "" || c.token == "" {
		return nil, errors.New("basic API-token credentials are required")
	}
	origin, err := url.Parse(c.server)
	if err != nil || origin == nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" || (origin.EscapedPath() != "" && origin.EscapedPath() != "/") || origin.Scheme != "https" {
		return nil, errors.New("an HTTPS Jira Cloud origin without a path or user information is required")
	}
	requestTimeout := threadHTTPTimeout
	if c.timeout > 0 {
		requestTimeout = min(requestTimeout, c.timeout)
	}
	return &threadReader{
		client: &http.Client{Transport: c.transport, Timeout: requestTimeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}},
		origin: strings.TrimSuffix(c.server, "/"), login: c.login, token: c.token,
	}, nil
}

// Keep the cause available to errors.Is without exposing URLs, query variables,
// proxy credentials, or arbitrary transport error text to the terminal.
type threadRequestError struct{ cause error }

// Error redacts transport details that may contain credentials or private queries.
func (e threadRequestError) Error() string { return "HTTP request failed" }

// Unwrap preserves cancellation and transport error matching despite redaction.
func (e threadRequestError) Unwrap() error { return e.cause }

// threadHTTPError exposes only the status, not a potentially private response body.
type threadHTTPError struct{ status int }

// Error reports the status without including response content.
func (e threadHTTPError) Error() string { return fmt.Sprintf("unexpected HTTP status %d", e.status) }

// get shares bounded transport handling between JSON endpoints and the HTML
// bootstrap page; read-only callers do not need the mutation-attempt indicator.
func (r *threadReader) get(ctx context.Context, path string, page bool) ([]byte, error) {
	body, _, err := r.request(ctx, http.MethodGet, path, nil, page)
	return body, err
}

// request bounds requests and response sizes, returning whether transport was
// attempted so mutation callers can distinguish rejection from uncertain outcomes.
// POST bodies are not replayable, even by the HTTP transport.
func (r *threadReader) request(ctx context.Context, method, path string, payload []byte, page bool) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, fmt.Errorf("before request: %w", err)
	}
	r.requests++
	if r.requests > threadRequestLimit {
		return nil, false, errors.New("request limit exceeded")
	}
	req, err := http.NewRequestWithContext(ctx, method, r.origin+path, bytes.NewReader(payload))
	if err != nil {
		return nil, false, errors.New("could not construct request")
	}
	// A reply must not be replayed by the HTTP transport, even on a reused connection.
	if method == http.MethodPost {
		req.GetBody = nil
	}
	req.SetBasicAuth(r.login, r.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/graphql-response+json, application/json, multipart/mixed")
	if page {
		req.Header.Set("Accept", "text/html")
	}
	res, err := r.client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("send request: %w", threadRequestError{cause: err})
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, true, threadHTTPError{status: res.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, threadResponseLimit+1))
	if err != nil {
		return nil, true, fmt.Errorf("read response: %w", threadRequestError{cause: err})
	}
	if len(body) > threadResponseLimit {
		return nil, true, errors.New("response size limit exceeded")
	}
	return body, true, nil
}

// threadDecode redacts JSON decoding details that may include private Jira values.
func threadDecode(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		// JSON type errors may contain private response values.
		return errors.New("invalid JSON response")
	}
	return nil
}

// bootstrap combines validated Cloud metadata with the REST issue ID to build
// the native issue identity, rejecting missing or conflicting tenant metadata.
func (r *threadReader) bootstrap(ctx context.Context, key, issueID string) (string, error) {
	body, err := r.get(ctx, "/browse/"+key, true)
	if err != nil {
		return "", fmt.Errorf("read issue page metadata: %w", err)
	}
	var cloudID string
	tokens := html.NewTokenizer(bytes.NewReader(body))
	for {
		token := tokens.Next()
		if token == html.ErrorToken {
			if !errors.Is(tokens.Err(), io.EOF) {
				return "", errors.New("invalid issue page metadata")
			}
			break
		}
		if token != html.StartTagToken && token != html.SelfClosingTagToken {
			continue
		}
		t := tokens.Token()
		if t.Data != "meta" {
			continue
		}
		var name, content string
		for _, attr := range t.Attr {
			switch attr.Key {
			case "name":
				name = attr.Val
			case "content":
				content = attr.Val
			}
		}
		if name == "ajs-cloud-id" {
			if !threadCloudID.MatchString(content) || (cloudID != "" && cloudID != content) {
				return "", errors.New("invalid or conflicting Cloud identity metadata")
			}
			cloudID = content
		}
	}
	if cloudID == "" {
		return "", errors.New("issue page lacks Cloud identity metadata")
	}
	return "ari:cloud:jira:" + cloudID + ":issue/" + issueID, nil
}

// threadEdge retains both cursor and identity for pagination consistency checks.
type threadEdge struct {
	Cursor string      `json:"cursor"`
	Node   *threadNode `json:"node"`
}

// threadPageInfo uses pointer flags to distinguish omitted coverage metadata
// from an explicit end of pagination.
type threadPageInfo struct {
	HasNextPage     *bool  `json:"hasNextPage"`
	HasPreviousPage *bool  `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
}

// threadConnection keeps records and coverage metadata together; neither may
// be omitted even for an empty native collection.
type threadConnection struct {
	Edges    []threadEdge    `json:"edges"`
	PageInfo *threadPageInfo `json:"pageInfo"`
}

// threadNode bridges native GraphQL identities and REST comment IDs, retaining
// both reply connections and child aliases so their linkage can be cross-checked.
type threadNode struct {
	ID                string            `json:"id"`
	Key               string            `json:"key"`
	CommentID         string            `json:"commentId"`
	CommentsPaginated *threadConnection `json:"commentsPaginated"`
	RepliesPaginated  *threadConnection `json:"repliesPaginated"`
	ChildComments     *threadConnection `json:"childComments"`
}

// threadVariablesV1 keeps feature-provider inputs aligned with the versioned
// persisted operations instead of relying on changing server-side defaults.
func threadVariablesV1(id string) map[string]any {
	return map[string]any{
		"id": id, "issueViewSlackSyncInfoEnabled": true, "shouldFetchAgentAttribution": false,
		threadMentionVarV1: true, threadCommentsVarV1: false,
	}
}

// query accepts only a complete GraphQL result for the requested native identity;
// partial data with errors cannot establish trustworthy parentage.
func (r *threadReader) query(ctx context.Context, route, operation string, variables map[string]any) (*threadNode, error) {
	encoded, err := json.Marshal(variables)
	if err != nil {
		return nil, errors.New("could not encode query variables")
	}
	query := url.Values{"operation": {operation}, "variables": {string(encoded)}}
	body, err := r.get(ctx, route+"?"+query.Encode(), false)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data struct {
			Node *threadNode `json:"node"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if err := threadDecode(body, &response); err != nil {
		return nil, err
	}
	if len(response.Errors) != 0 {
		return nil, errors.New("GraphQL returned errors (details redacted)")
	}
	if response.Data.Node == nil || response.Data.Node.ID != variables["id"] {
		return nil, errors.New("missing or inconsistent GraphQL node identity")
	}
	return response.Data.Node, nil
}

// threadPagination tracks forward progress and every seen cursor to reject
// repeated pages or cycles instead of accepting an incomplete collection.
type threadPagination struct {
	after string
	seen  map[string]bool
}

// advance requires explicit coverage from the first page onward and consistent,
// unique cursors before permitting another request.
func (p *threadPagination) advance(conn *threadConnection) (bool, error) {
	if conn == nil || conn.Edges == nil || conn.PageInfo == nil || conn.PageInfo.HasNextPage == nil || conn.PageInfo.HasPreviousPage == nil {
		return false, errors.New("missing connection or pagination metadata")
	}
	info := conn.PageInfo
	// Start at the beginning, then advance only forward. A previous page on
	// the initial query would leave unknown roots or replies behind.
	if *info.HasPreviousPage != (p.after != "") {
		return false, errors.New("incomplete pagination direction coverage")
	}
	if len(conn.Edges) == 0 {
		if p.after != "" || *info.HasNextPage || info.StartCursor != "" || info.EndCursor != "" {
			return false, errors.New("empty or nonprogressive pagination page")
		}
		return false, nil
	}
	if info.StartCursor != conn.Edges[0].Cursor || info.EndCursor != conn.Edges[len(conn.Edges)-1].Cursor {
		return false, errors.New("inconsistent page cursors")
	}
	if p.seen == nil {
		p.seen = make(map[string]bool)
	}
	for _, edge := range conn.Edges {
		if edge.Cursor == "" || p.seen[edge.Cursor] {
			return false, errors.New("missing, duplicate, or cyclic pagination cursor")
		}
		p.seen[edge.Cursor] = true
	}
	p.after = info.EndCursor
	return *info.HasNextPage, nil
}

// threadLink maps a REST comment to its verified parent and native mutation
// identity; an empty parentID denotes a root, not an unknown relationship.
type threadLink struct {
	parentID string
	nodeID   string
}

// parents exhausts native root and reply pages, enforcing uniqueness across both
// REST and GraphQL IDs so duplicates or cycles cannot masquerade as parentage.
func (r *threadReader) parents(ctx context.Context, issueID, key string) (map[string]threadLink, error) {
	parents := make(map[string]threadLink)
	globalIDs := make(map[string]bool)
	add := func(node *threadNode, parent string) error {
		if node == nil || node.ID == "" || node.CommentID == "" {
			return errors.New("missing comment identity or native linkage")
		}
		if _, exists := parents[node.CommentID]; exists || globalIDs[node.ID] {
			return errors.New("duplicate comment identity, conflicting parentage, or cycle")
		}
		if len(parents) >= threadRecordLimit {
			return errors.New("comment record limit exceeded")
		}
		parents[node.CommentID] = threadLink{parentID: parent, nodeID: node.ID}
		globalIDs[node.ID] = true
		return nil
	}
	var roots threadPagination
	for {
		variables := threadVariablesV1(issueID)
		for k, v := range map[string]any{
			"first": threadRootPageSize, "last": nil, "after": nil, "before": nil,
			"afterTarget": threadTargetWindow, "beforeTarget": threadTargetWindow, "commentTargetId": nil,
			"commentsSortBy":     map[string]string{"field": "CREATED", "order": "DESC"},
			"hasCommentTargetId": false, "skipComments": false,
			"jivRelayCommentsEnabled": false, "scrubberEnabled": false,
		} {
			variables[k] = v
		}
		if roots.after != "" {
			variables["after"] = roots.after
		}
		node, err := r.query(ctx, threadRootsRouteV1, threadRootsQueryV1, variables)
		if err != nil {
			return nil, fmt.Errorf("query roots: %w", err)
		}
		if !strings.EqualFold(node.Key, key) {
			return nil, errors.New("root issue key mismatch")
		}
		more, err := roots.advance(node.CommentsPaginated)
		if err != nil {
			return nil, fmt.Errorf("paginate roots: %w", err)
		}
		for _, edge := range node.CommentsPaginated.Edges {
			root := edge.Node
			if err := add(root, ""); err != nil {
				return nil, err
			}
			if err := r.replies(ctx, root, add); err != nil {
				return nil, fmt.Errorf("read replies: %w", err)
			}
		}
		if !more {
			return parents, nil
		}
	}
}

// replies exhausts a root's reply connection and checks child aliases against
// verified replies; aliases alone are not sufficient evidence of membership.
func (r *threadReader) replies(ctx context.Context, root *threadNode, add func(*threadNode, string) error) error {
	conn := root.RepliesPaginated
	children := make(map[string]string)
	var pages threadPagination
	for {
		more, err := pages.advance(conn)
		if err != nil {
			return err
		}
		for _, edge := range conn.Edges {
			if err := add(edge.Node, root.CommentID); err != nil {
				return err
			}
			children[edge.Node.CommentID] = edge.Node.ID
		}
		if !more {
			break
		}
		variables := threadVariablesV1(root.ID)
		variables["after"], variables["first"] = pages.after, threadReplyPageSize
		node, err := r.query(ctx, threadRepliesRouteV1, threadRepliesQueryV1, variables)
		if err != nil {
			return err
		}
		conn = node.RepliesPaginated
	}
	if root.ChildComments != nil {
		for _, edge := range root.ChildComments.Edges {
			if edge.Node == nil || edge.Node.CommentID == "" || edge.Node.ID == "" || children[edge.Node.CommentID] != edge.Node.ID {
				return errors.New("child alias has no matching verified reply")
			}
		}
	}
	return nil
}

// threadRESTPage distinguishes missing offsets and totals from valid zero values
// so an absent page cannot be mistaken for an empty comment collection.
type threadRESTPage struct {
	StartAt  *int           `json:"startAt"`
	Total    *int           `json:"total"`
	Comments []IssueComment `json:"comments"`
}

// validate requires contiguous progress and a stable bounded total, rejecting
// collection changes rather than returning an apparently complete partial page set.
func (p threadRESTPage) validate(offset, total int) error {
	if p.StartAt == nil || *p.StartAt != offset || p.Total == nil || *p.Total < 0 || *p.Total > threadRecordLimit || p.Comments == nil {
		return errors.New("invalid REST pagination metadata or record limit exceeded")
	}
	if total != -1 && total != *p.Total {
		return errors.New("REST comment total changed during pagination")
	}
	if offset+len(p.Comments) > *p.Total || (len(p.Comments) == 0 && offset < *p.Total) {
		return errors.New("incomplete or nonprogressive REST pagination")
	}
	return nil
}

// comments loads every REST page with unique IDs and valid ADF bodies, leaving
// native ParentID assignment to the subsequent full-coverage reconciliation.
func (r *threadReader) comments(ctx context.Context, key string) ([]IssueComment, error) {
	comments := make([]IssueComment, 0)
	seen := make(map[string]bool)
	total := -1
	for {
		path := baseURLv3 + "/issue/" + key + "/comment?maxResults=100&startAt=" + strconv.Itoa(len(comments))
		body, err := r.get(ctx, path, false)
		if err != nil {
			return nil, err
		}
		var page threadRESTPage
		if err := threadDecode(body, &page); err != nil {
			return nil, err
		}
		if err := page.validate(len(comments), total); err != nil {
			return nil, err
		}
		total = *page.Total
		for _, comment := range page.Comments {
			if comment.ID == "" || seen[comment.ID] {
				return nil, errors.New("missing or duplicate REST comment identity")
			}
			doc := ifaceToADF(comment.Body)
			if doc == nil || doc.DocType != "doc" || doc.Version != 1 || doc.Content == nil || comment.Created == "" {
				return nil, errors.New("missing or invalid REST comment body or creation metadata")
			}
			comment.Body = doc
			comments = append(comments, comment)
			seen[comment.ID] = true
		}
		if len(comments) == total {
			return comments, nil
		}
	}
}
