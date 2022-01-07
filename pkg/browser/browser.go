package browser

import (
	"bytes"
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

	openerArgs, err := shlex.Split(opener)
	if err != nil {
		return err
	}
	openerExe, err := exec.LookPath(openerArgs[0])
	if err != nil {
		return err
	}

	args := append(openerArgs[1:], url) //nolint:gocritic
	cmd := exec.Command(openerExe, args...)

	// io.Writer to which executed commands write standard output and error.
	// We are not interested in any output from cmd, so let's discard them.
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	return cmd.Run()
}

func getBrowserFromENV() string {
	br := os.Getenv("JIRA_BROWSER")
	if br == "" {
		br = os.Getenv("BROWSER")
	}
	return br
}
