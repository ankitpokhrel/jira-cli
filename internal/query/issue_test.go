package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type issueParamsErr struct {
	history    bool
	watching   bool
	resolution bool
	issueType  bool
	labels     bool
	status     bool
	sprints    bool
}

type issueFlagParser struct {
	err           issueParamsErr
	noHistory     bool
	noWatching    bool
	orderDesc     bool
	emptyType     bool
	labels        []string
	status        []string
	sprints       []string
	withCreated   bool
	withUpdated   bool
	created       string
	updated       string
	createdAfter  string
	createdBefore string
	updatedAfter  string
	updatedBefore string
	jql           string
	orderBy       string
}

func (tfp *issueFlagParser) GetBool(name string) (bool, error) {
	if tfp.err.history && name == "history" {
		return false, fmt.Errorf("oops! couldn't fetch history flag")
	}
	if tfp.err.watching && name == "watching" {
		return false, fmt.Errorf("oops! couldn't fetch watching flag")
	}
	if tfp.noHistory && name == "history" {
		return false, nil
	}
	if tfp.noWatching && name == "watching" {
		return false, nil
	}
	if tfp.orderDesc && name == "reverse" {
		return false, nil
	}
	return true, nil
}

//nolint:gocyclo
func (tfp *issueFlagParser) GetString(name string) (string, error) {
	if tfp.err.resolution && name == "resolution" {
		return "", fmt.Errorf("oops! couldn't fetch resolution flag")
	}
	if tfp.err.issueType && name == "type" {
		return "", fmt.Errorf("oops! couldn't fetch type flag")
	}
	if tfp.created != "" && name == "created" {
		return tfp.created, nil
	}
	if tfp.updated != "" && name == "updated" {
		return tfp.updated, nil
	}
	if tfp.emptyType && name == "type" {
		return "", nil
	}
	if name == "jql" {
		return tfp.jql, nil
	}
	if tfp.orderBy == "" && name == "order-by" {
		return "created", nil
	}
	if strings.HasPrefix(name, "created") {
		if tfp.withCreated {
			switch name {
			case "created-after":
				return tfp.createdAfter, nil
			case "created-before":
				return tfp.createdBefore, nil
			}
		}
		return "", nil
	}
	if strings.HasPrefix(name, "updated") {
		if tfp.withUpdated {
			switch name {
			case "updated-after":
				return tfp.updatedAfter, nil
			case "updated-before":
				return tfp.updatedBefore, nil
			}
		}
		return "", nil
	}
	if name == "paginate" {
		return "", nil
	}
	return "test", nil
}

func (tfp *issueFlagParser) GetStringArray(name string) ([]string, error) {
	if tfp.err.labels && name == "label" {
		return []string{}, fmt.Errorf("oops! couldn't fetch label flag")
	}
	if tfp.err.status && name == "status" {
		return []string{}, fmt.Errorf("oops! couldn't fetch status flag")
	}
	if tfp.err.sprints && name == "sprint" {
		return []string{}, fmt.Errorf("oops! couldn't fetch sprint flag")
	}
	if name == "status" {
		return tfp.status, nil
	}
	if name == "sprint" {
		return tfp.sprints, nil
	}
	return tfp.labels, nil
}

func (*issueFlagParser) GetStringToString(string) (map[string]string, error) { return nil, nil }
func (*issueFlagParser) GetUint(string) (uint, error)                        { return 100, nil }
func (*issueFlagParser) Set(string, string) error                            { return nil }

