package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ReplyTarget identifies a root thread offered by the detail loader.
type ReplyTarget struct {
	// ID is the root Jira comment ID passed to the reply handler.
	ID string
	// Author is shown with the root comment in the reply picker.
	Author string
	// Created is the root comment's timestamp shown in the picker.
	Created string
	// Preview is a plain-text excerpt shown beneath the thread label.
	Preview string
}

// DetailContent is one atomic snapshot of details and available reply targets.
type DetailContent struct {
	// Text may contain ANSI styling; tview markup is treated as literal text.
	Text string
	// ReplyTargets lists root threads offered to the native reply picker.
	ReplyTargets []ReplyTarget
	// ReplyUnavailable overrides ReplyTargets with a reason the picker cannot be used.
	ReplyUnavailable string
}

// DetailView binds a detail page and its actions to the selected item.
type DetailView struct {
	// Title is displayed in the detail page's border.
	Title string
	// Load may overlap earlier reloads; it must not mutate UI primitives.
	Load func() (DetailContent, error)
	// SubmitComment posts a top-level comment; nil disables composition.
	SubmitComment func(string) error
	// SubmitReply must return nil only after confirmation, since success clears
	// the draft. A nil callback disables posting rather than creating a flat comment.
	SubmitReply func(rootID, body string) error
	// OpenTicket resolves input without changing the active page; the candidate's
	// Load must succeed before the UI switches to its content and write callbacks.
	OpenTicket func(input string) (DetailView, error)
}

// DetailViewFunc is called on the UI loop when the user opens a table item.
// Returned callbacks must capture the target; they run asynchronously.
type DetailViewFunc func(row, col int, data any) DetailView

// tableDetail holds UI-loop-owned state; loadID invalidates stale loads and
// reply selections when a reload or ticket switch supersedes their snapshot.
type tableDetail struct {
	table    *Table
	view     DetailView
	text     *tview.TextView
	status   *tview.TextView
	composer *commentComposer
	loadID   uint64
	posted   string
	prompt   *ticketPrompt

	picker           *tview.List
	replyTargets     []ReplyTarget
	replyUnavailable string
}

const (
	detailKeys              = "c: comment · r: reply · /: view ticket · CtrlR/F5: reload · q/Esc: back"
	composerKeys            = "Enter: newline · ShiftTab: focus buttons\nCtrlS/ShiftEnter: post · Esc/CtrlC: cancel"
	composerButtonWidth     = 12
	replyPreviewLimit       = 80
	replyPostingUnavailable = "Reply posting unavailable; no request sent"
	replySelectionExpired   = "Thread selection expired; cancel and choose a thread again. No request sent."
)

// captureComposerInterrupts intercepts CtrlC before tview's application-level
// exit handling, allowing prompts to cancel and nonempty drafts to be protected.
// The returned callback restores the previous capture when the table exits.
func (t *Table) captureComposerInterrupts() func() {
	previous := t.screen.GetInputCapture()
	t.screen.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// tview handles CtrlC before primitive captures can protect a draft.
		if ev.Key() == tcell.KeyCtrlC && t.detail != nil {
			if t.detail.prompt != nil {
				t.detail.closeTicketPrompt()
				return nil
			}
			if t.detail.picker != nil {
				t.detail.closeReplyPicker()
				return nil
			}
			if t.detail.composer != nil {
				if !t.painter.HasPage("discard") {
					t.detail.cancelComposer()
				}
				return nil
			}
		}
		if previous != nil {
			return previous(ev)
		}
		return ev
	})
	return func() { t.screen.SetInputCapture(previous) }
}

// openDetail captures the selected ticket before workers start, so later table
// selection changes cannot redirect detail loads or comment submissions.
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
	page.SetInputCapture(d.captureDetailInput)
	t.painter.AddPage("detail", page, true, true)
	t.screen.SetFocus(d.text)
	d.load()
}

