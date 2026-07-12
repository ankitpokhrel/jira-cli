package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdListHasRawFlag(t *testing.T) {
	cmd := NewCmdList()

	flag := cmd.Flags().Lookup(flagRaw)

	assert.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}
