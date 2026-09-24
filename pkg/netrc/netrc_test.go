package netrc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead_URLParsing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		machine     string
		login       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "absolute path URL parses but has empty host",
			machine:     "/some/path",
			login:       "user",
			wantErr:     true,
			errContains: "missing host",
		},
		{
			name:        "bare hostname without scheme is invalid URI",
			machine:     "example.com",
			login:       "user",
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:    "valid URL with host returns not-found for unknown entry",
			machine: "https://no-such-host.jira-cli-test.invalid",
			login:   "user@example.com",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entry, err := Read(tc.machine, tc.login)
			require.Error(t, err)
			assert.Nil(t, entry)

			if tc.errContains != "" {
				assert.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestRead_NotFound(t *testing.T) {
	t.Parallel()

	_, err := Read("https://no-such-host.jira-cli-test.invalid", "user@example.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNetrcEntryNotFound), "expected ErrNetrcEntryNotFound, got: %v", err)
}

func TestRead_EmptyHostReturnsDistinctError(t *testing.T) {
	t.Parallel()

	_, err := Read("/absolute/path", "user")
	require.Error(t, err)
	// Should NOT be ErrNetrcEntryNotFound — it's a distinct validation error
	assert.False(t, errors.Is(err, ErrNetrcEntryNotFound))
	assert.Contains(t, err.Error(), "missing host")
}
