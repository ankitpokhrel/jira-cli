package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertUnexpectedResponse asserts that err is, or wraps, an *ErrUnexpectedResponse.
//
// Use this rather than passing a want-error to assert.Error: that argument is the
// error under test, not a type to match, so the assertion would hold for any error.
func assertUnexpectedResponse(t *testing.T, err error) {
	t.Helper()

	var target *ErrUnexpectedResponse

	assert.ErrorAs(t, err, &target)
}
