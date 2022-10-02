package query

import (
	"fmt"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Sprint is a query type for sprint command.
type Sprint struct {
	Flags  FlagParser
	params *SprintParams
}

// NewSprint creates and initializes a new Sprint type.
func NewSprint(flags FlagParser) (*Sprint, error) {
	sp := SprintParams{}
	if err := sp.init(flags); err != nil {
		return nil, err
	}
	return &Sprint{
		Flags:  flags,
		params: &sp,
	}, nil
}

// Get returns constructed query params.
func (s *Sprint) Get() string {
	var state string

	switch {
	case s.params.Status != "":
		state = fmt.Sprintf("state=%s", s.params.Status)
	case s.params.Current:
		state = fmt.Sprintf("state=%s", jira.SprintStateActive)
	case s.params.Prev:
		state = fmt.Sprintf("state=%s", jira.SprintStateClosed)
	case s.params.Next:
		state = fmt.Sprintf("state=%s", jira.SprintStateFuture)
	default:
		state = fmt.Sprintf("state=%s,%s", jira.SprintStateActive, jira.SprintStateClosed)
	}
	if s.params.debug {
		fmt.Printf("JQL: %s\n", state)
	}

	return state
}

// Params returns sprint command params.
func (s *Sprint) Params() *SprintParams {
	return s.params
}

// SprintParams is sprint command parameters.
type SprintParams struct {
	Status        string
	Current       bool
	Prev          bool
	Next          bool
	From          uint
	Limit         uint
	ShowAllIssues bool

	debug bool
}

func (sp *SprintParams) init(flags FlagParser) error {
	status, err := flags.GetString("state")
	if err != nil {
		return err
	}
	sp.Status = status

	current, err := flags.GetBool("current")
	if err != nil {
		return err
	}
	sp.Current = current

	prev, err := flags.GetBool("prev")
	if err != nil {
		return err
	}
	sp.Prev = prev

	next, err := flags.GetBool("next")
	if err != nil {
		return err
	}
	sp.Next = next

	showAll, err := flags.GetBool("show-all-issues")
	if err != nil {
		return err
	}
	sp.ShowAllIssues = showAll

	paginate, err := flags.GetString("paginate")
	if err != nil {
		return err
	}
	from, limit, err := getPaginateParams(paginate)
	if err != nil {
		return err
	}
	sp.From = from
	sp.Limit = limit

	debug, err := flags.GetBool("debug")
	if err != nil {
		return err
	}
	sp.debug = debug

	return nil
}
