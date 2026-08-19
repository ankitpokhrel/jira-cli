package view

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// ComponentOption is a functional option to wrap component properties.
type ComponentOption func(*Component)

// Component is a project component view.
type Component struct {
	data   []*jira.ProjectComponent
	writer io.Writer
	buf    *bytes.Buffer
}

// NewComponent initializes a component view.
func NewComponent(data []*jira.ProjectComponent, opts ...ComponentOption) *Component {
	c := Component{
		data: data,
		buf:  new(bytes.Buffer),
	}
	c.writer = tabwriter.NewWriter(c.buf, 0, tabWidth, 1, '\t', 0)

	for _, opt := range opts {
		opt(&c)
	}
	return &c
}

// WithComponentWriter sets a writer for the component.
func WithComponentWriter(w io.Writer) ComponentOption {
	return func(c *Component) {
		c.writer = w
	}
}

// Render renders the component view.
func (c Component) Render() error {
	c.printHeader()

	for _, d := range c.data {
		desc := ""
		if d.Description != nil {
			desc = fmt.Sprint(d.Description)
		}
		_, _ = fmt.Fprintf(c.writer, "%v\t%v\t%v\n", d.ID, prepareTitle(d.Name), desc)
	}
	if _, ok := c.writer.(*tabwriter.Writer); ok {
		err := c.writer.(*tabwriter.Writer).Flush()
		if err != nil {
			return err
		}
	}

	return tui.PagerOut(c.buf.String())
}

func (c Component) header() []string {
	return []string{
		"ID",
		"NAME",
		"DESCRIPTION",
	}
}

func (c Component) printHeader() {
	headers := c.header()
	end := len(headers) - 1
	for i, h := range headers {
		_, _ = fmt.Fprintf(c.writer, "%s", h)
		if i != end {
			_, _ = fmt.Fprintf(c.writer, "\t")
		}
	}
	_, _ = fmt.Fprintln(c.writer)
}
