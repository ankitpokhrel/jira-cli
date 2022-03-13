package envrc

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type netrcLine struct {
	machine  string
	login    string
	password string
}

// ReadEnvrcPassword retrieves password for the desired server and login.
func ReadEnvrcPassword(serverURL string, login string) (string, error) {
	URL, err := url.ParseRequestURI(serverURL)
	if err != nil {
		return "", err
	}

	netrcLines, err := readNetrc()
	if err != nil {
		return "", err
	}

	for _, line := range netrcLines {
		if line.machine == URL.Host && line.login == login {
			return line.password, nil
		}
	}

	return "", errors.New("envrc token not found")
}

func readNetrc() ([]netrcLine, error) {
	path, err := netrcPath()
	if err != nil {
		return []netrcLine{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return []netrcLine{}, err
		}
		return []netrcLine{}, err
	}

	netrc := parseNetrc(string(data))
	return netrc, err
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

func parseNetrc(data string) []netrcLine {
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