// captureDetailInput consumes detail actions while passing scrolling keys through;
// the slash shortcut is active only with detail text focus, not in an editor.
func (d *tableDetail) captureDetailInput(ev *tcell.EventKey) *tcell.EventKey {
	switch {
	case ev.Key() == tcell.KeyEsc || ev.Key() == tcell.KeyRune && ev.Rune() == 'q':
		d.close()
	case ev.Key() == tcell.KeyCtrlR || ev.Key() == tcell.KeyF5:
		d.load()
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'c':
		d.openComposer()
	case ev.Key() == tcell.KeyRune && ev.Rune() == 'r':
		d.openReplyPicker()
	case ev.Key() == tcell.KeyRune && ev.Rune() == '/' && d.text.HasFocus():
		d.openTicketPrompt()
	default:
		return ev
	}
	return nil
}

// close abandons pending navigation and returns focus to the existing list
// instead of rebuilding it or changing its selection.
func (d *tableDetail) close() {
	d.closeTicketPrompt()
	d.table.detail = nil
	d.table.painter.RemovePage("detail")
	d.table.screen.SetFocus(d.table.view)
}

// load expires reply selections immediately and applies only the latest worker's
// result on the UI loop. A reload failure preserves any confirmed-post notice.
func (d *tableDetail) load() {
	d.closeTicketPrompt()
	d.loadID++
	loadID := d.loadID
	d.replyTargets = nil
	d.replyUnavailable = "Reply threads are loading; wait for details."
	if d.picker != nil {
		d.closeReplyPicker()
	}
	if c := d.composer; c != nil && c.replyID != "" && !c.posting {
		c.post.SetDisabled(true)
		c.status.SetText(replySelectionExpired)
	}
	prefix := ""
	if d.posted != "" {
		prefix = d.posted + " posted. "
	}
	if d.view.Load == nil {
		d.replyUnavailable = "Reply threads unavailable: no detail loader."
		d.status.SetText(prefix + "Details unavailable: no loader.\n" + detailKeys)
		return
	}
	d.status.SetText(prefix + "Loading...\n" + detailKeys)
	load := d.view.Load
	go func() {
		content, err := load()
		d.table.screen.QueueUpdateDraw(func() {
			if d.table.detail != d || d.loadID != loadID {
				return
			}
			if err != nil {
				d.replyUnavailable = "Reply threads unavailable: detail load failed. CtrlR/F5: retry."
				message := "Load failed: "
				if d.posted != "" {
					message = d.posted + " posted. Reload failed (do not repost): "
				}
				d.status.SetText("CtrlR/F5: retry loading.\n" + message + err.Error())
				return
			}
			// Escape user text first so ANSI-generated tags remain the only styling.
			d.text.SetText(tview.TranslateANSI(tview.Escape(content.Text))).ScrollToBeginning()
			d.replyTargets, d.replyUnavailable = replySnapshot(content)
			d.status.SetText(prefix + detailKeys)
		})
	}()
}

// replySnapshot copies and validates the loaded targets so the picker cannot
// share a mutable slice or offer ambiguous, blank, or control-bearing IDs.
func replySnapshot(content DetailContent) ([]ReplyTarget, string) {
	if content.ReplyUnavailable != "" {
		return nil, content.ReplyUnavailable
	}
	if len(content.ReplyTargets) == 0 {
		return nil, "No reply threads loaded."
	}
	targets := append([]ReplyTarget(nil), content.ReplyTargets...)
	seen := make(map[string]bool, len(targets))
	for _, target := range targets {
		if target.ID == "" || strings.ContainsFunc(target.ID, func(r rune) bool {
			return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
		}) {
			return nil, "Reply threads unavailable: invalid target ID."
		}
		if seen[target.ID] {
			return nil, "Reply threads unavailable: duplicate target IDs."
		}
		seen[target.ID] = true
	}
	return targets, ""
}

