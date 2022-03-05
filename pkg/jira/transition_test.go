package jira

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
			resp, err := ioutil.ReadFile("./testdata/transitions.json")
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
