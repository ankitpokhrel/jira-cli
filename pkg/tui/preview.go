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
	painter             *tview.Grid
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
		contents:      NewTable(),
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
	pv.initSidebarView()
	pv.initContentsView()
	pv.initFooterView()

	pv.painter = tview.NewGrid().
		SetRows(0, 1, 2).
		SetColumns(sidebarMaxWidth, 1, 0).
		AddItem(pv.sidebar, 0, 0, 2, 1, 0, 0, true).
		AddItem(tview.NewTextView(), 0, 1, 1, 1, 0, 0, false). // Dummy view to fake col padding.
		AddItem(pv.contents.view, 0, 2, 2, 1, 0, 0, false).
		AddItem(tview.NewTextView(), 1, 0, 1, 1, 0, 0, false). // Dummy view to fake row padding.
		AddItem(pv.footer, 2, 0, 1, 3, 0, 0, false)

	pv.initLayout(pv.sidebar)
	pv.initLayout(pv.contents.view)
}

func (pv *Preview) initSidebarView() {
	pv.sidebar = tview.NewTable()

	pv.sidebar.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyRune {
			switch ev.Rune() {
			case 'q':
				pv.screen.Stop()
				os.Exit(0)
			case 'w':
				pv.screen.SetFocus(pv.contents.view)
			}
		}
		return ev
	})
}

func (pv *Preview) initContentsView() {
	pv.contents.view.
		SetBorder(true).
		SetBorderColor(tcell.ColorDarkGray)

	pv.contents.view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyRune {
			switch ev.Rune() {
			case 'q':
				pv.screen.Stop()
				os.Exit(0)
			case 'w':
				pv.screen.SetFocus(pv.sidebar)
			case 'v':
				if pv.contents.viewModeFunc != nil {
					sr, _ := pv.sidebar.GetSelection()
					r, c := pv.contents.view.GetSelection()
					contents := pv.contentsCache[pv.data[sr].Key]

					pv.screen.Suspend(func() { _ = pv.contents.viewModeFunc(r, c, contents) })
				}
			}
		}
		return ev
	})
}

func (pv *Preview) initFooterView() {
	view := tview.NewTextView().
		SetWordWrap(true).
		SetText(pad(pv.footerText, 1)).
		SetTextColor(tcell.ColorDefault)

	pv.footer = view
}

func (pv *Preview) initLayout(view *tview.Table) {
	view.SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Bold(true).Dim(true))

	view.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			pv.screen.Stop()
		}
	})

	view.SetFixed(1, 1)
}

func (pv *Preview) printText(s string) {
	lines := splitText(s)
	for i, line := range lines {
		pv.contents.view.SetCell(i, 0, tview.NewTableCell(pad(line, 1)).
			SetStyle(tcell.StyleDefault).
			SetSelectable(false))
	}
}
