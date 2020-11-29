package view

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Project is a project view.
type Project struct {
	Data []*jira.Project
}

func (p Project) header() []string {
	return []string{
		"KEY",
		"NAME",
		"LEAD",
	}
}

func (p Project) printHeader(w io.Writer) {
	for _, h := range p.header() {
		_, _ = fmt.Fprintf(w, "%s\t", h)
	}

	_, _ = fmt.Fprintln(w, "")
}

// Render renders the project view.
func (p Project) Render() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	p.printHeader(w)

	for _, p := range p.Data {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Key, p.Name, p.Lead.Name)
	}

	return w.Flush()
}
