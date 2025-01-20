package view

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// ServerInfoOption is a functional option to wrap serverinfo properties.
type ServerInfoOption func(*ServerInfo)

// ServerInfo is a serveronfo view.
type ServerInfo struct {
	data   *jira.ServerInfo
	writer io.Writer
	buf    *bytes.Buffer
}

// NewServerInfo initializes server info struct.
func NewServerInfo(data *jira.ServerInfo, opts ...ServerInfoOption) *ServerInfo {
	s := ServerInfo{
		data: data,
		buf:  new(bytes.Buffer),
	}
	s.writer = tabwriter.NewWriter(s.buf, 0, tabWidth, 1, '\t', 0)

	for _, opt := range opts {
		opt(&s)
	}
	return &s
}

// WithServerInfoWriter sets a writer for the serverinfo view.
func WithServerInfoWriter(w io.Writer) ServerInfoOption {
	return func(s *ServerInfo) {
		s.writer = w
	}
}

// Render renders the serverinfo view.
func (s ServerInfo) Render() error {
	_, _ = fmt.Fprintf(s.writer, `SERVER INFO
-----------

Version: 	 %s
Build Number: 	 %d
Deployment Type: %s
Default Locale:  %s
`, s.data.Version, s.data.BuildNumber, s.data.DeploymentType, s.data.DefaultLocale.Locale)

	return tui.PagerOut(s.buf.String())
}
