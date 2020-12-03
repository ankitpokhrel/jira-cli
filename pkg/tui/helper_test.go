package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColumnPadding(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		numPad   uint
		expected string
	}{
		{
			name:     "it doesn't add col padding for empty string",
			input:    "",
			numPad:   1,
			expected: "",
		},
		{
			name:     "it adds col padding in a string",
			input:    "Hello, World!",
			numPad:   1,
			expected: " Hello, World! ",
		},
		{
			name:     "it adds multiple col padding in a string",
			input:    "Hello, World!",
			numPad:   3,
			expected: "   Hello, World!   ",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, pad(tc.input, tc.numPad))
		})
	}
}
