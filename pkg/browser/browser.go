package browser

import (
	"bytes"
	"cmp"
	"os"
	"os/exec"

	"github.com/google/shlex"
	"github.com/pkg/browser"
)

// Browse opens given url in a web browser.
//
// It looks for `JIRA_BROWSER` and `BROWSER` env respectively to decide which
// executable to use. If none of them are set, the default browser is invoked.
func Browse(url string) error {
	opener := getBrowserFromENV()

	if opener == "" {
		// Launch default browser.
		return browser.OpenURL(url)
	}

	args, err := shlex.Split(opener)
	if err != nil {
		return err
	}
	exe, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}

	args = append(args[1:], url)
	// No context to hang this on: the browser is handed the URL and outlives us.
	cmd := exec.Command(exe, args...) //nolint:noctx

	// io.Writer to which executed commands write standard output and error.
	// We are not interested in any output from cmd, so let's discard them.
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	return cmd.Run()
}

func getBrowserFromENV() string {
	return cmp.Or(os.Getenv("JIRA_BROWSER"), os.Getenv("BROWSER"))
}
