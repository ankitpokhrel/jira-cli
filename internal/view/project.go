package view

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// ProjectOption is a functional option to wrap project properties.
type ProjectOption func(*Project)

// Project is a project view.
type Project struct {
	data   []*jira.Project
	writer io.Writer
	buf    *bytes.Buffer
}

// NewProject initializes a project.
func NewProject(data []*jira.Project, opts ...ProjectOption) *Project {
	p := Project{
		data: data,
		buf:  new(bytes.Buffer),
	}
	p.writer = tabwriter.NewWriter(p.buf, 0, 8, 1, '\t', 0)

	for _, opt := range opts {
		opt(&p)
	}
	return &p
}

// WithProjectWriter sets a writer for the project.
func WithProjectWriter(w io.Writer) ProjectOption {
	return func(p *Project) {
		p.writer = w
	}
}

// Render renders the project view.
func (p Project) Render() error {
	p.printHeader()

	for _, d := range p.data {
		fmt.Fprintf(p.writer, "%s\t%s\t%s\n", d.Key, prepareTitle(d.Name), d.Lead.Name)
	}
	if _, ok := p.writer.(*tabwriter.Writer); ok {
		err := p.writer.(*tabwriter.Writer).Flush()
		if err != nil {
			return err
		}
	}

	return PagerOut(p.buf.String())
}

func (p Project) header() []string {
	return []string{
		"KEY",
		"NAME",
		"LEAD",
	}
}

func (p Project) printHeader() {
	n := len(p.header())
	for i, h := range p.header() {
		fmt.Fprintf(p.writer, "%s", h)
		if i != n-1 {
			fmt.Fprintf(p.writer, "\t")
		}
	}
	fmt.Fprintln(p.writer)
}
