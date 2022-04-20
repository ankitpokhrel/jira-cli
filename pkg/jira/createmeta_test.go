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

func TestGetCreateMeta(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/createmeta", r.URL.Path)

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

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetCreateMeta(&CreateMetaRequest{
		Projects:       "TEST",
		IssueTypeNames: "Epic",
		Expand:         "projects.issuetypes.fields",
	})
	assert.NoError(t, err)

	expected := &CreateMetaResponse{[]struct {
		Key        string                 `json:"key"`
		Name       string                 `json:"name"`
		IssueTypes []*CreateMetaIssueType `json:"issuetypes"`
	}{
		{
			Key:  "TEST",
			Name: "Test Project",
			IssueTypes: []*CreateMetaIssueType{
				{
					IssueType: IssueType{
						ID:      "10001",
						Name:    "Epic",
						Subtask: false,
					},
					Fields: map[string]IssueTypeField{
						"customfield_10011": {
							Name: "Epic Name",
							Key:  "customfield_10011",
						},
						"priority": {
							Name: "Priority",
							Key:  "priority",
						},
						"customfield_10014": {
							Name: "Epic Link",
							Key:  "customfield_10014",
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
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
