package clone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReplace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		from    string
		to      string
		wantErr bool
	}{
		{
			name:  "simple pair",
			input: "find me:replace with me",
			from:  "find me",
			to:    "replace with me",
		},
		{
			name:  "empty replacement",
			input: "remove me:",
			from:  "remove me",
			to:    "",
		},
		{
			name:  "replacement containing colons",
			input: "old link:https://example.com:8080/path",
			from:  "old link",
			to:    "https://example.com:8080/path",
		},
		{
			name:  "replacement containing a time",
			input: "TBD:10:30 AM",
			from:  "TBD",
			to:    "10:30 AM",
		},
		{
			name:    "missing separator",
			input:   "no separator",
			wantErr: true,
		},
		{
			name:    "empty search string",
			input:   ":x",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			from, to, err := parseReplace(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.from, from)
			assert.Equal(t, tc.to, to)
		})
	}
}
