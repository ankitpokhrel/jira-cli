package netrc

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	netrcOnce sync.Once
	netrc     []netrcLine
	netrcErr  error
)

// ReadNetrcPassword retrieves password for the desired server and login.
// We just borrowed original code due to its internal nature, it is original implementation of .netrc read in Go
// https://github.com/golang/go/blob/master/src/cmd/go/internal/auth/netrc.go
func ReadNetrcPassword(jiraServer string, login string) (string, error) {
	netrcOnce.Do(func() {
		readNetrc()
	})
	if netrcErr != nil {
		return "", netrcErr
	}

	jiraServerURL, err := url.ParseRequestURI(jiraServer)
	if err != nil {
		return "", err
	}

	for _, line := range netrc {
		if line.machine == jiraServerURL.Host && line.login == login {
			return line.password, nil
		}
	}

	return "", errors.New("netrc token not found")
}

type netrcLine struct {
	machine  string
	login    string
	password string
}

func parseNetrc(data string) []netrcLine {
	// See https://www.gnu.org/software/inetutils/manual/html_node/The-_002enetrc-file.html
	// for documentation on the .netrc format.
	var nrc []netrcLine
	var l netrcLine
	inMacro := false
	for _, line := range strings.Split(data, "\n") {
		if inMacro {
			if line == "" {
				inMacro = false
			}
			continue
		}

		f := strings.Fields(line)
		i := 0
		for ; i < len(f)-1; i += 2 {
			// Reset at each "machine" token.
			// “The auto-login process searches the .netrc file for a machine token
			// that matches […]. Once a match is made, the subsequent .netrc tokens
			// are processed, stopping when the end of file is reached or another
			// machine or a default token is encountered.”
			switch f[i] {
			case "machine":
				l = netrcLine{machine: f[i+1]}
			// Commenting redundant case default below to avoid CI failures
			// case "default":
			//	break
			case "login":
				l.login = f[i+1]
			case "password":
				l.password = f[i+1]
			case "macdef":
				// “A macro is defined with the specified name; its contents begin with
				// the next .netrc line and continue until a null line (consecutive
				// new-line characters) is encountered.”
				inMacro = true
			}
			if l.machine != "" && l.login != "" && l.password != "" {
				nrc = append(nrc, l)
				l = netrcLine{}
			}
		}

		if i < len(f) && f[i] == "default" {
			// “There can be only one default token, and it must be after all machine tokens.”
			break
		}
	}

	return nrc
}

func netrcPath() (string, error) {
	if env := os.Getenv("NETRC"); env != "" {
		return env, nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := ".netrc"
	if runtime.GOOS == "windows" {
		base = "_netrc"
	}
	return filepath.Join(dir, base), nil
}

func readNetrc() {
	path, err := netrcPath()
	if err != nil {
		netrcErr = err
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			netrcErr = err
		}
		return
	}

	netrc = parseNetrc(string(data))
}