// plainReplyText strips terminal styling and invisible controls before untrusted
// comment metadata is used in single-line picker labels or composer titles.
func plainReplyText(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, ansi.Strip(text))
	return strings.Join(strings.Fields(text), " ")
}

// openReplyPicker binds choices to the current load generation so a stale picker
// cannot open a reply composer after a reload or ticket switch.
func (d *tableDetail) openReplyPicker() {
	if d.picker != nil || d.composer != nil || d.prompt != nil {
		return
	}
	if d.replyUnavailable != "" || len(d.replyTargets) == 0 {
		message := d.replyUnavailable
		if d.posted != "" {
			message = d.posted + " posted. " + message
		}
		d.status.SetText(message + "\n" + detailKeys)
		return
	}
	targets, loadID := d.replyTargets, d.loadID
	picker := tview.NewList().ShowSecondaryText(true).SetHighlightFullLine(true)
	picker.SetBorder(true).SetTitle("Which thread do you want to reply to?")
	d.picker = picker
	for _, target := range targets {
		main := plainReplyText(target.ID + " · " + target.Author + " · " + target.Created)
		preview := []rune(plainReplyText(target.Preview))
		if len(preview) > replyPreviewLimit {
			preview = append(preview[:replyPreviewLimit], '…')
		}
		picker.AddItem(tview.Escape(main), tview.Escape(string(preview)), 0, nil)
	}
	picker.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if d.table.detail != d || d.picker != picker || d.loadID != loadID || index < 0 || index >= len(targets) {
			return
		}
		target := targets[index]
		d.closeReplyPicker()
		d.openReplyComposer(target.ID)
	})
	picker.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEsc:
			d.closeReplyPicker()
		case tcell.KeyCtrlR, tcell.KeyF5:
			d.load()
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyEnter:
			return ev
		}
		return nil
	})
	footer := tview.NewTextView().SetText(fmt.Sprintf("%d threads · ↑/↓: choose · Enter: reply · Esc/CtrlC: cancel", len(targets)))
	page := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(picker, 0, 1, true).AddItem(footer, 2, 0, false)
	d.table.painter.AddPage("reply-picker", page, true, true)
	d.table.screen.SetFocus(picker)
}

// closeReplyPicker returns to the existing detail text without resetting scroll.
func (d *tableDetail) closeReplyPicker() {
	d.picker = nil
	d.table.painter.RemovePage("reply-picker")
	d.table.screen.SetFocus(d.text)
}

// openReplyComposer captures the chosen root in the submit callback; an absent
// reply handler stays unavailable rather than falling back to a top-level comment.
func (d *tableDetail) openReplyComposer(rootID string) {
	callback := d.view.SubmitReply
	var submit func(string) error
	if callback != nil {
		submit = func(body string) error { return callback(rootID, body) }
	}
	d.showComposer(rootID, submit)
}

// openComposer uses the captured top-level comment handler independently of
// thread selection, keeping the c action distinct from a native reply.
func (d *tableDetail) openComposer() {
	if d.picker != nil || d.composer != nil || d.prompt != nil {
		return
	}
	if d.view.SubmitComment == nil {
		d.status.SetText("Commenting unavailable.\n" + detailKeys)
		return
	}
	d.showComposer("", d.view.SubmitComment)
}

