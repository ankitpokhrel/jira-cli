package tui

import (
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const sidebarMaxWidth = 60

// PreviewData is the data to be shown in preview layout.
type PreviewData struct {
	Key      string
	Menu     string
	Contents func(string) interface{}
}

// Preview is the preview layout.
//
// It contains 2 tables internally, viz: sidebar and contents.
type Preview struct {
	screen              *Screen
	painter             *tview.Pages
	sidebar             *tview.Table
	contents            *Table
	footer              *tview.TextView
	data                []PreviewData
	initialText         string
	footerText          string
	sidebarSelectedFunc SelectedFunc
	contentsCache       map[string]interface{}
}

// PreviewOption is a functional option that wraps preview properties.
type PreviewOption func(*Preview)

// NewPreview constructs a new preview layout.
func NewPreview(opts ...PreviewOption) *Preview {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	pv := Preview{
		screen:        NewScreen(),
		sidebar:       tview.NewTable(),
		contents:      NewTable(),
		footer:        tview.NewTextView(),
		contentsCache: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(&pv)
	}
	pv.init()

	return &pv
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

// WithSidebarSelectedFunc sets a function that is called when any option in sidebar is selected.
func WithSidebarSelectedFunc(fn SelectedFunc) PreviewOption {
	return func(p *Preview) {
		p.sidebarSelectedFunc = fn
	}
}

// WithContentTableOpts sets contents table options.
func WithContentTableOpts(opts ...TableOption) PreviewOption {
	return func(p *Preview) {
		for _, opt := range opts {
			opt(p.contents)
		}
	}
}

// Paint paints the preview layout.
func (pv *Preview) Paint(pd []PreviewData) error {
	if len(pd) == 0 {
		return errNoData
	}

	pv.data = pd

	pv.sidebar.SetSelectionChangedFunc(func(r, c int) {
		pv.contents.view.Clear()
		pv.printText("Loading...")

		go pv.renderContents(pd[r])
	})

	if pv.sidebarSelectedFunc != nil {
		pv.sidebar.SetSelectedFunc(func(r, c int) {
			if r > 0 {
				pv.sidebarSelectedFunc(r, c, pd[r])
			}
		})
	}

	for i, d := range pd {
		style := tcell.StyleDefault
		if i == 0 {
			style = style.Bold(true)
		}

		cell := tview.NewTableCell(pad(d.Menu, 1)).
			SetMaxWidth(sidebarMaxWidth).
			SetStyle(style)

		pv.sidebar.SetCell(i, 0, cell)
	}

	pv.printText(pv.initialText)

	return pv.screen.Paint(pv.painter)
}

func (pv *Preview) renderContents(pd PreviewData) {
	if pd.Contents == nil {
		pv.printText("No contents defined.")
		return
	}

	if _, ok := pv.contentsCache[pd.Key]; !ok {
		pv.contentsCache[pd.Key] = pd.Contents(pd.Key)
	}

	switch v := pv.contentsCache[pd.Key].(type) {
	case string:
		pv.printText(v)
	case TableData:
		data := pv.contentsCache[pd.Key].(TableData)

		pv.screen.QueueUpdateDraw(func() {
			pv.contents.view.Clear()

			if len(data) == 1 {
				pv.printText("No results to show.")
				return
			}

			pv.contents.render(data)
		})
	}
}

func (pv *Preview) init() {
	pv.initSidebar()
	pv.initContents()
	pv.initFooter()

	grid := tview.NewGrid().
		SetRows(0, 1, 2).
		SetColumns(sidebarMaxWidth, 1, 0).
		AddItem(pv.sidebar, 0, 0, 2, 1, 0, 0, true).
		AddItem(pv.contents.view, 0, 2, 2, 1, 0, 0, false).
		AddItem(pv.footer, 2, 0, 1, 3, 0, 0, false)

	pv.painter = tview.NewPages().
		AddPage("primary", grid, true, true).
		AddPage("secondary", getInfoModal(), true, false)

	pv.initLayout(pv.sidebar)
	pv.initLayout(pv.contents.view)
}

func (pv *Preview) initSidebar() {
	pv.sidebar.
		SetSelectable(true, false).
		SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			if ev.Key() == tcell.KeyTab {
				pv.screen.SetFocus(pv.contents.view)
				pv.contents.view.SetSelectable(true, false).Select(1, 0)
			}
			if ev.Key() == tcell.KeyRune {
				switch ev.Rune() {
				case 'q':
					pv.screen.Stop()
					os.Exit(0)
				case 'w':
					pv.screen.SetFocus(pv.contents.view)
					pv.contents.view.SetSelectable(true, false).Select(1, 0)
				default:
					pv.contents.view.SetSelectable(false, false)
				}
			}
			return ev
		})
}

func (pv *Preview) initContents() {
	pv.contents.view.
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkGray).
		SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			contents := func() interface{} {
				sr, _ := pv.sidebar.GetSelection()
				return pv.contentsCache[pv.data[sr].Key]
			}
			if ev.Key() == tcell.KeyCtrlK {
				if pv.contents.copyKeyFunc == nil {
					return ev
				}
				r, c := pv.contents.view.GetSelection()
				pv.contents.copyKeyFunc(r, c, contents())
			}
			if ev.Key() == tcell.KeyTab {
				pv.screen.SetFocus(pv.sidebar)
				pv.contents.view.SetSelectable(false, false)
			}
			if ev.Key() == tcell.KeyRune {
				switch ev.Rune() {
				case 'q':
					pv.screen.Stop()
					os.Exit(0)
				case 'w':
					pv.screen.SetFocus(pv.sidebar)
					pv.contents.view.SetSelectable(false, false)
				case 'c':
					if pv.contents.copyFunc == nil {
						break
					}
					r, c := pv.contents.view.GetSelection()
					pv.contents.copyFunc(r, c, contents())
				case 'v':
					if pv.contents.viewModeFunc == nil {
						break
					}
					sr, _ := pv.sidebar.GetSelection()
					r, c := pv.contents.view.GetSelection()

					go func() {
						func() {
							pv.painter.ShowPage("secondary")
							defer func() {
								pv.painter.HidePage("secondary")
								pv.screen.SetFocus(pv.contents.view)
							}()

							contents := pv.contentsCache[pv.data[sr].Key]
							dataFn, renderFn := pv.contents.viewModeFunc(r, c, contents)

							out, err := renderFn(dataFn())
							if err == nil {
								pv.screen.Suspend(func() { _ = PagerOut(out) })
							}
						}()

						// Refresh the screen.
						pv.screen.Draw()
					}()
				}
			}
			return ev
		})
}

func (pv *Preview) initFooter() {
	pv.footer.
		SetWordWrap(true).
		SetText(pad(pv.footerText, 1)).
		SetTextColor(tcell.ColorDefault)
}

func (pv *Preview) initLayout(view *tview.Table) {
	view.SetSelectedStyle(customTUIStyle(pv.contents.style)).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEsc {
				pv.screen.Stop()
			}
		})

	view.SetFixed(1, int(pv.contents.colFixed))
}

func (pv *Preview) printText(s string) {
	lines := splitText(s)
	for i, line := range lines {
		pv.contents.view.SetCell(i, 0, tview.NewTableCell(pad(line, 1)).
			SetStyle(tcell.StyleDefault).
			SetSelectable(false))
	}
}
