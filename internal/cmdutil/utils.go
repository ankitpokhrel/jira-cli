package cmdutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"

	"github.com/ankitpokhrel/jira-cli/pkg/browser"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// ExitIfError exists with error message if err is not nil.
func ExitIfError(err error) {
	if err == nil {
		return
	}

	var msg string

	if e, ok := err.(*jira.ErrUnexpectedResponse); ok {
		dm := fmt.Sprintf(
			"\njira: Received unexpected response '%s'.\nPlease check the parameters you supplied and try again.",
			e.Status,
		)
		bd := e.Error()

		msg = dm
		if len(bd) > 0 {
			msg = fmt.Sprintf("%s%s", bd, dm)
		}
	} else if e, ok := err.(*jira.ErrMultipleFailed); ok {
		msg = fmt.Sprintf("\n%s%s", "SOME REQUESTS REPORTED ERROR:", e.Error())
	} else {
		switch err {
		case jira.ErrEmptyResponse:
			msg = "jira: Received empty response.\nPlease try again."
		default:
			msg = fmt.Sprintf("Error: %s", err.Error())
		}
	}

	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
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

// Success prints success message in stdout.
func Success(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, fmt.Sprintf("\n\u001B[0;32m✓\u001B[0m %s\n", msg), args...)
}

// Warn prints warning message in stderr.
func Warn(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("\u001B[0;33m%s\u001B[0m\n", msg), args...)
}

// Fail prints failure message in stderr.
func Fail(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("\u001B[0;31m✗\u001B[0m %s\n", msg), args...)
}

// Failed prints failure message in stderr and exits.
func Failed(msg string, args ...interface{}) {
	Fail(msg, args...)
	os.Exit(1)
}

// Navigate navigates to jira issue.
func Navigate(server, path string) error {
	url := fmt.Sprintf("%s/browse/%s", server, path)
	return browser.Browse(url)
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

// GetJiraIssueKey constructs actual issue key based on given key.
func GetJiraIssueKey(project, key string) string {
	if project == "" {
		return key
	}
	if _, err := strconv.Atoi(key); err != nil {
		return strings.ToUpper(key)
	}
	return fmt.Sprintf("%s-%s", project, key)
}

// NormalizeJiraError normalizes error message we receive from jira.
func NormalizeJiraError(msg string) string {
	msg = strings.TrimSpace(strings.Replace(msg, "Error:\n", "", 1))
	msg = strings.Replace(msg, "- ", "", 1)

	return msg
}

// GetSubtaskHandle fetches actual subtask handle.
// This value can either be handle or name based
// on the used jira version.
func GetSubtaskHandle(issueTypes []*jira.IssueType) string {
	handle := jira.IssueTypeSubTask
	for _, it := range issueTypes {
		if it.Subtask {
			if it.Handle != "" {
				handle = it.Handle
			} else {
				handle = it.Name
			}
		}
	}
	return handle
}
