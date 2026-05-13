package add

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testFlagParser struct {
	bools   map[string]bool
	strings map[string]string
}

func (tfp *testFlagParser) GetBool(name string) (bool, error) {
	return tfp.bools[name], nil
}

func (tfp *testFlagParser) GetString(name string) (string, error) {
	return tfp.strings[name], nil
}

func (*testFlagParser) GetStringArray(string) ([]string, error) { return nil, nil }
func (*testFlagParser) GetStringToString(string) (map[string]string, error) {
	return nil, nil
}
func (*testFlagParser) GetUint(string) (uint, error) { return 0, nil }
func (*testFlagParser) Set(string, string) error     { return nil }

func TestParseArgsAndFlags(t *testing.T) {
	params := parseArgsAndFlags(
		[]string{"ISSUE-1", "comment body"},
		&testFlagParser{
			bools: map[string]bool{
				"debug":    true,
				"no-input": true,
				"internal": true,
				"raw":      true,
			},
			strings: map[string]string{
				"template": "comment.tmpl",
			},
		},
	)

	assert.Equal(t, "ISSUE-1", params.issueKey)
	assert.Equal(t, "comment body", params.body)
	assert.Equal(t, "comment.tmpl", params.template)
	assert.True(t, params.noInput)
	assert.True(t, params.internal)
	assert.True(t, params.raw)
	assert.True(t, params.debug)
}
