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

func TestUserSearch(t *testing.T) {
	var (
		apiVersion2          bool
		unexpectedStatusCode bool
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiVersion2 {
			assert.Equal(t, "/rest/api/2/user/assignable/search", r.URL.Path)
		} else {
			assert.Equal(t, "/rest/api/3/user/assignable/search", r.URL.Path)
		}

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"query":      []string{"doe"},
				"startAt":    []string{"1"},
				"maxResults": []string{"5"},
				"accountId":  []string{"a123b"},
			}, r.URL.Query())

			resp, err := ioutil.ReadFile("./testdata/users.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.UserSearch(&UserSearchOptions{
		Query:      "doe",
		AccountID:  "a123b",
		StartAt:    1,
		MaxResults: 5,
	})
	assert.NoError(t, err)

	expected := []*User{
		{
			AccountID:   "5fb82376aca10c006949f35b",
			Email:       "jane@domain.tld",
			Name:        "janedoe",
			DisplayName: "Jane Doe",
			Active:      true,
		},
		{
			AccountID:   "5fb82376aca10c006949f35c",
			Email:       "jon@domain.tld",
			DisplayName: "Jon Doe",
			Active:      false,
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.UserSearch(nil)
	assert.Error(t, ErrInvalidSearchOption, err)

	_, err = client.UserSearch(&UserSearchOptions{})
	assert.Error(t, ErrInvalidSearchOption, err)

	apiVersion2 = true

	_, err = client.UserSearchV2(&UserSearchOptions{})
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
