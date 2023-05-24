package output

import (
	"fmt"
)

type Format string

const (
	JSON  Format = "json"
	Plain Format = "plain"
	Fancy Format = "fancy"
)

func (e *Format) String() string {
	return string(*e)
}
func (e *Format) Help() string {
	return "Sets output type. allowed: json|plain|fancy"
}

func (e *Format) Set(v string) error {
	switch v {
	case "json", "plain", "fancy":
		*e = Format(v)
	default:
		return fmt.Errorf("invalid value %q", v)
	}
	return nil
}

func (e *Format) Type() string {
	return "Format"
}
