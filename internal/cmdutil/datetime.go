package cmdutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Supported datetime layouts.
const (
	DateLayout          = "2006-01-02"
	DateTimeLayout      = "2006-01-02 15:04:05"
	ShortDateLayout     = "20060102"
	ShortDateTimeLayout = "20060102150405"
)

var (
	// ErrInvlalidTimezone is returned if timezone is not in a valid IANA timezone format.
	ErrInvlalidTimezone = fmt.Errorf("timezone should be a valid IANA timezone, eg: Asia/Kathmandu, Europe/Berlin etc")
	// ErrorInvalidDateTime is returned when datetime string is invalid.
	ErrorInvalidDateTime = fmt.Errorf("datetime string should be in a valid format, eg: 2022-01-02 10:10:05 or 2022-01-02")
)

// DateStringToJiraFormatInLocation parses a standard string to jira compatible RFC3339 datetime format.
//
//nolint:gomnd
func DateStringToJiraFormatInLocation(value string, timezone string) (string, error) {
	if value == "" || value == "0" || value == "0000-00-00 00:00:00" || value == "0000-00-00" || value == "00:00:00" {
		return "", nil
	}

	if _, err := time.Parse(jira.RFC3339MilliLayout, value); err == nil {
		return value, nil
	}

	layout := DateTimeLayout
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		switch {
		case len(value) == 8:
			layout = ShortDateLayout
		case len(value) == 14:
			layout = ShortDateTimeLayout
		}
	} else if len(value) == 10 && strings.Count(value, "-") == 2 {
		layout = DateLayout
	}

	t, err := parseByLayout(layout, value, timezone)
	if err != nil {
		return "", err
	}
	return t.Format(jira.RFC3339MilliLayout), nil
}

// parseByLayout parses a string as a time by layout in given timezone.
func parseByLayout(layout, value string, timezone string) (*time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, ErrInvlalidTimezone
	}
	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		return nil, ErrorInvalidDateTime
	}
	return &t, nil
}
