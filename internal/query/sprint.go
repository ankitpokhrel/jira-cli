package query

import (
	"fmt"
)

// Sprint is a query type for sprint command.
type Sprint struct {
	Flags  FlagParser
	params *sprintParams
}

// NewSprint creates and initializes a new Sprint type.
func NewSprint(flags FlagParser) (*Sprint, error) {
	sprint := Sprint{
		Flags: flags,
	}

	sp := sprintParams{}

	err := sp.init(flags)
	if err != nil {
		return nil, err
	}

	sprint.params = &sp

	return &sprint, nil
}

// Get returns constructed query params.
func (s *Sprint) Get() string {
	state := "state=active,closed"

	if s.params.status != "" {
		state = fmt.Sprintf("state=%s", s.params.status)
	}

	if s.params.debug {
		fmt.Printf("JQL: %s\n", state)
	}

	return state
}

type sprintParams struct {
	status string
	debug  bool
}

func (sp *sprintParams) init(flags FlagParser) error {
	status, err := flags.GetString("state")
	if err != nil {
		return err
	}

	debug, err := flags.GetBool("debug")
	if err != nil {
		return err
	}

	sp.status = status
	sp.debug = debug

	return nil
}
