package tui

import (
	"bufio"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func pad(in string, n uint) string {
	if in == "" {
		return in
	}

	var (
		i   uint
		out strings.Builder
	)

	for i = 0; i < n; i++ {
		out.WriteString(" ")
	}

	out.WriteString(in)

	for i = 0; i < n; i++ {
		out.WriteString(" ")
	}

	return out.String()
}

func splitText(s string) []string {
	var lines []string

	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	return lines
}

func renderTableHeader(t *Table, data []string) {
	style := tcell.StyleDefault.Bold(true).Background(tcell.ColorDarkCyan)

	for c := 0; c < len(data); c++ {
		text := " " + data[c]

		cell := tview.NewTableCell(text).
			SetStyle(style).
			SetSelectable(false).
			SetMaxWidth(int(t.maxColWidth))

		t.view.SetCell(0, c, cell)
	}
}

func renderTableCell(t *Table, data [][]string) {
	rows, cols := len(data), len(data[0])

	for r := 1; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell := tview.NewTableCell(pad(data[r][c], t.colPad)).
				SetMaxWidth(int(t.maxColWidth)).
				SetStyle(tcell.StyleDefault)

			t.view.SetCell(r, c, cell)
		}
	}
}
