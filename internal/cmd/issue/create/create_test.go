package create

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Issue-type IDs are written to the profile config unquoted, so viper decodes
// them as int (YAML) or float64 (JSON). setIssueTypes used to assert them as
// string, which panicked on every config `jira init` has ever generated.
func TestSetIssueTypesCoercesNonStringIDs(t *testing.T) {
	cases := []struct {
		name string
		id   interface{}
		want string
	}{
		{name: "yaml int", id: 10001, want: "10001"},
		{name: "json float64", id: float64(10001), want: "10001"},
		{name: "quoted string", id: "10001", want: "10001"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("issue.types", []interface{}{
				map[string]interface{}{
					"id": tc.id, "name": "Task", "handle": "Task", "subtask": false,
				},
			})

			cc := createCmd{}
			assert.NoError(t, cc.setIssueTypes())
			assert.Len(t, cc.issueTypes, 1)
			assert.Equal(t, tc.want, cc.issueTypes[0].ID)
			assert.Equal(t, "Task", cc.issueTypes[0].Name)
			assert.False(t, cc.issueTypes[0].Subtask)
		})
	}
}

func TestSetIssueTypesSkipsEpicAndKeepsSubtask(t *testing.T) {
	viper.Reset()
	viper.Set("issue.types", []interface{}{
		map[string]interface{}{"id": 10000, "name": "Epic", "handle": "Epic", "subtask": false},
		map[string]interface{}{"id": 10003, "name": "Sub-task", "handle": "Sub-task", "subtask": true},
	})

	cc := createCmd{}
	assert.NoError(t, cc.setIssueTypes())
	assert.Len(t, cc.issueTypes, 1)
	assert.Equal(t, "10003", cc.issueTypes[0].ID)
	assert.True(t, cc.issueTypes[0].Subtask)
}

// A genuinely unusable config must return the error, not panic.
func TestSetIssueTypesErrorsRatherThanPanics(t *testing.T) {
	cases := []struct {
		name  string
		types interface{}
	}{
		{name: "not a list", types: "nope"},
		{name: "entry not a map", types: []interface{}{"nope"}},
		{name: "missing name", types: []interface{}{map[string]interface{}{"id": 10001}}},
		{name: "unusable id", types: []interface{}{
			map[string]interface{}{"id": []interface{}{1}, "name": "Task"},
		}},
		{name: "missing id", types: []interface{}{map[string]interface{}{"name": "Task"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("issue.types", tc.types)

			cc := createCmd{}
			assert.NotPanics(t, func() {
				assert.EqualError(t, cc.setIssueTypes(), "invalid issue types in config")
			})
		})
	}
}
