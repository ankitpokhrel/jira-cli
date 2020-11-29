package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultColPad   = 1
	defaultColWidth = 50
)

var errNoData = fmt.Errorf("no data")

// TableData is the data to be displayed in a table.
type TableData [][]string

// Table is a table layout.
type Table struct {
	screen      *Screen
	painter     *tview.Grid
	view        *tview.Table
	colPad      uint
	maxColWidth uint
	footerText  string
}

// TableOption is a functional option to wrap table properties.
type TableOption func(*Table)

// NewTable returns new table layout.
func NewTable(s *Screen, opts ...TableOption) *Table {
	tbl := Table{
		screen:      s,
		colPad:      defaultColPad,
		maxColWidth: defaultColWidth,
	}

	for _, opt := range opts {
		opt(&tbl)
	}

	tableView := tview.NewTable()
	footerView := tview.NewTextView().SetWordWrap(true)

	initTableView(s, tableView)
	initFooterView(footerView, tbl.footerText)

	tbl.painter = tview.NewGrid().
		SetRows(0, 1, 2).
		AddItem(tableView, 0, 0, 1, 1, 3, 0, true).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 1, 1, false). // Dummy view to fake row padding.
		AddItem(footerView, 2, 0, 1, 1, 1, 1, false)

	tbl.view = tableView

	return &tbl
}

func initFooterView(view *tview.TextView, text string) {
	view.SetText(pad(text, 1)).SetTextColor(tcell.ColorAntiqueWhite)
}

func initTableView(s *Screen, view *tview.Table) {
	view.SetSelectable(true, false)

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

	view.SetFixed(1, 1)
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

// WithFooterText sets footer text that is displayed after the table.
func WithFooterText(text string) TableOption {
	return func(t *Table) {
		t.footerText = text
	}
}

func (t *Table) renderHeader(data []string) {
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

// Render renders the table layout. First row is treated as a table header.
func (t *Table) Render(data [][]string) error {
	if len(data) == 0 {
		return errNoData
	}

	rows, cols := len(data), len(data[0])

	t.renderHeader(data[0])

	for r := 1; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell := tview.NewTableCell(pad(data[r][c], t.colPad)).
				SetMaxWidth(int(t.maxColWidth)).
				SetStyle(tcell.StyleDefault)

			t.view.SetCell(r, c, cell)
		}
	}

	return t.screen.SetRoot(t.painter, true).SetFocus(t.painter).Run()
}

func pad(in string, n uint) string {
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
