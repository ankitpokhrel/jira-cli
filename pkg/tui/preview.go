package tui

import (
	"bufio"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PreviewData is the data to be shown in preview layout.
type PreviewData struct {
	Key      string
	Menu     string
	Contents func(string) interface{}
}

// Preview is the preview layout.
type Preview struct {
	screen      *Screen
	painter     *tview.Grid
	sidebar     *tview.Table
	contents    *tview.Table
	initialText string
	footerText  string
}

// PreviewOption is a functional option to wrap preview properties.
type PreviewOption func(preview *Preview)

// NewPreview returns new preview layout.
func NewPreview(s *Screen, opts ...PreviewOption) *Preview {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack

	pv := Preview{
		screen: s,
	}

	for _, opt := range opts {
		opt(&pv)
	}

	sidebar := tview.NewTable()
	contents := tview.NewTable()
	footerView := tview.NewTextView().SetWordWrap(true)

	initFooterView(footerView, pv.footerText)

	contents.SetBorder(true).
		SetBorderColor(tcell.ColorDarkGray).
		SetBackgroundColor(tcell.ColorBlack)

	pv.painter = tview.NewGrid().
		SetRows(0, 1, 2).
		SetColumns(60, 1, 0).
		AddItem(sidebar, 0, 0, 2, 1, 0, 0, true).
		AddItem(tview.NewTextView(), 0, 1, 1, 1, 0, 0, false). // Dummy view to fake col padding.
		AddItem(contents, 0, 2, 2, 1, 0, 0, false).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 0, 0, false). // Dummy view to fake row padding.
		AddItem(footerView, 2, 0, 1, 3, 0, 0, false)

	pv.painter.SetBackgroundColor(tcell.ColorBlack)

	pv.sidebar = sidebar
	pv.contents = contents

	pv.initLayout(sidebar, contents)
	pv.initLayout(contents, sidebar)

	return &pv
}

func (pv *Preview) initLayout(view *tview.Table, nextView *tview.Table) {
	view.SetSelectable(true, false)

	view.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			pv.screen.Stop()
		}
	})

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				pv.screen.Stop()

			case 'w':
				if view.HasFocus() {
					pv.screen.SetFocus(nextView)
				} else {
					pv.screen.SetFocus(view)
				}
			}
		}

		return event
	})

	view.SetFixed(1, 1)
}

// WithInitialText sets initial text that is displayed in the contents screen.
func WithInitialText(text string) PreviewOption {
	return func(p *Preview) {
		p.initialText = text
	}
}

// WithPreviewFooterText sets footer text that is displayed after the preview layout.
func WithPreviewFooterText(text string) PreviewOption {
	return func(p *Preview) {
		p.footerText = text
	}
}

// Render renders the table layout. First row is treated as a table header.
func (pv *Preview) Render(pd []PreviewData) error {
	if len(pd) == 0 {
		return errNoData
	}

	for i, d := range pd {
		style := tcell.StyleDefault
		if i == 0 {
			style = style.Bold(true)
		}

		cell := tview.NewTableCell(pad(d.Menu, 1)).
			SetMaxWidth(60).
			SetStyle(style)

		pv.sidebar.SetCell(i, 0, cell)

		pv.sidebar.SetSelectionChangedFunc(func(r, c int) {
			var data TableData

			pv.contents.Clear()

			pv.contents.SetCell(0, 0, tview.NewTableCell(pad("Loading...", 1)).
				SetStyle(tcell.StyleDefault).
				SetSelectable(false))

			go func() {
				if pd[r].Contents == nil {
					return
				}

				switch v := pd[r].Contents(pd[r].Key).(type) {
				case string:
					pv.printText(v)

				case TableData:
					pv.screen.QueueUpdateDraw(func() {
						pv.contents.Clear()

						data = pd[r].Contents(pd[r].Key).(TableData)

						rows, cols := len(data), len(data[0])

						if rows == 1 {
							pv.contents.SetCell(0, 0, tview.NewTableCell(pad("No results to show.", 1)).
								SetStyle(tcell.StyleDefault).
								SetSelectable(false))

							return
						}

						for r := 0; r < rows; r++ {
							for c := 0; c < cols; c++ {
								style := tcell.StyleDefault.Background(tcell.ColorBlack)
								if r == 0 {
									style = style.Bold(true).Background(tcell.ColorDarkCyan)
								}

								cell := tview.NewTableCell(pad(data[r][c], 1)).
									SetMaxWidth(70).
									SetStyle(style)

								if r == 0 {
									cell.SetSelectable(false)
								}

								pv.contents.SetCell(r, c, cell)
							}
						}
					})
				}
			}()
		})
	}

	pv.printText(pv.initialText)

	return pv.screen.SetRoot(pv.painter, true).SetFocus(pv.painter).Run()
}

func (pv *Preview) printText(s string) {
	var lines []string

	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	for i, line := range lines {
		pv.contents.SetCell(i, 0, tview.NewTableCell(pad(line, 1)).
			SetStyle(tcell.StyleDefault).
			SetSelectable(false))
	}
}
