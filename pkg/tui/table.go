package tui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultColPad   = 1
	defaultColWidth = 80
)

var errNoData = fmt.Errorf("no data")

// SelectedFunc is fired when a user press enter key in the table cell.
type SelectedFunc func(row, column int, data interface{})

// ViewModeFunc sets view mode handler func which gets triggered when a user press 'v'.
type ViewModeFunc func(row, col int, data interface{}) (func() interface{}, func(data interface{}) (string, error))

// RefreshFunc is fired when a user press 'CTRL+R' or `F5` character in the table.
type RefreshFunc func()

// CopyFunc is fired when a user press 'c' character in the table cell.
type CopyFunc func(row, column int, data interface{})

// CopyKeyFunc is fired when a user press 'CTRL+K' character in the table cell.
type CopyKeyFunc func(row, column int, data interface{})

// CustomKeyFunc is fired when one of custom keys is pressed in the table cell.
type CustomKeyFunc func(row, column int, data interface{}, command string)

// TableData is the data to be displayed in a table.
type TableData [][]string

// TableStyle sets the style of the table.
type TableStyle struct {
	SelectionBackground string
	SelectionForeground string
	SelectionTextIsBold bool
}

// Table is a table layout.
type Table struct {
	screen        *Screen
	painter       *tview.Pages
	view          *tview.Table
	footer        *tview.TextView
	style         TableStyle
	data          TableData
	colPad        uint
	colFixed      uint
	maxColWidth   uint
	footerText    string
	selectedFunc  SelectedFunc
	viewModeFunc  ViewModeFunc
	customKeys    map[string]interface{}
	customKeyFunc CustomKeyFunc
	refreshFunc   RefreshFunc
	copyFunc      CopyFunc
	copyKeyFunc   CopyKeyFunc
}

// TableOption is a functional option to wrap table properties.
type TableOption func(*Table)

// NewTable constructs a new table layout.
func NewTable(opts ...TableOption) *Table {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	tbl := Table{
		screen:      NewScreen(),
		view:        tview.NewTable(),
		footer:      tview.NewTextView(),
		colPad:      defaultColPad,
		maxColWidth: defaultColWidth,
	}
	for _, opt := range opts {
		opt(&tbl)
	}

	tbl.initTable()
	tbl.initFooter()

	grid := tview.NewGrid().
		SetRows(0, 1, 2).
		AddItem(tbl.view, 0, 0, 1, 1, 0, 0, true).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 0, 0, false). // Dummy view to fake row padding.
		AddItem(tbl.footer, 2, 0, 1, 1, 0, 0, false)

	tbl.painter = tview.NewPages().
		AddPage("primary", grid, true, true).
		AddPage("secondary", getInfoModal(), true, false)

	return &tbl
}

// WithTableStyle sets the style of the table.
func WithTableStyle(style TableStyle) TableOption {
	return func(t *Table) {
		t.style = style
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

// WithViewModeFunc sets a func that is triggered when a user press 'v'.
func WithViewModeFunc(fn ViewModeFunc) TableOption {
	return func(t *Table) {
		t.viewModeFunc = fn
	}
}

// WithCustomKeyFunc sets a func that is triggered when one of custom keys is pressed.
func WithCustomKeyFunc(fn CustomKeyFunc, keys map[string]interface{}) TableOption {
	return func(t *Table) {
		t.customKeyFunc = fn
		t.customKeys = keys
	}
}

// WithRefreshFunc sets a func that is triggered when a user press 'CTRL+R' or 'F5'.
func WithRefreshFunc(fn RefreshFunc) TableOption {
	return func(t *Table) {
		t.refreshFunc = fn
	}
}

// WithCopyFunc sets a func that is triggered when a user press 'c'.
func WithCopyFunc(fn CopyFunc) TableOption {
	return func(t *Table) {
		t.copyFunc = fn
	}
}

// WithCopyKeyFunc sets a func that is triggered when a user press 'CTRL+K'.
func WithCopyKeyFunc(fn CopyKeyFunc) TableOption {
	return func(t *Table) {
		t.copyKeyFunc = fn
	}
}

// WithFixedColumns sets the number of columns that are locked (do not scroll right).
func WithFixedColumns(cols uint) TableOption {
	return func(t *Table) {
		t.colFixed = cols
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

func (t *Table) initFooter() {
	t.footer.
		SetWordWrap(true).
		SetText(pad(t.footerText, 1)).
		SetTextColor(tcell.ColorDefault)
}

func (t *Table) initTable() {
	t.view.SetSelectable(true, false).
		SetSelectedStyle(customTUIStyle(t.style)).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEsc {
				t.screen.Stop()
			}
		}).
		SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			if ev.Key() == tcell.KeyCtrlR || ev.Key() == tcell.KeyF5 {
				if t.refreshFunc == nil {
					return ev
				}
				t.screen.Stop()
				t.refreshFunc()
			}
			if ev.Key() == tcell.KeyCtrlK {
				if t.copyKeyFunc == nil {
					return ev
				}
				r, c := t.view.GetSelection()
				t.copyKeyFunc(r, c, t.data)
			}
			if ev.Key() == tcell.KeyRune {
				switch ev.Rune() {
				case 'q':
					t.screen.Stop()
					os.Exit(0)
				case 'c':
					if t.copyFunc == nil {
						break
					}
					r, c := t.view.GetSelection()
					t.copyFunc(r, c, t.data)
				case 'v':
					if t.viewModeFunc == nil {
						break
					}
					r, c := t.view.GetSelection()

					go func() {
						func() {
							t.painter.ShowPage("secondary")
							defer t.painter.HidePage("secondary")

							dataFn, renderFn := t.viewModeFunc(r, c, t.data)

							out, err := renderFn(dataFn())
							if err == nil {
								t.screen.Suspend(func() { _ = PagerOut(out) })
							}
						}()

						// Refresh the screen.
						t.screen.Draw()
					}()
				default:
					if t.customKeyFunc != nil && t.customKeys != nil {
						for k, command := range t.customKeys {
							if string(ev.Rune()) == k {
								r, c := t.view.GetSelection()
								t.customKeyFunc(r, c, t.data, command.(string))
							}
						}
					}
				}
			}
			return ev
		})

	t.view.SetFixed(1, int(t.colFixed))
}

func renderTableHeader(t *Table, data []string) {
	style := tcell.StyleDefault.Bold(true)

	for c := 0; c < len(data); c++ {
		text := " " + data[c]

		cell := tview.NewTableCell(text).
			SetStyle(style).
			SetSelectable(false).
			SetTextColor(tcell.ColorSnow).
			SetBackgroundColor(tcell.ColorDarkCyan)

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
