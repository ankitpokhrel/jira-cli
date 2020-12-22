package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TransitionRequest struct holds request data for transition request.
type TransitionRequest struct {
	Transition *TransitionRequestData `json:"transition"`
}

// TransitionRequestData is a transition request data.
type TransitionRequestData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type transitionResponse struct {
	Expand      string        `json:"expand"`
	Transitions []*Transition `json:"transitions"`
}

// Transitions fetches valid transitions for an issue from GET /issue/{key}/transitions endpoint.
func (c *Client) Transitions(key string) ([]*Transition, error) {
	path := fmt.Sprintf("/issue/%s/transitions", key)

	res, err := c.Get(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrEmptyResponse
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, ErrUnexpectedStatusCode
	}

	var out transitionResponse

	err = json.NewDecoder(res.Body).Decode(&out)

	return out.Transitions, err
}

// Transition moves issue from one state to another using POST /issue/{key}/transitions endpoint.
func (c *Client) Transition(key string, data *TransitionRequest) (int, error) {
	body, err := json.Marshal(&data)
	if err != nil {
		return 0, err
	}

	path := fmt.Sprintf("/issue/%s/transitions", key)

	res, err := c.Post(context.Background(), path, body, Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	if err != nil {
		return 0, err
	}

	if res == nil {
		return 0, ErrEmptyResponse
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent {
		return res.StatusCode, ErrUnexpectedStatusCode
	}

	return res.StatusCode, nil
}
