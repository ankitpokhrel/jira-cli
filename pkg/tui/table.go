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

// SelectedFunc is fired when a user press enter key in the table cell.
type SelectedFunc func(row, column int, data *TableData)

// TableData is the data to be displayed in a table.
type TableData [][]string

// Table is a table layout.
type Table struct {
	screen       *Screen
	painter      *tview.Grid
	view         *tview.Table
	footer       *tview.TextView
	colPad       uint
	maxColWidth  uint
	footerText   string
	selectedFunc SelectedFunc
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

	tbl.initTableView()
	tbl.initFooterView()

	tbl.painter = tview.NewGrid().
		SetRows(0, 1, 2).
		AddItem(tbl.view, 0, 0, 1, 1, 0, 0, true).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 0, 0, false). // Dummy view to fake row padding.
		AddItem(tbl.footer, 2, 0, 1, 1, 0, 0, false)

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

// WithFooterText sets footer text that is displayed after the table.
func WithFooterText(text string) TableOption {
	return func(t *Table) {
		t.footerText = text
	}
}

// WithSelectedFunc sets a func that is triggered when table cell is selected.
func WithSelectedFunc(fn SelectedFunc) TableOption {
	return func(t *Table) {
		t.selectedFunc = fn
	}
}

func (t *Table) initFooterView() {
	view := tview.NewTextView().SetWordWrap(true)

	view.SetText(pad(t.footerText, 1)).SetTextColor(tcell.ColorAntiqueWhite)

	t.footer = view
}

func (t *Table) initTableView() {
	view := tview.NewTable()

	view.SetSelectable(true, false)

	view.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			t.screen.Stop()
		}
	})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			t.screen.Stop()
		}

		return event
	})

	view.SetFixed(1, 1)

	t.view = view
}

// Render renders the table layout. First row is treated as a table header.
func (t *Table) Render(data TableData) error {
	if len(data) == 0 {
		return errNoData
	}

	if t.selectedFunc != nil {
		t.view.SetSelectedFunc(func(r, c int) {
			t.selectedFunc(r, c, &data)
		})
	}

	renderTableHeader(t, data[0])
	renderTableCell(t, data)

	return t.screen.Paint(t.painter)
}
