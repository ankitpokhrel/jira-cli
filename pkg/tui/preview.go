package tui

import (
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
type Preview struct {
	screen      *Screen
	painter     *tview.Grid
	sidebar     *tview.Table
	contents    *Table
	footer      *tview.TextView
	initialText string
	footerText  string
}

// PreviewOption is a functional option that wraps preview properties.
type PreviewOption func(*Preview)

// NewPreview returns new preview layout.
func NewPreview(opts ...PreviewOption) *Preview {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack

	pv := Preview{
		screen:   NewScreen(),
		contents: NewTable(),
	}

	for _, opt := range opts {
		opt(&pv)
	}

	pv.init()

	return &pv
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

	pv.painter.SetBackgroundColor(tcell.ColorBlack)

	pv.initLayout(pv.sidebar, pv.contents.view)
	pv.initLayout(pv.contents.view, pv.sidebar)
}

func (pv *Preview) initSidebarView() {
	pv.sidebar = tview.NewTable()
}

func (pv *Preview) initContentsView() {
	contents := tview.NewTable()

	contents.SetBorder(true).
		SetBorderColor(tcell.ColorDarkGray).
		SetBackgroundColor(tcell.ColorBlack)

	pv.contents.view = contents
}

func (pv *Preview) initFooterView() {
	view := tview.NewTextView().SetWordWrap(true)

	view.SetText(pad(pv.footerText, 1)).SetTextColor(tcell.ColorAntiqueWhite)

	pv.footer = view
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

// WithContentTableOpts sets contents table options.
func WithContentTableOpts(opts ...TableOption) PreviewOption {
	return func(p *Preview) {
		for _, opt := range opts {
			opt(p.contents)
		}
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
			SetMaxWidth(sidebarMaxWidth).
			SetStyle(style)

		pv.sidebar.SetCell(i, 0, cell)

		pv.sidebar.SetSelectionChangedFunc(func(r, c int) {
			pv.contents.view.Clear()
			pv.printText("Loading...")

			go pv.renderContents(pd[r])
		})
	}

	pv.printText(pv.initialText)

	return pv.screen.Paint(pv.painter)
}

func (pv *Preview) renderContents(pd PreviewData) {
	if pd.Contents == nil {
		pv.printText("No contents defined.")

		return
	}

	switch v := pd.Contents(pd.Key).(type) {
	case string:
		pv.printText(v)

	case TableData:
		pv.screen.QueueUpdateDraw(func() {
			pv.contents.view.Clear()

			data := pd.Contents(pd.Key).(TableData)

			if len(data) == 1 {
				pv.printText("No results to show.")

				return
			}

			pv.contents.view.SetSelectedFunc(func(r, c int) {
				pv.contents.selectedFunc(r, c, &data)
			})

			renderTableHeader(pv.contents, data[0])
			renderTableCell(pv.contents, data)
		})
	}
}

func (pv *Preview) printText(s string) {
	lines := splitText(s)

	for i, line := range lines {
		pv.contents.view.SetCell(i, 0, tview.NewTableCell(pad(line, 1)).
			SetStyle(tcell.StyleDefault).
			SetSelectable(false))
	}
}
