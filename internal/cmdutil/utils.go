package cmdutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/browser"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Errorf prints formatted error in stderr and exits.
func Errorf(msg string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", msg), a...)
	os.Exit(1)
}

// ExitIfError exists with error message if err is not nil.
func ExitIfError(err error) {
	if err != nil {
		var msg string

		switch err {
		case jira.ErrUnexpectedStatusCode:
			msg = "jira: Received unexpected response.\nPlease check the parameters you supplied and try again."
		case jira.ErrEmptyResponse:
			msg = "jira: Received empty response.\nPlease try again."
		default:
			msg = err.Error()
		}

		Errorf("\nError: %s", msg)
	}
}

// Info displays spinner.
func Info(msg string) *spinner.Spinner {
	const refreshRate = 100 * time.Millisecond

	s := spinner.New(
		spinner.CharSets[14],
		refreshRate,
		spinner.WithSuffix(" "+msg),
		spinner.WithHiddenCursor(true),
		spinner.WithWriter(color.Error),
	)
	s.Start()

	return s
}

// Navigate navigates to jira issue.
func Navigate(server, path string) error {
	url := fmt.Sprintf("%s/browse/%s", server, path)
	return browser.OpenURL(url)
}

// FormatDateTimeHuman formats date time in human readable format.
func FormatDateTimeHuman(dt, format string) string {
	t, err := time.Parse(format, dt)
	if err != nil {
		return dt
	}
	return t.Format("Mon, 02 Jan 06")
}

// GetConfigHome returns the config home directory.
func GetConfigHome() (string, error) {
	home := os.Getenv("XDG_CONFIG_HOME")
	if home != "" {
		return home, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return home + "/.config", nil
}

// StdinHasData checks if standard input has any data to be processed.
func StdinHasData() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return false
	}
	return true
}

// ReadFile reads contents of the given file.
func ReadFile(filePath string) ([]byte, error) {
	if filePath != "-" && filePath != "" {
		return ioutil.ReadFile(filePath)
	}
	if filePath == "-" || StdinHasData() {
		b, err := ioutil.ReadAll(os.Stdin)
		_ = os.Stdin.Close()
		return b, err
	}
	return []byte(""), nil
}
