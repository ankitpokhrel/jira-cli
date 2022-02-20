package jira

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBoards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/agile/1.0/board", r.URL.Path)

		qs := r.URL.Query()

		switch qs.Get("projectKeyOrId") {
		case "BAD":
			w.WriteHeader(400)
		case "TEST":
			assert.Equal(t, url.Values{
				"projectKeyOrId": []string{"TEST"},
				"type":           []string{"scrum"},
			}, qs)

			resp, err := ioutil.ReadFile("./testdata/boards.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	_, err := client.Boards("BAD", "scrum")
	assert.Error(t, &ErrUnexpectedResponse{}, err)

	actual, err := client.Boards("TEST", "scrum")
	assert.NoError(t, err)

	expected := &BoardResult{
		MaxResults: 50,
		Total:      2,
		Boards: []*Board{
			{
				ID:   1,
				Name: "Board 1",
				Type: "scrum",
			},
			{
				ID:   2,
				Name: "Board 2",
				Type: "scrum",
			},
		},
	}
	assert.Equal(t, expected, actual)
}
