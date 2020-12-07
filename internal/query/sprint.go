package query

import (
	"fmt"
)

type sprintParams struct {
	status string
}

func (sp *sprintParams) init(flags FlagParser) error {
	status, err := flags.GetString("state")
	if err != nil {
		return err
	}

	sp.status = status

	return nil
}

// Sprint is a query type for issue command.
type Sprint struct {
	Flags  FlagParser
	params *sprintParams
}

// NewSprint creates and initialize new issue type.
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
	if s.params.status != "" {
		return fmt.Sprintf("state=%s", s.params.status)
	}

	return "state=active,closed"
}
