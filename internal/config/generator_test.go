package config

import (
	"os"
	"path/filepath"
	"runtime"
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
	dir := t.TempDir()
	file := filepath.Join(dir, ".jira.yml")

	// case: file doesn't exist - should create at 0600.
	assert.NoError(t, create(file))
	assert.FileExists(t, file)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(file)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "expected mode 0600 on initial create")
	}

	// Write some content to simulate viper's WriteConfig populating the file.
	assert.NoError(t, os.WriteFile(file, []byte("server: https://example.atlassian.net\n"), 0o600))

	// case: file exists - should atomically replace, no .bkp residue.
	assert.NoError(t, create(file))
	assert.FileExists(t, file)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(file)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "expected mode 0600 after re-create")
	}

	// No .bkp file should be left behind by create() on either invocation.
	_, err := os.Stat(file + ".bkp")
	assert.True(t, os.IsNotExist(err), "create() must not leave a .bkp residue (got err=%v)", err)

	// And no stray tempfiles should remain in the directory.
	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	for _, e := range entries {
		name := e.Name()
		assert.NotContains(t, name, ".bkp", "directory must not contain .bkp residue")
		assert.NotContains(t, name, ".tmp.", "directory must not contain unfinished tempfiles")
	}
}
