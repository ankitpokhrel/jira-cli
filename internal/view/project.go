package view

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// ProjectOption is a functional option to wrap board properties.
type ProjectOption func(*Project)

// Project is a project view.
type Project struct {
	data   []*jira.Project
	writer io.Writer
}

// NewProject initializes a project.
func NewProject(data []*jira.Project, opts ...ProjectOption) *Project {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	p := Project{
		data:   data,
		writer: w,
	}

	for _, opt := range opts {
		opt(&p)
	}

	return &p
}

// WithProjectWriter sets a writer for board.
func WithProjectWriter(w io.Writer) ProjectOption {
	return func(p *Project) {
		p.writer = w
	}
}

func (p Project) header() []string {
	return []string{
		"KEY",
		"NAME",
		"LEAD",
	}
}

func (p Project) printHeader() {
	for _, h := range p.header() {
		_, _ = fmt.Fprintf(p.writer, "%s\t", h)
	}

	_, _ = fmt.Fprintln(p.writer, "")
}

// Render renders the project view.
func (p Project) Render() error {
	p.printHeader()

	for _, d := range p.data {
		_, _ = fmt.Fprintf(p.writer, "%s\t%s\t%s\n", d.Key, prepareTitle(d.Name), d.Lead.Name)
	}

	if _, ok := p.writer.(*tabwriter.Writer); ok {
		return p.writer.(*tabwriter.Writer).Flush()
	}

	return nil
}
