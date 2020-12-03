package tui

import (
	"fmt"

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
func NewTable(opts ...TableOption) *Table {
	tbl := Table{
		screen:      NewScreen(),
		colPad:      defaultColPad,
		maxColWidth: defaultColWidth,
	}

	for _, opt := range opts {
		opt(&tbl)
	}

	tableView := tview.NewTable()
	footerView := tview.NewTextView().SetWordWrap(true)

	initTableView(tbl.screen, tableView)
	initFooterView(footerView, tbl.footerText)

	tbl.painter = tview.NewGrid().
		SetRows(0, 1, 2).
		AddItem(tableView, 0, 0, 1, 1, 0, 0, true).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 0, 0, false). // Dummy view to fake row padding.
		AddItem(footerView, 2, 0, 1, 1, 0, 0, false)

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

// Render renders the table layout. First row is treated as a table header.
func (t *Table) Render(data [][]string) error {
	if len(data) == 0 {
		return errNoData
	}

	renderTableHeader(t, data[0])
	renderTableCell(t, data)

	return t.screen.Paint(t.painter)
}
