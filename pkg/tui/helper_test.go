package tui

import (
	"os"
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

func TestSplitText(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "it properly splits one line text",
			input:    "Hello, World!",
			expected: []string{"Hello, World!"},
		},
		{
			name:     `it splits multiline text separated with \n`,
			input:    "Hello, World!\nHow is it going?",
			expected: []string{"Hello, World!", "How is it going?"},
		},
		{
			name:     `it splits multiline text separated with \r\n`,
			input:    "Hello, World!\r\nHow is it going?",
			expected: []string{"Hello, World!", "How is it going?"},
		},
		{
			name: "it splits multiline text separated by backticks",
			input: `Hello, World!
					How is it going?
					Is everything alright!`,
			expected: []string{"Hello, World!", "\t\t\t\t\tHow is it going?", "\t\t\t\t\tIs everything alright!"},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, splitText(tc.input))
		})
	}
}

func TestGetPager(t *testing.T) {
	t.Parallel()

	pager := os.Getenv("PAGER")

	_ = os.Setenv("PAGER", "")
	assert.Equal(t, "less -r", GetPager())

	_ = os.Setenv("PAGER", "more")
	assert.Equal(t, "more", GetPager())

	_ = os.Setenv("PAGER", pager)
}
