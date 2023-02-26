package config

import (
	"os"
	"path/filepath"
	"testing"

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
