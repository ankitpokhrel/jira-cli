package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
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

func TestShallOverwrite(t *testing.T) {
	cases := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name: "returns false when no_input mode is enabled",
			setup: func() {
				viper.Reset()
				viper.Set("no_input", true)
			},
			expected: false,
		},
		{
			name: "returns false when no_input mode is disabled or unset",
			setup: func() {
				viper.Reset()
				viper.Set("no_input", false)
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			defer viper.Reset()

			result := shallOverwrite()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfigureInstallationTypeNoInput(t *testing.T) {
	cases := []struct {
		name             string
		setup            func()
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name: "returns error when no_input mode is enabled and installation type not provided",
			setup: func() {
				viper.Reset()
				viper.Set("no_input", true)
			},
			expectError:      true,
			expectedErrorMsg: "installation type required in non-interactive mode",
		},
		{
			name: "succeeds when installation type is provided in no_input mode",
			setup: func() {
				viper.Reset()
				viper.Set("no_input", true)
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			defer viper.Reset()

			cfg := &JiraCLIConfig{
				Installation: "cloud",
			}
			if tc.expectError {
				cfg.Installation = ""
			}

			gen := NewJiraCLIConfigGenerator(cfg)
			err := gen.configureInstallationType()

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
