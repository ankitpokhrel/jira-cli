package view

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

// Attachment is an attachment view.
type Attachment struct {
	Data   []jira.Attachment
	Writer io.Writer
	buf    *bytes.Buffer
}

// NewAttachment initializes an attachment view.
func NewAttachment(data []jira.Attachment) *Attachment {
	a := Attachment{
		Data: data,
		buf:  new(bytes.Buffer),
	}
	a.Writer = tabwriter.NewWriter(a.buf, 0, tabWidth, 1, '\t', 0)

	return &a
}

// Render renders the attachment view.
func (a *Attachment) Render() error {
	a.printHeader()

	for _, d := range a.Data {
		size := cmdutil.FormatBytes(int64(d.Size))
		created := cmdutil.FormatDateTimeHuman(d.Created, jira.RFC3339)
		_, _ = fmt.Fprintf(a.Writer, "%s\t%s\t%s\n", d.Filename, size, created)
	}

	if _, ok := a.Writer.(*tabwriter.Writer); ok {
		if err := a.Writer.(*tabwriter.Writer).Flush(); err != nil {
			return err
		}
	}

	return tui.PagerOut(a.buf.String())
}

func attachmentHeader() []string {
	return []string{
		"NAME",
		"SIZE",
		"CREATED",
	}
}

func (a *Attachment) printHeader() {
	header := attachmentHeader()
	n := len(header)
	for i, h := range header {
		_, _ = fmt.Fprintf(a.Writer, "%s", h)
		if i != n-1 {
			_, _ = fmt.Fprintf(a.Writer, "\t")
		}
	}
	_, _ = fmt.Fprintln(a.Writer, "")
}
