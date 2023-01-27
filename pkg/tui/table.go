package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/ankitpokhrel/jira-cli/pkg/tui/primitive"
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

// RefreshTableStateFunc is used to refresh the table state.
type RefreshTableStateFunc func(row, col int, val string)

// MoveHandlerFunc is a handler for move action.
type MoveHandlerFunc func(state string) error

// MoveFunc is fired when a user press 'm' character in the table cell.
type MoveFunc func(row, col int) func() (key string, actions []string, handler MoveHandlerFunc, status string, refresh RefreshTableStateFunc)

// CopyFunc is fired when a user press 'c' character in the table cell.
type CopyFunc func(row, column int, data interface{})

// CopyKeyFunc is fired when a user press 'CTRL+K' character in the table cell.
type CopyKeyFunc func(row, column int, data interface{})

// TableData is the data to be displayed in a table.
type TableData [][]string

// Get returns the value of the cell at the given row and column.
func (td TableData) Get(r, c int) string {
	if r != -1 && c != -1 {
		return td[r][c]
	}
	return ""
}

// GetIndex returns the index of the specified column.
func (td TableData) GetIndex(key string) int {
	if len(td) == 0 {
		return -1
	}
	for i, v := range td[0] {
		if strings.EqualFold(v, key) {
			return i
		}
	}
	return -1
}

// Update updates the data at given row and column.
func (td TableData) Update(r, c int, val string) {
	if r != -1 && c != -1 {
		td[r][c] = val
	}
}

// TableStyle sets the style of the table.
type TableStyle struct {
	SelectionBackground string
	SelectionForeground string
	SelectionTextIsBold bool
}

// Table is a table layout.
type Table struct {
	screen       *Screen
	painter      *tview.Pages
	view         *tview.Table
	footer       *tview.TextView
	secondary    *tview.Modal
	help         *primitive.InfoModal
	action       *primitive.ActionModal
	style        TableStyle
	data         TableData
	colPad       uint
	colFixed     uint
	maxColWidth  uint
	footerText   string
	helpText     string
	selectedFunc SelectedFunc
	viewModeFunc ViewModeFunc
	moveFunc     MoveFunc
	refreshFunc  RefreshFunc
	copyFunc     CopyFunc
	copyKeyFunc  CopyKeyFunc
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
		help:        primitive.NewInfoModal(),
		secondary:   getInfoModal(),
		action:      getActionModal(),
		colPad:      defaultColPad,
		maxColWidth: defaultColWidth,
	}
	for _, opt := range opts {
		opt(&tbl)
	}

	tbl.initTable()
	tbl.initFooter()
	tbl.initHelp()

	grid := tview.NewGrid().
		SetRows(0, 1, 2).
		AddItem(tbl.view, 0, 0, 1, 1, 0, 0, true).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 0, 0, false). // Dummy view to fake row padding.
		AddItem(tbl.footer, 2, 0, 1, 1, 0, 0, false)

	tbl.action.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q') {
			tbl.painter.HidePage("action")
		}
		return ev
	})

	tbl.painter = tview.NewPages().
		AddPage("primary", grid, true, true).
		AddPage("secondary", tbl.secondary, true, false).
		AddPage("help", tbl.help, true, false).
		AddPage("action", tbl.action, true, false)

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

// WithTableHelpText sets the help text for the view.
func WithTableHelpText(text string) TableOption {
	return func(t *Table) {
		t.helpText = text
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

// WithMoveFunc sets a func that is triggered when an action button is pressed.
func WithMoveFunc(fn MoveFunc) TableOption {
	return func(t *Table) {
		t.moveFunc = fn
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

func (t *Table) initHelp() {
	t.help.
		SetInfo(t.helpText).
		SetAlign(tview.AlignLeft).
		SetTitle("USAGE")

	t.help.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q') {
			t.painter.HidePage("help")
		}
		return ev
	})
}

//nolint:gocyclo
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
				case '?':
					t.painter.ShowPage("help")
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
				case 'm':
					if t.moveFunc == nil {
						break
					}

					refreshContextInFooter := func() {
						t.action.GetFooter().SetText("Use TAB or ← → to navigate, ENTER to select, ESC or q to cancel.").SetTextColor(tcell.ColorGray)
					}

					go func() {
						func() {
							t.painter.ShowPage("secondary").SendToFront("secondary")
							defer func() {
								t.painter.HidePage("secondary")
								t.painter.ShowPage("action")
							}()
							refreshContextInFooter()

							r, c := t.view.GetSelection()
							key, actions, handler, currentStatus, refreshFunc := t.moveFunc(r, c)()

							currentStatusIdx := func() int {
								for i, btn := range actions {
									if btn == currentStatus {
										return i
									}
								}
								return 0
							}

							t.action.ClearButtons().AddButtons(actions).SetFocus(currentStatusIdx())
							t.action.SetText(
								fmt.Sprintf("Select desired state to transition %s to:", key),
							)

							t.action.SetDoneFunc(func(btnIndex int, btnLabel string) {
								t.action.GetFooter().SetText("Processing. Please wait...").SetTextColor(tcell.ColorGray)
								t.screen.ForceDraw()

								err := handler(btnLabel)
								if err != nil {
									t.action.GetFooter().SetText(
										fmt.Sprintf("Error: %s", err.Error()),
									).SetTextColor(tcell.ColorRed)
									return
								}
								t.painter.HidePage("action")
								refreshContextInFooter()

								if refreshFunc != nil {
									refreshFunc(r, c, btnLabel)
									_ = t.Paint(t.data)
								}
							})
						}()

						// Refresh the screen.
						t.screen.Draw()
					}()
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

func renderTableCell(t *Table, data TableData) {
	rows, cols := len(data), len(data[0])

	for r := 1; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell := tview.NewTableCell(pad(data.Get(r, c), t.colPad)).
				SetMaxWidth(int(t.maxColWidth)).
				SetTextColor(tcell.ColorDefault)

			t.view.SetCell(r, c, cell)
		}
	}
}
