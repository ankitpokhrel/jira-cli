package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Sprint states.
const (
	SprintStateActive = "active"
	SprintStateClosed = "closed"
	SprintStateFuture = "future"
)

// SprintResult holds response from /board/{boardID}/sprint endpoint.
type SprintResult struct {
	MaxResults int       `json:"maxResults"`
	StartAt    int       `json:"startAt"`
	IsLast     bool      `json:"isLast"`
	Sprints    []*Sprint `json:"values"`
}

// Sprints fetches all sprints for a given board.
//
// qp is an additional query parameters in key, value pair format, eg: state=closed.
func (c *Client) Sprints(boardID int, qp string, startAt, max int) (*SprintResult, error) {
	res, err := c.GetV1(
		context.Background(),
		fmt.Sprintf("/board/%d/sprint?%s&startAt=%d&maxResults=%d", boardID, qp, startAt, max),
		nil,
	)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out SprintResult

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

// SprintsInBoards fetches sprints across given board IDs.
//
// qp is an additional query parameters in key, value pair format, eg: state=closed.
func (c *Client) SprintsInBoards(boardIDs []int, qp string, limit int) []*Sprint {
	n := len(boardIDs)
	ch := make(chan []*Sprint, n)

	for _, boardID := range boardIDs {
		go func(id int) {
			s, err := c.lastNSprints(id, qp, limit)
			if err != nil {
				ch <- nil
				return
			}

			injectBoardID(s.Sprints, id)

			ch <- s.Sprints
		}(boardID)
	}

	var sprints []*Sprint

	seen := make(map[int]struct{}, n)

	for i := 0; i < n; i++ {
		v := <-ch

		for _, s := range v {
			if _, ok := seen[s.ID]; ok {
				continue
			}
			sprints = append(sprints, s)
			seen[s.ID] = struct{}{}
		}
	}
	reverse(sprints)

	return sprints
}

// SprintIssues fetches issues in the given sprint.
func (c *Client) SprintIssues(boardID, sprintID int, jql string, limit uint) (*SearchResult, error) {
	path := fmt.Sprintf("/board/%d/sprint/%d/issue?maxResults=%d", boardID, sprintID, limit)
	if jql != "" {
		path += fmt.Sprintf("&jql=%s", url.QueryEscape(jql))
	}

	res, err := c.GetV1(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out SearchResult

	err = json.NewDecoder(res.Body).Decode(&out)

	return &out, err
}

// LastNSprints fetches sprint in descending order.
//
// Jira api to get all sprints doesn't provide an option to sort results and
// returns result in ascending order by default. So, we will have to send
// multiple requests to get the results we are interested in.
func (c *Client) lastNSprints(boardID int, qp string, limit int) (*SprintResult, error) {
	var (
		s        *SprintResult
		err      error
		n, total int
	)

	for {
		s, err = c.Sprints(boardID, qp, n, limit)
		if err != nil {
			break
		}
		if s.IsLast {
			total = s.StartAt + len(s.Sprints)
			break
		}
		n += limit
	}

	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, ErrNoResult
	}

	n = total - limit
	if n < 0 {
		return s, err
	}
	return c.Sprints(boardID, qp, n, limit)
}

func injectBoardID(sprints []*Sprint, boardID int) {
	for _, s := range sprints {
		s.BoardID = boardID
	}
}

func reverse(s []*Sprint) {
	n := len(s)
	if n < 2 {
		return
	}
	for i := 0; i < n/2; i++ {
		j := n - i - 1
		s[i], s[j] = s[j], s[i]
	}
}
