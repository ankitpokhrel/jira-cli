package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// DetailView binds a detail page and its actions to the selected item.
type DetailView struct {
	Title         string
	Load          func() (string, error)
	SubmitComment func(string) error
}

// DetailViewFunc is called on the UI loop when the user opens a table item.
// Returned callbacks must capture the target; they run asynchronously.
type DetailViewFunc func(row, col int, data any) DetailView

type tableDetail struct {
	table    *Table
	view     DetailView
	text     *tview.TextView
	status   *tview.TextView
	composer *commentComposer
	loadID   uint64
	posted   bool
}

const (
	detailKeys          = "c: comment · CtrlR/F5: reload · q/Esc: back"
	composerKeys        = "Enter: newline · CtrlS/ShiftEnter: post · Tab/ShiftTab: navigate · Esc/CtrlC: cancel"
	composerButtonWidth = 12
)

func (t *Table) captureComposerInterrupts() func() {
	previous := t.screen.GetInputCapture()
	t.screen.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// tview handles CtrlC before primitive captures can protect a draft.
		if ev.Key() == tcell.KeyCtrlC && t.detail != nil && t.detail.composer != nil {
			if !t.painter.HasPage("discard") {
				t.detail.cancelComposer()
			}
			return nil
		}
		if previous != nil {
			return previous(ev)
		}
		return ev
	})
	return func() { t.screen.SetInputCapture(previous) }
}

func (t *Table) openDetail() {
	r, c := t.view.GetSelection()
	if t.detailViewFunc == nil || len(t.data) == 0 || len(t.data[0]) == 0 ||
		r <= 0 || r >= len(t.data) || c < 0 || c >= len(t.data[0]) || len(t.data[r]) < len(t.data[0]) {
		return
	}
	// Resolve the target before starting workers, not from a later table selection.
	d := &tableDetail{table: t, view: t.detailViewFunc(r, c, t.data)}
	t.detail = d
	d.text = tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetWrap(true)
	d.text.SetBorder(true).SetTitle(tview.Escape(d.view.Title))
	d.status = tview.NewTextView().SetWrap(true)
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(d.text, 0, 1, true).
		AddItem(d.status, 3, 0, false)
	page.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEsc || ev.Key() == tcell.KeyRune && ev.Rune() == 'q':
			d.close()
			return nil
		case ev.Key() == tcell.KeyCtrlR || ev.Key() == tcell.KeyF5:
			d.load()
			return nil
		case ev.Key() == tcell.KeyRune && ev.Rune() == 'c':
			d.openComposer()
			return nil
		}
		return ev
	})
	t.painter.AddPage("detail", page, true, true)
	t.screen.SetFocus(d.text)
	d.load()
}

func (d *tableDetail) close() {
	d.table.detail = nil
	d.table.painter.RemovePage("detail")
	d.table.screen.SetFocus(d.table.view)
}

func (d *tableDetail) load() {
	d.loadID++
	loadID := d.loadID
	prefix := ""
	if d.posted {
		prefix = "Comment posted. "
	}
	if d.view.Load == nil {
		d.status.SetText(prefix + "Details unavailable: no loader.\n" + detailKeys)
		return
	}
	d.status.SetText(prefix + "Loading...\n" + detailKeys)
	go func() {
		body, err := d.view.Load()
		d.table.screen.QueueUpdateDraw(func() {
			if d.table.detail != d || d.loadID != loadID {
				return
			}
			if err != nil {
				message := "Load failed: "
				if d.posted {
					message = "Comment posted. Reload failed (do not repost): "
				}
				d.status.SetText("CtrlR/F5: retry loading.\n" + message + err.Error())
				return
			}
			// Escape user text first so ANSI-generated tags remain the only styling.
			d.text.SetText(tview.TranslateANSI(tview.Escape(body))).ScrollToBeginning()
			d.status.SetText(prefix + detailKeys)
		})
	}()
}

