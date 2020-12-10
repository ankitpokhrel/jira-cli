package view

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// BoardOption is a functional option to wrap board properties.
type BoardOption func(*Board)

// Board is a board view.
type Board struct {
	data   []*jira.Board
	writer io.Writer
}

// NewBoard initializes a board.
func NewBoard(data []*jira.Board, opts ...BoardOption) *Board {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	b := Board{
		data:   data,
		writer: w,
	}

	for _, opt := range opts {
		opt(&b)
	}

	return &b
}

// WithBoardWriter sets a writer for board.
func WithBoardWriter(w io.Writer) BoardOption {
	return func(b *Board) {
		b.writer = w
	}
}

func (b Board) header() []string {
	return []string{
		"ID",
		"NAME",
		"TYPE",
	}
}

func (b Board) printHeader() {
	for _, h := range b.header() {
		_, _ = fmt.Fprintf(b.writer, "%s\t", h)
	}

	_, _ = fmt.Fprintln(b.writer, "")
}

// Render renders the board view.
func (b Board) Render() error {
	b.printHeader()

	for _, d := range b.data {
		_, _ = fmt.Fprintf(b.writer, "%d\t%s\t%s\n", d.ID, prepareTitle(d.Name), d.Type)
	}

	if _, ok := b.writer.(*tabwriter.Writer); ok {
		return b.writer.(*tabwriter.Writer).Flush()
	}

	return nil
}
