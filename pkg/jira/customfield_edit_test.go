package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEditWithUserCustomField(t *testing.T) {
	expectedBody := `{"update":{"customfield_12574":[{"set":{"accountId":"5f7e1b2c"}}]},"fields":{"parent":{}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.JSONEq(t, expectedBody, actualBody.String())

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	edr := EditRequest{
		CustomFields: map[string]string{
			"pm-owner": "5f7e1b2c",
		},
	}
	edr.WithCustomFields([]IssueTypeField{
		{
			Name: "PM owner",
			Key:  "customfield_12574",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{DataType: "user"},
		},
	})
	edr.ForInstallationType(InstallationTypeCloud)

	err := client.Edit("TEST-1", &edr)
	assert.NoError(t, err)
}

func TestEditWithUserCustomFieldLocal(t *testing.T) {
	expectedBody := `{"update":{"customfield_12574":[{"set":{"name":"john.doe"}}]},"fields":{"parent":{}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.JSONEq(t, expectedBody, actualBody.String())

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	edr := EditRequest{
		CustomFields: map[string]string{
			"pm-owner": "john.doe",
		},
	}
	edr.WithCustomFields([]IssueTypeField{
		{
			Name: "PM owner",
			Key:  "customfield_12574",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{DataType: "user"},
		},
	})
	edr.ForInstallationType(InstallationTypeLocal)

	err := client.Edit("TEST-1", &edr)
	assert.NoError(t, err)
}

func TestEditWithMultiUserCustomField(t *testing.T) {
	expectedBody := `{"update":{"customfield_10100":[{"set":[{"accountId":"user1"},{"accountId":"user2"}]}]},"fields":{"parent":{}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		if !assert.JSONEq(t, expectedBody, actualBody.String()) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	edr := EditRequest{
		CustomFields: map[string]string{
			"reviewers": "user1,user2",
		},
	}
	edr.WithCustomFields([]IssueTypeField{
		{
			Name: "Reviewers",
			Key:  "customfield_10100",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{DataType: "array", Items: "user"},
		},
	})
	edr.ForInstallationType(InstallationTypeCloud)

	err := client.Edit("TEST-1", &edr)
	assert.NoError(t, err)
}

func TestConstructCustomFieldsForEditUser(t *testing.T) {
	data := &editRequest{}

	fields := map[string]string{
		"pm-owner": "5f7e1b2c",
	}
	configuredFields := []IssueTypeField{
		{
			Name: "PM owner",
			Key:  "customfield_12574",
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
			}{DataType: "user"},
		},
	}

	constructCustomFieldsForEdit(fields, configuredFields, InstallationTypeCloud, data)

	result, ok := data.Update.M.customFields["customfield_12574"]
	assert.True(t, ok)

	b, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.JSONEq(t, `[{"set":{"accountId":"5f7e1b2c"}}]`, string(b))
}

func TestConstructCustomFieldsForEditMultiUser(t *testing.T) {
	cases := []struct {
		name         string
		installation string
		value        string
		want         string
	}{
		{
			name: "cloud multiple users", installation: InstallationTypeCloud,
			value: "user1,user2", want: `[{"set":[{"accountId":"user1"},{"accountId":"user2"}]}]`,
		},
		{
			name: "local multiple users", installation: InstallationTypeLocal,
			value: "user1,user2", want: `[{"set":[{"name":"user1"},{"name":"user2"}]}]`,
		},
		{
			name: "cloud one user remains an array", installation: InstallationTypeCloud,
			value: "user1", want: `[{"set":[{"accountId":"user1"}]}]`,
		},
		{
			name: "local one user remains an array", installation: InstallationTypeLocal,
			value: "user1", want: `[{"set":[{"name":"user1"}]}]`,
		},
		{
			name: "cloud trims each user", installation: InstallationTypeCloud,
			value: " user1, user2 ", want: `[{"set":[{"accountId":"user1"},{"accountId":"user2"}]}]`,
		},
		{
			name: "local trims each user", installation: InstallationTypeLocal,
			value: " user1, user2 ", want: `[{"set":[{"name":"user1"},{"name":"user2"}]}]`,
		},
		{
			name:  "default uses account IDs",
			value: "user1,user2", want: `[{"set":[{"accountId":"user1"},{"accountId":"user2"}]}]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := &editRequest{}
			configuredFields := []IssueTypeField{
				{
					Name: "Reviewers",
					Key:  "customfield_10100",
					Schema: struct {
						DataType string `json:"type"`
						Items    string `json:"items,omitempty"`
					}{DataType: "array", Items: "user"},
				},
			}
			constructCustomFieldsForEdit(map[string]string{"reviewers": tc.value}, configuredFields, tc.installation, data)
			result, ok := data.Update.M.customFields["customfield_10100"]
			assert.True(t, ok)
			b, err := json.Marshal(result)
			assert.NoError(t, err)
			assert.JSONEq(t, tc.want, string(b))
		})
	}
}

func TestConstructCustomFieldsForEditEmpty(t *testing.T) {
	data := &editRequest{}

	constructCustomFieldsForEdit(nil, nil, InstallationTypeCloud, data)
	assert.Nil(t, data.Update.M.customFields)

	constructCustomFieldsForEdit(map[string]string{}, []IssueTypeField{}, InstallationTypeCloud, data)
	assert.Nil(t, data.Update.M.customFields)
}