func TestIssueGet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		initialize func() *Issue
		expected   string
	}{
		{
			name: "query with default parameters",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" ` +
				`AND assignee="test" AND component="test" AND parent="test" ORDER BY lastViewed ASC`,
		},
		{
			name: "query without issue history parameter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{noHistory: true})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" ` +
				`AND assignee="test" AND component="test" AND parent="test" ORDER BY created ASC`,
		},
		{
			name: "query only with fields filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{noHistory: true, noWatching: true})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" ` +
				`AND assignee="test" AND component="test" AND parent="test" ORDER BY created ASC`,
		},
		{
			name: "query with error when fetching history flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					history: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query with error when fetching watching flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					watching: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query with error when fetching resolution flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					resolution: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query with error when fetching labels flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					labels: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query with error when fetching status flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					status: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query with error when fetching type flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					issueType: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query without issue type flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{emptyType: true})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" ORDER BY lastViewed ASC`,
		},
		{
			name: "query with reverse set to true",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{orderDesc: true})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" ` +
				`AND assignee="test" AND component="test" AND parent="test" ORDER BY lastViewed DESC`,
		},
		{
			name: "query with labels",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{labels: []string{"first", "second", "third"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND labels IN ("first", "second", "third") ORDER BY lastViewed ASC`,
		},
		{
			name: "query with status",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{status: []string{"first", "second", "~third"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND status IN ("first", "second") AND status NOT IN ("third") ORDER BY lastViewed ASC`,
		},
		{
			name: "query with created and updated today filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{created: "today", updated: "today"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>=startOfDay() AND updatedDate>=startOfDay() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with created and updated week filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{created: "week", updated: "week"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>=startOfWeek() AND updatedDate>=startOfWeek() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with created and updated month filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{created: "month", updated: "month"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>=startOfMonth() AND updatedDate>=startOfMonth() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with created and updated year filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{created: "year", updated: "year"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>=startOfYear() AND updatedDate>=startOfYear() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with created and updated filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{created: "2020-12-31", updated: "2020-12-31"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" AND component="test" ` +
				`AND parent="test" AND createdDate>="2020-12-31" AND createdDate<"2021-01-01" AND updatedDate>="2020-12-31" AND updatedDate<"2021-01-01" ` +
				`ORDER BY lastViewed ASC`,
		},
		{
			name: "created and updated filter with incorrect date format",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{created: "2020-15-31", updated: "2020-12-31 10:30:30"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>="2020-15-31" AND updatedDate>="2020-12-31 10:30:30" ORDER BY lastViewed ASC`,
		},
		{
			name: "query with created-after and created-before filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{createdAfter: "2020-12-01", createdBefore: "2020-12-31", withCreated: true})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>"2020-12-01" AND createdDate<"2020-12-31" ORDER BY lastViewed ASC`,
		},
		{
			name: "query with updated-after and updated-before filter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{updatedAfter: "2020-12-01", updatedBefore: "2020-12-31", withUpdated: true})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND updatedDate>"2020-12-01" AND updatedDate<"2020-12-31" ORDER BY lastViewed ASC`,
		},
		{
			name: "created and updated flags gets precedence",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{
					created:       "2020-11-01",
					updated:       "-10d",
					createdAfter:  "2020-12-01",
					updatedBefore: "2020-12-31",
					withCreated:   true,
					withUpdated:   true,
				})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND createdDate>="2020-11-01" AND createdDate<"2020-11-02" AND updatedDate>="-10d" ` +
				`ORDER BY lastViewed ASC`,
		},
		{
			name: "created order gets priority over updated flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{
					created:       "2020-11-01",
					updated:       "-10d",
					createdAfter:  "2020-12-01",
					updatedBefore: "2020-12-31",
					withCreated:   true,
					withUpdated:   true,
					noHistory:     true,
				})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN watchedIssues() AND type="test" AND resolution="test" ` +
				`AND priority="test" AND reporter="test" AND assignee="test" AND component="test" ` +
				`AND parent="test" AND createdDate>="2020-11-01" AND createdDate<"2020-11-02" AND updatedDate>="-10d" ` +
				`ORDER BY created ASC`,
		},
		{
			name: "it orders by updated if only updated flags are present",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{
					updatedAfter:  "2020-11-31",
					updatedBefore: "2020-12-31",
					withCreated:   false,
					withUpdated:   true,
					noHistory:     true,
				})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN watchedIssues() AND type="test" AND resolution="test" ` +
				`AND priority="test" AND reporter="test" AND assignee="test" AND component="test" ` +
				`AND parent="test" AND updatedDate>"2020-11-31" AND updatedDate<"2020-12-31" ` +
				`ORDER BY updated ASC`,
		},
		{
			name: "query with error when fetching sprint flag",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{err: issueParamsErr{
					sprints: true,
				}})
				assert.Error(t, err)
				return i
			},
			expected: "",
		},
		{
			name: "query with sprint by name",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"Sprint 42"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN ("Sprint 42") ORDER BY lastViewed ASC`,
		},
		{
			name: "query with sprint by id (numeric, unquoted)",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"123"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN (123) ORDER BY lastViewed ASC`,
		},
		{
			name: "query with current sprint keyword",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"current"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN openSprints() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with active sprint keyword is equivalent to current",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"ACTIVE"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN openSprints() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with closed sprint keyword",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"closed"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN closedSprints() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with future sprint keyword",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"future"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN futureSprints() ORDER BY lastViewed ASC`,
		},
		{
			name: "query with multiple sprints mixes quoted names and unquoted ids",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"Sprint 41", "Sprint 42", "123"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN ("Sprint 41", "Sprint 42", 123) ` +
				`ORDER BY lastViewed ASC`,
		},
		{
			name: "query mixing sprint keywords and names deduplicates keywords",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{sprints: []string{"current", "active", "closed", "Sprint 42"}})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" AND assignee="test" ` +
				`AND component="test" AND parent="test" AND sprint IN closedSprints() AND sprint IN openSprints() ` +
				`AND sprint IN ("Sprint 42") ORDER BY lastViewed ASC`,
		},
		{
			name: "query with jql parameter",
			initialize: func() *Issue {
				i, err := NewIssue("TEST", &issueFlagParser{jql: "summary ~ cli OR x = y"})
				assert.NoError(t, err)
				return i
			},
			expected: `project="TEST" AND summary ~ cli OR x = y AND issue IN issueHistory() AND issue IN watchedIssues() AND ` +
				`type="test" AND resolution="test" AND priority="test" AND reporter="test" ` +
				`AND assignee="test" AND component="test" AND parent="test" ORDER BY lastViewed ASC`,
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
