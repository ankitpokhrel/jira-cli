package cmdutil

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const refreshRate = 100 * time.Millisecond

// ExitWithErrMessage exits after printing given message in stderr.
func ExitWithErrMessage(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
}

// ExitIfError exists with error message if err is not nil.
func ExitIfError(err error) {
	if err != nil {
		var msg string

		switch err {
		case jira.ErrUnexpectedStatusCode:
			msg = "Received unexpected response code from jira. Please check the parameters you supplied and try again."
		case jira.ErrEmptyResponse:
			msg = "Received empty response from jira. Please try again."
		default:
			msg = err.Error()
		}

		ExitWithErrMessage(fmt.Sprintf("Error: %s", msg))
	}
}

// Info displays spinner.
func Info(msg string) *spinner.Spinner {
	s := spinner.New(
		spinner.CharSets[14],
		refreshRate,
		spinner.WithSuffix(" "+msg),
		spinner.WithHiddenCursor(true),
	)

	s.Start()

	return s
}

// PrintErrF prints formatted error in stderr.
func PrintErrF(msg string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", msg), a...)
}
