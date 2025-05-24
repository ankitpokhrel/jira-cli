package view

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// ProjectVersionOptions is a functional option to wrap project version properties.
type ProjectVersionOptions func(*Release)

// Release is a release view.
type Release struct {
	data   []*jira.ProjectVersion
	writer io.Writer
	buf    *bytes.Buffer
}

// NewRelease constructs a project release command.
func NewRelease(data []*jira.ProjectVersion, opts ...ProjectVersionOptions) *Release {
	r := Release{
		data: data,
		buf:  new(bytes.Buffer),
	}
	r.writer = tabwriter.NewWriter(r.buf, 0, tabWidth, 1, '\t', 0)

	for _, opt := range opts {
		opt(&r)
	}
	return &r
}

// WithReleaseWriter sets a writer for the project release.
func WithReleaseWriter(w io.Writer) ProjectVersionOptions {
	return func(r *Release) {
		r.writer = w
	}
}

// Render renders the project release view.
func (r Release) Render() error {
	r.printHeader()

	for _, d := range r.data {
		desc := ""
		if d.Description != nil {
			desc = fmt.Sprint(d.Description)
		}
		_, _ = fmt.Fprintf(r.writer, "%v\t%v\t%v\t%v\n", d.ID, prepareTitle(d.Name), d.Released, desc)
	}
	if _, ok := r.writer.(*tabwriter.Writer); ok {
		err := r.writer.(*tabwriter.Writer).Flush()
		if err != nil {
			return err
		}
	}

	return tui.PagerOut(r.buf.String())
}

func (r Release) header() []string {
	return []string{
		"ID",
		"NAME",
		"RELEASED",
		"DESCRIPTION",
	}
}

func (r Release) printHeader() {
	headers := r.header()
	end := len(headers) - 1
	for i, h := range headers {
		_, _ = fmt.Fprintf(r.writer, "%s", h)
		if i != end {
			_, _ = fmt.Fprintf(r.writer, "\t")
		}
	}
	_, _ = fmt.Fprintln(r.writer)
}
