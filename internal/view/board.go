package view

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Board is a board view.
type Board struct {
	Data []*jira.Board
}

func (b Board) header() []string {
	return []string{
		"ID",
		"NAME",
		"TYPE",
	}
}

func (b Board) printHeader(w io.Writer) {
	for _, h := range b.header() {
		_, _ = fmt.Fprintf(w, "%s\t", h)
	}

	_, _ = fmt.Fprintln(w, "")
}

// Render renders the board view.
func (b Board) Render() error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	b.printHeader(w)

	for _, d := range b.Data {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", d.ID, d.Name, d.Type)
	}

	return w.Flush()
}
