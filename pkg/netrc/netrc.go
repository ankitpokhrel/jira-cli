// Package netrc implements GNU .netrc specification.
// This implementation is borrowed from the original implementation by the go authors.
// See https://github.com/golang/go/blob/master/src/cmd/go/internal/auth/netrc.go
// See https://www.gnu.org/software/inetutils/manual/html_node/The-_002enetrc-file.html
package netrc

import (
	"fmt"
	"net/url"
)

// ErrNetrcEntryNotFound is thrown if details for the machine is not found.
var ErrNetrcEntryNotFound = fmt.Errorf("netrc config: entry not found")

// Entry is a netrc config entry.
type Entry struct {
	Machine  string
	Login    string
	Password string
}

// Read reads config for the given machine.
func Read(machine string, login string) (*Entry, error) {
	netrcOnce.Do(readNetrc)
	if netrcErr != nil {
		return nil, netrcErr
	}

	serverURL, err := url.ParseRequestURI(machine)
	if err != nil {
		return nil, err
	}

	for _, line := range netrc {
		if line.machine == serverURL.Host && line.login == login {
			return &Entry{
				Machine:  line.machine,
				Login:    line.login,
				Password: line.password,
			}, nil
		}
	}

	return nil, ErrNetrcEntryNotFound
}
