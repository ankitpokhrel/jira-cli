package view

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// BoardOption is a functional option to wrap board properties.
type BoardOption func(*Board)

// Board is a board view.
type Board struct {
	data   []*jira.Board
	writer io.Writer
	buf    *bytes.Buffer
}

// NewBoard initializes a board.
func NewBoard(data []*jira.Board, opts ...BoardOption) *Board {
	b := Board{
		data: data,
		buf:  new(bytes.Buffer),
	}
	b.writer = tabwriter.NewWriter(b.buf, 0, tabWidth, 1, '\t', 0)

	for _, opt := range opts {
		opt(&b)
	}
	return &b
}

// WithBoardWriter sets a writer for the board.
func WithBoardWriter(w io.Writer) BoardOption {
	return func(b *Board) {
		b.writer = w
	}
}

// Render renders the board view.
func (b Board) Render() error {
	b.printHeader()

	for _, d := range b.data {
		fmt.Fprintf(b.writer, "%d\t%s\t%s\n", d.ID, prepareTitle(d.Name), d.Type)
	}
	if _, ok := b.writer.(*tabwriter.Writer); ok {
		err := b.writer.(*tabwriter.Writer).Flush()
		if err != nil {
			return err
		}
	}

	return PagerOut(b.buf.String())
}

func (b Board) header() []string {
	return []string{
		"ID",
		"NAME",
		"TYPE",
	}
}

func (b Board) printHeader() {
	n := len(b.header())
	for i, h := range b.header() {
		fmt.Fprintf(b.writer, "%s", h)
		if i != n-1 {
			fmt.Fprintf(b.writer, "\t")
		}
	}
	fmt.Fprintln(b.writer, "")
}
