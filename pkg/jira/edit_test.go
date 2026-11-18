package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdit(t *testing.T) {
	var expectedQueryParams string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, expectedQueryParams, r.URL.RawQuery)

		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.Edit("TEST-1", &EditRequest{ParentIssueKey: "EPIC-1"})
	assert.NoError(t, err)

	expectedQueryParams = "notifyUsers=false"
	err = client.Edit("TEST-1", &EditRequest{ParentIssueKey: "EPIC-1", SkipNotify: true})
	assert.NoError(t, err)
}

func TestEditMultiUserCustomField(t *testing.T) {
	field := IssueTypeField{Name: "Backend Developers", Key: "customfield_10001"}
	field.Schema.DataType = "array"
	field.Schema.Items = "user"

	tests := []struct {
		name string
		ids  string
		want string
	}{
		{"one user", "712020:alice", `[{"set":[{"accountId":"712020:alice"}]}]`},
		{"multiple users", "712020:alice, 712020:bob", `[{"set":[{"accountId":"712020:alice"},{"accountId":"712020:bob"}]}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload struct {
					Update map[string]json.RawMessage `json:"update"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); !assert.NoError(t, err) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				assert.JSONEq(t, tt.want, string(payload.Update[field.Key]))
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
			req := &EditRequest{CustomFields: map[string]string{"backend-developers": tt.ids}}
			req.WithCustomFields([]IssueTypeField{field})
			require.NoError(t, client.Edit("TEST-1", req))
		})
	}
}
