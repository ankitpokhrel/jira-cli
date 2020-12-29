package tui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultColPad   = 1
	defaultColWidth = 50
)

var errNoData = fmt.Errorf("no data")

// SelectedFunc is fired when a user press enter key in the table cell.
type SelectedFunc func(row, column int, data interface{})

// ViewModeFunc sets view mode handler func which gets triggered when a user press 'v'.
type ViewModeFunc func(row, column int, data interface{}) error

// TableData is the data to be displayed in a table.
type TableData [][]string

// Table is a table layout.
type Table struct {
	screen       *Screen
	painter      tview.Primitive
	view         *tview.Table
	footer       *tview.TextView
	data         TableData
	colPad       uint
	maxColWidth  uint
	footerText   string
	selectedFunc SelectedFunc
	viewModeFunc ViewModeFunc
}

// TableOption is a functional option to wrap table properties.
type TableOption func(*Table)

// NewTable constructs a new table layout.
func NewTable(opts ...TableOption) *Table {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

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

// WithTableFooterText sets footer text that is displayed after the table.
func WithTableFooterText(text string) TableOption {
	return func(t *Table) {
		t.footerText = text
	}
}

// WithSelectedFunc sets a func that is triggered when table row is selected.
func WithSelectedFunc(fn SelectedFunc) TableOption {
	return func(t *Table) {
		t.selectedFunc = fn
	}
}

// WithViewModeFunc sets a func that is triggered when user a press 'v'.
func WithViewModeFunc(fn ViewModeFunc) TableOption {
	return func(t *Table) {
		t.viewModeFunc = fn
	}
}

// Paint paints the table layout. First row is treated as a table header.
func (t *Table) Paint(data TableData) error {
	if len(data) == 0 {
		return errNoData
	}
	t.data = data
	t.render(data)
	return t.screen.Paint(t.painter)
}

func (t *Table) render(data TableData) {
	if t.selectedFunc != nil {
		t.view.SetSelectedFunc(func(r, c int) {
			t.selectedFunc(r, c, data)
		})
	}
	renderTableHeader(t, data[0])
	renderTableCell(t, data)
}

func (t *Table) initFooterView() {
	view := tview.NewTextView().
		SetWordWrap(true).
		SetText(pad(t.footerText, 1)).
		SetTextColor(tcell.ColorDefault)

	t.footer = view
}

func (t *Table) initTableView() {
	view := tview.NewTable()

	view.SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Bold(true).Dim(true))

	view.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			t.screen.Stop()
		}
	}).SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyRune {
			switch ev.Rune() {
			case 'q':
				t.screen.Stop()
				os.Exit(0)
			case 'v':
				r, c := t.view.GetSelection()
				t.screen.Suspend(func() { _ = t.viewModeFunc(r, c, t.data) })
			}
		}
		return ev
	})

	view.SetFixed(1, 1)

	t.view = view
}

func renderTableHeader(t *Table, data []string) {
	style := tcell.StyleDefault.Bold(true).Background(tcell.ColorDarkCyan)

	for c := 0; c < len(data); c++ {
		text := " " + data[c]

		cell := tview.NewTableCell(text).
			SetStyle(style).
			SetSelectable(false).
			SetMaxWidth(int(t.maxColWidth)).
			SetTextColor(tcell.ColorSnow)

		t.view.SetCell(0, c, cell)
	}
}

func renderTableCell(t *Table, data [][]string) {
	rows, cols := len(data), len(data[0])

	for r := 1; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell := tview.NewTableCell(pad(data[r][c], t.colPad)).
				SetMaxWidth(int(t.maxColWidth)).
				SetTextColor(tcell.ColorDefault)

			t.view.SetCell(r, c, cell)
		}
	}
}
