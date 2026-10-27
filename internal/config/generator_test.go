package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/stretchr/testify/assert"
)

func TestExists(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "it returns false for empty file",
			input:    "",
			expected: false,
		},
		{
			name:     "it returns false if file doesn't exist",
			input:    "invalid.txt",
			expected: false,
		},
		{
			name:     "it returns true if the file exist",
			input:    "/testdata/empty.txt",
			expected: true,
		},
	}

	cwd, err := os.Getwd()
	assert.NoError(t, err)

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := tc.input
			if path != "" {
				path = cwd + tc.input
			}

			assert.Equal(t, tc.expected, Exists(path))
		})
	}
}

func TestCreate(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	file := cwd + "/testdata/.tmp/.jira.yml"

	// case: file doesn't exist
	assert.NoError(t, create(file))

	// case: file exists, will create .bkp file
	assert.NoError(t, create(file))

	// Remove created file. Fails if those files were not created.
	assert.NoError(t, os.Remove(file))
	assert.NoError(t, os.Remove(file+".bkp"))
	assert.NoError(t, os.Remove(filepath.Dir(file)))
}

func TestConfigureFields(t *testing.T) {
	fieldsWithoutEpic := `[
		{"id":"customfield_10001","name":"Story Points","custom":true,"schema":{"type":"string"}}
	]`

	createMeta := `{
		"projects": [
			{
				"key": "TEST",
				"name": "Test Project",
				"issuetypes": [
					{
						"id": "10001",
						"name": "Epic",
						"subtask": false,
						"fields": {
							"customfield_10011": {"name":"Epic Name","key":"customfield_10011"},
							"customfield_10014": {"name":"Epic Link","key":"customfield_10014"}
						}
					},
					{
						"id": "10002",
						"name": "Task",
						"subtask": false,
						"fields": {
							"customfield_10014": {"name":"Epic Link","key":"customfield_10014"}
						}
					}
				]
			}
		]
	}`

	t.Run("resolves epic fields from create metadata when missing in field list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/rest/api/2/field":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(fieldsWithoutEpic))
			case "/rest/api/2/issue/createmeta":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(createMeta))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		gen := &JiraCLIConfigGenerator{
			jiraClient: jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second)),
		}
		gen.value.project = &projectConf{Key: "TEST"}

		assert.NoError(t, gen.configureFields())

		assert.Equal(t, "customfield_10011", gen.value.epic.Name)
		assert.Equal(t, "customfield_10014", gen.value.epic.Link)
		assert.Len(t, gen.value.customFields, 1)
	})

	t.Run("prefers epic fields from the global field list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/rest/api/2/field":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[
					{"id":"customfield_10001","name":"Epic Name","custom":true,"schema":{"type":"string"}},
					{"id":"customfield_10002","name":"Epic Link","custom":true,"schema":{"type":"string"}}
				]`))
			default:
				t.Errorf("unexpected call to %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		gen := &JiraCLIConfigGenerator{
			jiraClient: jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second)),
		}
		gen.value.project = &projectConf{Key: "TEST"}

		assert.NoError(t, gen.configureFields())

		assert.Equal(t, "customfield_10001", gen.value.epic.Name)
		assert.Equal(t, "customfield_10002", gen.value.epic.Link)
	})
}
