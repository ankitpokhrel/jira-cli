package jira

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCreateMeta(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/createmeta", r.URL.Path)

		qs := r.URL.Query()

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			assert.Equal(t, url.Values{
				"projectKeys":    []string{"TEST"},
				"issuetypeNames": []string{"Epic"},
				"expand":         []string{"projects.issuetypes.fields"},
			}, qs)

			resp, err := ioutil.ReadFile("./testdata/createmeta.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3))

	actual, err := client.GetCreateMeta(&CreateMetaRequest{
		Projects:       "TEST",
		IssueTypeNames: "Epic",
		Expand:         "projects.issuetypes.fields",
	})
	assert.NoError(t, err)

	expected := &CreateMetaResponse{[]struct {
		Key        string `json:"key"`
		Name       string `json:"name"`
		IssueTypes []struct {
			Name   string                 `json:"name"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"issuetypes"`
	}{
		{
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []struct {
				Name   string                 `json:"name"`
				Fields map[string]interface{} `json:"fields"`
			}{
				{
					Name: "Epic",
					Fields: map[string]interface{}{
						"customfield_10011": map[string]interface{}{
							"name": "Epic Name",
							"key":  "customfield_10011",
						},
						"priority": map[string]interface{}{
							"name": "Priority",
							"key":  "priority",
						},
						"customfield_10014": map[string]interface{}{
							"name": "Epic Link",
							"key":  "customfield_10014",
						},
					},
				},
			},
		},
	}}

	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetCreateMeta(&CreateMetaRequest{
		Projects:       "TEST",
		IssueTypeNames: "Epic",
		Expand:         "projects.issuetypes.fields",
	})
	assert.Error(t, ErrUnexpectedStatusCode, err)
}