// showComposer binds a draft to its submit callback and load generation while
// leaving Enter and Tab available for multiline text instead of form submission.
func (d *tableDetail) showComposer(replyID string, submit func(string) error) {
	if d.picker != nil || d.composer != nil || d.prompt != nil {
		return
	}
	kind, title := "Comment", "New comment"
	if replyID != "" {
		kind = "Reply"
		title = plainReplyText("Reply to thread " + replyID + " on " + d.view.Title)
	}
	c := &commentComposer{
		Flex:    tview.NewFlex().SetDirection(tview.FlexRow),
		draft:   tview.NewTextArea().SetLabel(kind + " for " + tview.Escape(d.view.Title) + ": "),
		status:  tview.NewTextView().SetText(composerKeys),
		submit:  submit,
		kind:    kind,
		replyID: replyID,
		loadID:  d.loadID,
	}
	d.composer = c
	c.draft.SetBorder(true).SetTitle(tview.Escape(title))
	c.post = tview.NewButton("Post").SetSelectedFunc(d.submit).SetDisabled(submit == nil)
	if submit == nil {
		c.status.SetText(replyPostingUnavailable + "\n" + composerKeys)
	}
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
		case ev.Key() == tcell.KeyBacktab:
			items := []tview.Primitive{c.draft, c.cancel, c.post}
			for i, item := range items {
				if item.HasFocus() {
					d.table.screen.SetFocus(items[(i+len(items)-1)%len(items)])
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

// cancelComposer requires confirmation before discarding nonempty text and
// refuses cancellation during a write so its outcome cannot be hidden.
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

// closeComposer drops the editor only after successful posting or explicit
// discard, returning focus without replacing the detail text.
func (d *tableDetail) closeComposer() {
	d.table.painter.RemovePage("composer")
	d.composer = nil
	d.table.screen.SetFocus(d.text)
}

// submit locks the draft during one write and clears it only when the callback
// confirms success. Errors retain text and target; a later reload failure must
// not turn a confirmed post into an invitation to submit it again.
func (d *tableDetail) submit() {
	c := d.composer
	if c == nil || c.posting {
		return
	}
	if c.replyID != "" && c.loadID != d.loadID {
		c.post.SetDisabled(true)
		c.status.SetText(replySelectionExpired)
		return
	}
	if c.submit == nil {
		c.status.SetText(replyPostingUnavailable + "\n" + composerKeys)
		return
	}
	body := c.draft.GetText()
	if strings.TrimSpace(body) == "" {
		c.status.SetText(c.kind + " must not be empty.\n" + composerKeys)
		d.table.screen.SetFocus(c.draft)
		return
	}
	c.posting = true
	c.draft.SetDisabled(true)
	c.post.SetDisabled(true)
	c.cancel.SetDisabled(true)
	c.status.SetText("Posting... Please wait; draft is locked until the request completes.")
	submit := c.submit
	go func() {
		err := submit(body)
		d.table.screen.QueueUpdateDraw(func() {
			if d.table.detail != d || d.composer != c {
				return
			}
			c.posting = false
			if err != nil {
				c.draft.SetDisabled(false)
				c.post.SetDisabled(c.replyID != "" && c.loadID != d.loadID)
				c.cancel.SetDisabled(false)
				message := "Post failed: draft kept. CtrlS/Post: retry; Esc/CtrlC: cancel.\n"
				if c.replyID != "" {
					message = "Reply not confirmed; check thread before retrying. Draft kept. Esc/CtrlC: cancel.\n"
				}
				c.status.SetText(message + err.Error())
				d.table.screen.SetFocus(c.draft)
				return
			}
			c.draft.SetText("", false)
			d.closeComposer()
			d.posted = c.kind
			d.load()
		})
	}()
}

// commentComposer retains the target and load generation with the draft, while
// posting guards both keyboard editing and paste against duplicate submissions.
type commentComposer struct {
	*tview.Flex
	draft   *tview.TextArea
	status  *tview.TextView
	post    *tview.Button
	cancel  *tview.Button
	posting bool
	submit  func(string) error
	kind    string
	replyID string
	loadID  uint64
}

// PasteHandler blocks pasted text while posting because the pinned tview
// TextArea's disabled state alone does not prevent paste from changing the draft.
func (c *commentComposer) PasteHandler() func(string, func(tview.Primitive)) {
	return func(text string, setFocus func(tview.Primitive)) {
		// The pinned tview TextArea accepts paste even when disabled.
		if !c.posting {
			c.Flex.PasteHandler()(text, setFocus)
		}
	}
}
