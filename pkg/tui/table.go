package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultColPad   = 1
	defaultColWidth = 50
)

// TableData is the data to be displayed in a table.
type TableData [][]string

// Table is a table layout.
type Table struct {
	screen      *Screen
	painter     *tview.Table
	colPad      uint
	maxColWidth uint
}

// TableOption is a functional option to wrap table properties.
type TableOption func(*Table)

// NewTable returns new table layout.
func NewTable(s *Screen, opts ...TableOption) *Table {
	view := tview.NewTable()

	view.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			s.Stop()
		}
	})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			s.Stop()
		}

		return event
	})

	tbl := Table{
		screen:      s,
		painter:     view,
		colPad:      defaultColPad,
		maxColWidth: defaultColWidth,
	}

	for _, opt := range opts {
		opt(&tbl)
	}

	return &tbl
}

// WithColPadding sets column padding property of the table.
func WithColPadding(pad uint) TableOption {
	return func(t *Table) {
		t.colPad = pad
	}
}

// WithMaxColWidth sets max column width property of the table.
func WithMaxColWidth(width uint) TableOption {
	return func(t *Table) {
		t.maxColWidth = width
	}
}

// Render renders the table layout. First row is treated as a header.
func (t *Table) Render(data [][]string) error {
	rows, cols := len(data), len(data[0])

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			style := tcell.StyleDefault

			if r == 0 {
				style = style.Bold(true).Background(tcell.ColorDarkCyan)
			}

			var text string

			if c != 0 {
				text = t.applyTextPadding(data[r][c])
			} else {
				text = " " + data[r][c]
			}

			cell := tview.NewTableCell(text).SetMaxWidth(int(t.maxColWidth)).SetStyle(style)
			t.painter.SetCell(r, c, cell)
		}
	}

	return t.screen.SetRoot(t.painter, true).SetFocus(t.painter).Run()
}

func (t *Table) applyTextPadding(in string) string {
	var (
		i   uint
		out strings.Builder
	)

	for i = 0; i < t.colPad; i++ {
		out.WriteString(" ")
	}

	out.WriteString(in)

	for i = 0; i < t.colPad; i++ {
		out.WriteString(" ")
	}

	return out.String()
}
