package cmdcommon

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// jira.IssueTypeField declares only json tags, so viper cannot match the yaml keys
// `jira init` writes ("schema.datatype") against the struct's json names
// ("schema.type"). The decode survives purely on mapstructure's case-insensitive
// fallback to Go field names, which no type or tag pins down. Viper has already
// swapped its mapstructure implementation once; this test is what would catch the
// next such swap silently dropping custom fields.
func TestGetConfiguredCustomFields(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.SetConfigType("yaml")

	err := viper.ReadConfig(strings.NewReader(`
issue:
  fields:
    custom:
      - name: Story Points
        key: customfield_10016
        schema:
          datatype: number
      - name: Sprint
        key: customfield_10020
        schema:
          datatype: array
          items: string
`))
	assert.NoError(t, err)

	fields, err := GetConfiguredCustomFields()
	assert.NoError(t, err)
	assert.Len(t, fields, 2)

	assert.Equal(t, "Story Points", fields[0].Name)
	assert.Equal(t, "customfield_10016", fields[0].Key)
	assert.Equal(t, "number", fields[0].Schema.DataType)
	assert.Empty(t, fields[0].Schema.Items)

	assert.Equal(t, "Sprint", fields[1].Name)
	assert.Equal(t, "customfield_10020", fields[1].Key)
	assert.Equal(t, "array", fields[1].Schema.DataType)
	assert.Equal(t, "string", fields[1].Schema.Items)
}
