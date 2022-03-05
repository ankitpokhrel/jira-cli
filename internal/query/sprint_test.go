package query

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type sprintParamsErr struct {
	state   bool
	current bool
}

type sprintFlagParser struct {
	err     sprintParamsErr
	state   string
	current bool
	prev    bool
	next    bool
}

func (sfp sprintFlagParser) GetBool(name string) (bool, error) {
	if sfp.err.current && name == "current" {
		return true, fmt.Errorf("oops! couldn't fetch current flag")
	}
	if sfp.current && name == "current" {
		return true, nil
	}
	if sfp.prev && name == "prev" {
		return true, nil
	}
	if sfp.next && name == "next" {
		return true, nil
	}
	return false, nil
}

func (sfp sprintFlagParser) GetString(name string) (string, error) {
	if sfp.err.state && name == "state" {
		return "", fmt.Errorf("oops! couldn't fetch state flag")
	}
	return sfp.state, nil
}

func (sprintFlagParser) GetStringArray(string) ([]string, error)             { return []string{}, nil }
func (sprintFlagParser) GetStringToString(string) (map[string]string, error) { return nil, nil }
func (sprintFlagParser) GetUint(string) (uint, error)                        { return 100, nil }
func (sprintFlagParser) Set(string, string) error                            { return nil }

func TestSprintGet(t *testing.T) {
	cases := []struct {
		name       string
		initialize func() *Sprint
		expected   string
	}{
		{
			name: "query with default parameters",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{})
				assert.NoError(t, err)
				return s
			},
			expected: "state=active,closed",
		},
		{
			name: "query with state parameter",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{state: "future"})
				assert.NoError(t, err)
				return s
			},
			expected: "state=future",
		},
		{
			name: "query with error when fetching state flag",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{err: sprintParamsErr{state: true}})
				assert.Error(t, err)
				return s
			},
			expected: "",
		},
		{
			name: "query with current parameter",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{current: true})
				assert.NoError(t, err)
				return s
			},
			expected: "state=active",
		},
		{
			name: "query with error when fetching current flag",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{err: sprintParamsErr{current: true}})
				assert.Error(t, err)
				return s
			},
			expected: "",
		},
		{
			name: "query with prev parameter",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{prev: true})
				assert.NoError(t, err)
				return s
			},
			expected: "state=closed",
		},
		{
			name: "query with next parameter",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{next: true})
				assert.NoError(t, err)
				return s
			},
			expected: "state=future",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := tc.initialize()
			if q != nil {
				assert.Equal(t, tc.expected, q.Get())
			}
		})
	}
}
