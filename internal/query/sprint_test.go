package query

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type sprintParamsErr struct {
	state bool
}

type sprintFlagParser struct {
	err        sprintParamsErr
	state      string
	emptyState bool
}

func (tfp sprintFlagParser) GetBool(string) (bool, error) {
	return true, nil
}

func (tfp sprintFlagParser) GetString(name string) (string, error) {
	if tfp.err.state && name == "state" {
		return "", fmt.Errorf("oops! couldn't fetch state flag")
	}

	if tfp.emptyState && name == "state" {
		return "", nil
	}

	return "future", nil
}

func (tfp sprintFlagParser) GetStringArray(string) ([]string, error) {
	return []string{}, nil
}

func (tfp sprintFlagParser) Set(string, string) error {
	return nil
}

func TestSprintGet(t *testing.T) {
	cases := []struct {
		name       string
		initialize func() *Sprint
		expected   string
	}{
		{
			name: "query with default parameters",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{
					emptyState: true,
				})
				assert.NoError(t, err)

				return s
			},
			expected: "state=active,closed",
		},
		{
			name: "query with state parameter",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{})
				assert.NoError(t, err)

				return s
			},
			expected: "state=future",
		},
		{
			name: "query with error when fetching state flag",
			initialize: func() *Sprint {
				s, err := NewSprint(&sprintFlagParser{err: sprintParamsErr{
					state: true,
				}})
				assert.Error(t, err)

				return s
			},
			expected: "",
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
