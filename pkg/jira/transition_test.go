package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransitions(t *testing.T) {
	var (
		apiVersion2          bool
		unexpectedStatusCode bool
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiVersion2 {
			assert.Equal(t, "/rest/api/2/issue/TEST/transitions", r.URL.Path)
		} else {
			assert.Equal(t, "/rest/api/3/issue/TEST/transitions", r.URL.Path)
		}

		assert.Equal(t, "GET", r.Method)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/transitions.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.Transitions("TEST")
	assert.NoError(t, err)

	expected := []*Transition{
		{
			ID:          "11",
			Name:        "To Do",
			IsAvailable: true,
		},
		{
			ID:          "21",
			Name:        "In Progress",
			IsAvailable: true,
		},
		{
			ID:          "31",
			Name:        "Done",
			IsAvailable: false,
		},
	}
	assert.Equal(t, expected, actual)

	apiVersion2 = true
	unexpectedStatusCode = true

	_, err = client.TransitionsV2("TEST")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestTransition(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST/transitions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		expectedBody := `{"transition":{"id":"31","name":"Done"}}`
		assert.Equal(t, expectedBody, actualBody.String())

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(204)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	requestData := TransitionRequest{Transition: &TransitionRequestData{
		ID:   "31",
		Name: "Done",
	}}
	code, err := client.Transition("TEST", &requestData)
	assert.NoError(t, err)
	assert.Equal(t, code, 204)
}

func TestTransitionsScreenFields(t *testing.T) {
	// Mirrors a real Jira response: a transition with no screen reports an empty
	// fields object, while one with a screen lists the fields it carries. Note that
	// Jira never advertises "comment" here, even on transitions that do accept one.
	const body = `{
		"expand": "transitions",
		"transitions": [
			{"id": "421", "name": "Ready for review", "isAvailable": true, "fields": {}},
			{"id": "451", "name": "Done", "isAvailable": true, "fields": {
				"resolution": {"required": true},
				"summary": {"required": true}
			}}
		]
	}`

	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/rest/api/3/issue/TEST/transitions", r.URL.Path)
		gotQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.Transitions("TEST")
	assert.NoError(t, err)

	// The fields expansion must be requested, otherwise Fields is always empty and a
	// screen-less transition is indistinguishable from one with a screen.
	assert.Equal(t, "expand=transitions.fields", gotQuery)

	assert.Len(t, actual, 2)
	assert.Empty(t, actual[0].Fields, "transition without a screen should report no fields")
	assert.Len(t, actual[1].Fields, 2, "transition with a screen should report its fields")
}