func (d *tableDetail) openComposer() {
	if d.view.SubmitComment == nil {
		d.status.SetText("Commenting unavailable.\n" + detailKeys)
		return
	}
	c := &commentComposer{
		Flex:   tview.NewFlex().SetDirection(tview.FlexRow),
		draft:  tview.NewTextArea().SetLabel("Comment for " + tview.Escape(d.view.Title) + ": "),
		status: tview.NewTextView().SetText(composerKeys),
	}
	d.composer = c
	c.draft.SetBorder(true).SetTitle("New comment")
	c.post = tview.NewButton("Post").SetSelectedFunc(d.submit)
	c.cancel = tview.NewButton("Cancel").SetSelectedFunc(d.cancelComposer)
	buttons := tview.NewFlex().
		AddItem(c.post, composerButtonWidth, 0, false).
		AddItem(c.cancel, composerButtonWidth, 0, false)
	c.AddItem(c.draft, 0, 1, true).AddItem(c.status, 3, 0, false).AddItem(buttons, 1, 0, false)
	c.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if c.posting {
			return nil
		}
		switch {
		case ev.Key() == tcell.KeyCtrlS || ev.Key() == tcell.KeyEnter && ev.Modifiers()&tcell.ModShift != 0:
			d.submit()
			return nil
		case ev.Key() == tcell.KeyEsc:
			d.cancelComposer()
			return nil
		case ev.Key() == tcell.KeyTab || ev.Key() == tcell.KeyBacktab:
			items := []tview.Primitive{c.draft, c.post, c.cancel}
			for i, item := range items {
				if item.HasFocus() {
					step := 1
					if ev.Key() == tcell.KeyBacktab {
						step = len(items) - 1
					}
					d.table.screen.SetFocus(items[(i+step)%len(items)])
					break
				}
			}
			return nil
		}
		return ev
	})
	d.table.painter.AddPage("composer", c, true, true)
	d.table.screen.SetFocus(c.draft)
}

func (d *tableDetail) cancelComposer() {
	c := d.composer
	if c == nil || c.posting {
		return
	}
	if c.draft.GetText() == "" {
		d.closeComposer()
		return
	}
	confirm := tview.NewModal().SetText("Discard this draft?").AddButtons([]string{"Keep editing", "Discard"})
	confirm.SetDoneFunc(func(index int, _ string) {
		d.table.painter.RemovePage("discard")
		if index == 1 {
			d.closeComposer()
			return
		}
		d.table.screen.SetFocus(c.draft)
	})
	d.table.painter.AddPage("discard", confirm, true, true)
	confirm.SetFocus(0)
	d.table.screen.SetFocus(confirm)
}

func (d *tableDetail) closeComposer() {
	d.table.painter.RemovePage("composer")
	d.composer = nil
	d.table.screen.SetFocus(d.text)
}

func (d *tableDetail) submit() {
	c := d.composer
	if c == nil || c.posting || d.view.SubmitComment == nil {
		return
	}
	body := c.draft.GetText()
	if strings.TrimSpace(body) == "" {
		c.status.SetText("Comment must not be empty.\n" + composerKeys)
		d.table.screen.SetFocus(c.draft)
		return
	}
	c.posting = true
	c.draft.SetDisabled(true)
	c.post.SetDisabled(true)
	c.cancel.SetDisabled(true)
	c.status.SetText("Posting... Please wait; draft is locked until the request completes.")
	go func() {
		err := d.view.SubmitComment(body)
		d.table.screen.QueueUpdateDraw(func() {
			if d.table.detail != d || d.composer != c {
				return
			}
			c.posting = false
			if err != nil {
				c.draft.SetDisabled(false)
				c.post.SetDisabled(false)
				c.cancel.SetDisabled(false)
				c.status.SetText(fmt.Sprintf("Post failed: draft kept. CtrlS/Post: retry; Esc/CtrlC: cancel.\n%s", err))
				d.table.screen.SetFocus(c.draft)
				return
			}
			c.draft.SetText("", false)
			d.closeComposer()
			d.posted = true
			d.load()
		})
	}()
}

type commentComposer struct {
	*tview.Flex
	draft   *tview.TextArea
	status  *tview.TextView
	post    *tview.Button
	cancel  *tview.Button
	posting bool
}

func (c *commentComposer) PasteHandler() func(string, func(tview.Primitive)) {
	return func(text string, setFocus func(tview.Primitive)) {
		// The pinned tview TextArea accepts paste even when disabled.
		if !c.posting {
			c.Flex.PasteHandler()(text, setFocus)
		}
	}
}
