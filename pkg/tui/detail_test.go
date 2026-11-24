package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

type detailHarness struct {
	t      *testing.T
	table  *Table
	screen tcell.SimulationScreen
	frames chan struct{}
	done   chan struct{}
	err    error
}

func newDetailHarness(t *testing.T, opts ...TableOption) *detailHarness {
	t.Helper()
	h := &detailHarness{t: t, table: NewTable(opts...), screen: tcell.NewSimulationScreen("UTF-8"), frames: make(chan struct{}, 1), done: make(chan struct{})}
	h.table.screen.SetScreen(h.screen)
	h.screen.SetSize(100, 24)
	h.table.screen.SetAfterDrawFunc(func(tcell.Screen) {
		select {
		case h.frames <- struct{}{}:
		default:
		}
	})
	data := TableData{{"KEY", "SUMMARY"}}
	for i := 1; i <= 60; i++ {
		data = append(data, []string{fmt.Sprintf("TEST-%d", i), strings.Repeat("summary ", 20)})
	}
	go func() {
		h.err = h.table.Paint(data)
		close(h.done)
	}()
	await(t, h.frames)
	t.Cleanup(func() {
		h.table.screen.Stop()
		await(t, h.done)
		if h.err != nil {
			t.Errorf("application: %v", h.err)
		}
	})
	return h
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for application/callback")
		var zero T
		return zero
	}
}

func (h *detailHarness) ui(fn func()) {
	h.t.Helper()
	h.table.screen.QueueUpdateDraw(fn)
}

func (h *detailHarness) wait(description string, condition func() bool) {
	h.t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		var ok bool
		h.table.screen.QueueUpdate(func() { ok = condition() })
		if ok {
			return
		}
		select {
		case <-h.frames:
		case <-timer.C:
			h.t.Fatalf("timed out: %s", description)
		}
	}
}

func (h *detailHarness) events(events ...tcell.Event) {
	h.t.Helper()
	// A sentinel in the same event queue is a barrier, unlike QueueUpdate,
	// whose updates may be selected before pending key events.
	barrier := make(chan struct{})
	var previous func(*tcell.EventKey) *tcell.EventKey
	h.table.screen.QueueUpdate(func() {
		previous = h.table.screen.GetInputCapture()
		h.table.screen.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			if ev.Key() == tcell.KeyF12 {
				close(barrier)
				return nil
			}
			if previous != nil {
				return previous(ev)
			}
			return ev
		})
	})
	for _, ev := range events {
		h.table.screen.QueueEvent(ev)
	}
	h.table.screen.QueueEvent(tcell.NewEventKey(tcell.KeyF12, 0, tcell.ModNone))
	select {
	case <-barrier:
	case <-h.done:
		h.t.Fatal("application exited while handling input")
	case <-time.After(5 * time.Second):
		h.t.Fatal("timed out waiting for input barrier")
	}
	h.table.screen.QueueUpdate(func() { h.table.screen.SetInputCapture(previous) })
}

func (h *detailHarness) key(key tcell.Key, modifiers tcell.ModMask) {
	h.t.Helper()
	h.events(tcell.NewEventKey(key, 0, modifiers))
}

func (h *detailHarness) typeText(text string) {
	h.t.Helper()
	events := make([]tcell.Event, 0, len(text))
	for _, r := range text {
		events = append(events, tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	h.events(events...)
}

func (h *detailHarness) paste(text string) {
	h.t.Helper()
	events := make([]tcell.Event, 0, len(text)+2)
	events = append(events, tcell.NewEventPaste(true))
	for _, r := range text {
		key := tcell.KeyRune
		switch r {
		case '\n':
			key = tcell.KeyEnter
		case '\t':
			key = tcell.KeyTab
		}
		events = append(events, tcell.NewEventKey(key, r, tcell.ModNone))
	}
	events = append(events, tcell.NewEventPaste(false))
	h.events(events...)
}

func (h *detailHarness) open() {
	h.t.Helper()
	h.typeText("v")
	h.wait("detail loaded", func() bool {
		return h.table.detail != nil && !strings.Contains(h.table.detail.status.GetText(false), "Loading...")
	})
}

func (h *detailHarness) screenText() string {
	cells, width, _ := h.screen.GetContents()
	var text strings.Builder
	for i, cell := range cells {
		text.WriteString(string(cell.Runes))
		if (i+1)%width == 0 {
			text.WriteByte('\n')
		}
	}
	return text.String()
}

func detailOption(view DetailView) TableOption {
	return WithDetailViewFunc(func(int, int, any) DetailView { return view })
}

type commentRequest struct {
	body   string
	result chan error
}

func commentCallback(requests chan<- commentRequest) func(string) error {
	return func(body string) error {
		request := commentRequest{body: body, result: make(chan error, 1)}
		requests <- request
		return <-request.result
	}
}

func TestDetailComposerSubmission(t *testing.T) {
	for _, method := range []string{"CtrlS", "ShiftEnter", "Post button"} {
		t.Run(method, func(t *testing.T) {
			requests := make(chan commentRequest, 10)
			var loads atomic.Int32
			h := newDetailHarness(t, detailOption(DetailView{
				Title:         "TEST-1",
				Load:          func() (string, error) { return fmt.Sprintf("details %d", loads.Add(1)), nil },
				SubmitComment: commentCallback(requests),
			}))
			h.open()
			h.typeText("c")
			h.typeText("  first")
			h.key(tcell.KeyEnter, tcell.ModNone)
			h.key(tcell.KeyEnter, tcell.ModAlt)
			h.typeText("cvqm?  ")
			want := "  first\n\ncvqm?  "
			var composer *commentComposer
			h.ui(func() {
				composer = h.table.detail.composer
				if got := composer.draft.GetText(); got != want {
					t.Errorf("Enter/literal keys: got %q, want %q", got, want)
				}
			})
			switch method {
			case "CtrlS":
				h.key(tcell.KeyCtrlS, tcell.ModNone)
			case "ShiftEnter":
				h.key(tcell.KeyEnter, tcell.ModShift)
			case "Post button":
				h.key(tcell.KeyTab, tcell.ModNone)
				h.ui(func() {
					if !composer.post.HasFocus() {
						t.Error("Tab did not focus Post")
					}
				})
				h.key(tcell.KeyEnter, tcell.ModNone)
			}
			request := await(t, requests)
			if request.body != want {
				t.Errorf("submitted body %q, want original whitespace %q", request.body, want)
			}
			// These must remain responsive but cannot edit, cancel, or issue another POST.
			h.key(tcell.KeyCtrlS, tcell.ModNone)
			h.key(tcell.KeyEnter, tcell.ModShift)
			h.key(tcell.KeyEsc, tcell.ModNone)
			h.key(tcell.KeyCtrlC, tcell.ModNone)
			h.key(tcell.KeyTab, tcell.ModNone)
			h.typeText("qcv")
			h.paste("q\nc\nnew text")
			h.ui(func() {
				if !composer.posting || composer.draft.GetText() != want || h.table.detail.composer != composer {
					t.Error("pending submission changed or lost draft")
				}
				if !strings.Contains(composer.status.GetText(false), "Posting...") {
					t.Error("missing posting status")
				}
			})
			select {
			case <-requests:
				t.Error("duplicate submission")
			default:
			}
			request.result <- nil
			h.wait("success and refresh", func() bool {
				d := h.table.detail
				return d.composer == nil && d.text.GetText(false) == "details 2" && strings.Contains(d.status.GetText(false), "Comment posted.")
			})
			h.ui(func() {
				if composer.draft.GetText() != "" || !h.table.detail.text.HasFocus() {
					t.Error("success did not clear draft and restore detail focus")
				}
			})
		})
	}
}

func TestComposerPasteAndNavigation(t *testing.T) {
	requests := make(chan commentRequest, 1)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil }, SubmitComment: commentCallback(requests),
	}))
	h.open()
	h.typeText("c")
	h.typeText("cvqm?")
	paste := "\nq\nc\nv\nm\n?\n[red]\tend"
	h.paste(paste)
	want := "cvqm?" + paste
	var c *commentComposer
	h.ui(func() {
		c = h.table.detail.composer
		if c == nil || c.draft.GetText() != want || !c.draft.HasFocus() {
			t.Errorf("paste/literal shortcuts changed page, focus, or body")
		}
		if !strings.Contains(c.draft.GetLabel(), "TEST-1") {
			t.Error("composer not labeled with target")
		}
	})
	h.key(tcell.KeyTab, tcell.ModNone)
	h.ui(func() {
		if !c.post.HasFocus() {
			t.Error("Post not focused")
		}
	})
	h.key(tcell.KeyTab, tcell.ModNone)
	h.ui(func() {
		if !c.cancel.HasFocus() {
			t.Error("Cancel not focused")
		}
	})
	h.key(tcell.KeyTab, tcell.ModNone)
	h.ui(func() {
		if !c.draft.HasFocus() {
			t.Error("Tab did not wrap to draft")
		}
	})
	h.key(tcell.KeyBacktab, tcell.ModNone)
	h.ui(func() {
		if !c.cancel.HasFocus() {
			t.Error("Backtab did not wrap to Cancel")
		}
	})
	select {
	case <-requests:
		t.Error("editing/paste/navigation unexpectedly posted")
	default:
	}
}

func TestComposerDiscardConfirmation(t *testing.T) {
	requests := make(chan commentRequest, 1)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil }, SubmitComment: commentCallback(requests),
	}))
	h.open()
	h.typeText("c")
	want := "keep this draft"
	h.typeText(want)
	var c *commentComposer
	h.ui(func() { c = h.table.detail.composer })
	h.key(tcell.KeyBacktab, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if name, _ := h.table.painter.GetFrontPage(); name != "discard" {
			t.Error("nonempty Cancel did not confirm discard")
		}
	})
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if !c.draft.HasFocus() || c.draft.GetText() != want || h.table.painter.HasPage("discard") {
			t.Error("default action did not keep editing")
		}
	})
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.ui(func() {
		if !c.draft.HasFocus() || c.draft.GetText() != want {
			t.Error("Escape in discard confirmation did not keep draft")
		}
	})
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.key(tcell.KeyTab, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.composer != nil || !h.table.detail.text.HasFocus() {
			t.Error("explicit discard did not return to details")
		}
	})
	select {
	case <-requests:
		t.Error("discard confirmation unexpectedly posted")
	default:
	}
}

func TestComposerEmptyCancellation(t *testing.T) {
	requests := make(chan commentRequest, 1)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil }, SubmitComment: commentCallback(requests),
	}))
	h.open()
	for _, cancel := range []tcell.Key{tcell.KeyEsc, tcell.KeyEnter} {
		h.typeText("c")
		if cancel == tcell.KeyEnter {
			h.key(tcell.KeyBacktab, tcell.ModNone)
		}
		h.key(cancel, tcell.ModNone)
		h.ui(func() {
			if h.table.detail.composer != nil || h.table.painter.HasPage("discard") {
				t.Error("empty draft cancellation should return without confirmation")
			}
		})
	}
	select {
	case <-requests:
		t.Error("empty cancellation unexpectedly posted")
	default:
	}
}

func TestComposerRejectsWhitespace(t *testing.T) {
	requests := make(chan commentRequest, 1)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil }, SubmitComment: commentCallback(requests),
	}))
	h.open()
	h.typeText("c")
	body := " \t\n\u2003 "
	h.paste(body)
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	h.ui(func() {
		c := h.table.detail.composer
		if c.posting || c.draft.GetText() != body || !strings.Contains(c.status.GetText(false), "must not be empty") {
			t.Error("whitespace submission was not rejected with draft preserved")
		}
	})
	select {
	case <-requests:
		t.Error("whitespace body reached submit callback")
	default:
	}
}

func TestComposerFailureRetainsBodyForExplicitRetry(t *testing.T) {
	requests := make(chan commentRequest, 2)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil }, SubmitComment: commentCallback(requests),
	}))
	h.open()
	h.typeText("c")
	h.paste("  body\nkept  ")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	first := await(t, requests)
	first.result <- errors.New("network failure [red]")
	h.wait("post failure", func() bool { return strings.Contains(h.table.detail.composer.status.GetText(false), "Post failed:") })
	h.ui(func() {
		c := h.table.detail.composer
		if c.draft.GetText() != first.body || c.posting || c.draft.GetDisabled() || c.post.IsDisabled() || !c.draft.HasFocus() {
			t.Error("failed submission did not preserve editable draft")
		}
	})
	select {
	case <-requests:
		t.Error("POST retried automatically")
	default:
	}
	h.typeText("!")
	h.key(tcell.KeyTab, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	second := await(t, requests)
	if second.body != first.body+"!" {
		t.Errorf("retry body = %q", second.body)
	}
	second.result <- nil
	h.wait("explicit retry success", func() bool { return h.table.detail.composer == nil })
}

func TestPostSuccessReloadFailureDoesNotRestoreDraft(t *testing.T) {
	requests := make(chan commentRequest, 2)
	var loads atomic.Int32
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", SubmitComment: commentCallback(requests),
		Load: func() (string, error) {
			if loads.Add(1) == 2 {
				return "", errors.New("refresh offline")
			}
			return "fresh details", nil
		},
	}))
	h.open()
	h.typeText("c")
	h.typeText("posted once")
	var c *commentComposer
	h.ui(func() { c = h.table.detail.composer })
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	request := await(t, requests)
	request.result <- nil
	h.wait("success distinct from reload failure", func() bool {
		return strings.Contains(h.table.detail.status.GetText(false), "Comment posted. Reload failed (do not repost): refresh offline")
	})
	h.ui(func() {
		if h.table.detail.composer != nil || c.draft.GetText() != "" {
			t.Error("successful POST draft was restored after failed refresh")
		}
	})
	h.typeText("c")
	h.ui(func() {
		if h.table.detail.composer.draft.GetText() != "" {
			t.Error("new composer restored a submitted draft")
		}
	})
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.key(tcell.KeyF5, tcell.ModNone)
	h.wait("explicit reload only", func() bool { return h.table.detail.status.GetText(false) == "Comment posted. "+detailKeys })
	if loads.Load() != 3 {
		t.Errorf("load count = %d, want 3", loads.Load())
	}
	select {
	case <-requests:
		t.Error("reload retried POST")
	default:
	}
}

func TestDetailRenderingScrollingAndListState(t *testing.T) {
	body := "[red]literal[-] [\"region\"] \x1b[31mRED\x1b[0m\n" + strings.Repeat("filler\n", 60) + "bottom"
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-35 [blue]title", Load: func() (string, error) { return body, nil },
		SubmitComment: func(string) error { return nil },
	}))
	h.ui(func() { h.table.view.Select(35, 0).SetOffset(30, 1) })
	var row, col, offsetRow, offsetCol int
	h.ui(func() {
		row, col = h.table.view.GetSelection()
		offsetRow, offsetCol = h.table.view.GetOffset()
	})
	h.open()
	h.ui(func() {
		text := h.screenText()
		for _, want := range []string{"TEST-35 [blue]title", "[red]literal[-] [\"region\"] RED"} {
			if !strings.Contains(text, want) {
				t.Errorf("rendered screen lacks literal %q", want)
			}
		}
		cells, _, _ := h.screen.GetContents()
		var styled bool
		for i := 0; i+2 < len(cells); i++ {
			if string(cells[i].Runes)+string(cells[i+1].Runes)+string(cells[i+2].Runes) == "RED" {
				fg, _, _ := cells[i].Style.Decompose()
				styled = fg != tcell.ColorDefault
			}
		}
		if !styled {
			t.Error("ANSI content lost color")
		}
	})
	h.key(tcell.KeyEnd, tcell.ModNone)
	h.ui(func() {
		if scroll, _ := h.table.detail.text.GetScrollOffset(); scroll == 0 || !strings.Contains(h.screenText(), "bottom") {
			t.Error("details did not scroll to bottom")
		}
	})
	for _, back := range []tcell.Key{tcell.KeyRune, tcell.KeyEsc} {
		if back == tcell.KeyRune {
			h.typeText("q")
		} else {
			h.open()
			h.key(back, tcell.ModNone)
		}
		h.ui(func() {
			r, c := h.table.view.GetSelection()
			or, oc := h.table.view.GetOffset()
			if h.table.detail != nil || !h.table.view.HasFocus() || r != row || c != col || or != offsetRow || oc != offsetCol {
				t.Errorf("back changed list: selection (%d,%d), offset (%d,%d); want (%d,%d), (%d,%d)", r, c, or, oc, row, col, offsetRow, offsetCol)
			}
		})
	}
}

type detailResult struct {
	body string
	err  error
}

func TestDetailLoadErrorRetryAndStaleCompletion(t *testing.T) {
	requests := make(chan chan detailResult, 10)
	var boundRows []int
	h := newDetailHarness(t, WithDetailViewFunc(func(row, col int, data any) DetailView {
		boundRows = append(boundRows, row)
		return DetailView{
			Title: data.(TableData).Get(row, 0),
			Load: func() (string, error) {
				result := make(chan detailResult, 1)
				requests <- result
				r := <-result
				return r.body, r.err
			},
		}
	}))
	h.typeText("v")
	first := await(t, requests)
	first <- detailResult{err: errors.New("offline")}
	h.wait("load error", func() bool { return strings.Contains(h.table.detail.status.GetText(false), "Load failed: offline") })
	h.key(tcell.KeyF5, tcell.ModNone)
	old := await(t, requests)
	// Closing while a worker is blocked must not block the list or next detail.
	h.typeText("q")
	h.key(tcell.KeyDown, tcell.ModNone)
	h.typeText("v")
	current := await(t, requests)
	current <- detailResult{body: "new target"}
	h.wait("new target loaded", func() bool { return h.table.detail.text.GetText(false) == "new target" })
	h.table.screen.QueueUpdate(func() {})
	for len(h.frames) > 0 {
		<-h.frames
	}
	old <- detailResult{body: "stale target"}
	await(t, h.frames)
	h.ui(func() {
		if h.table.detail.text.GetText(false) != "new target" || h.table.detail.view.Title != "TEST-2" {
			t.Error("late result overwrote reopened detail")
		}
		if fmt.Sprint(boundRows) != "[1 2]" {
			t.Errorf("targets bound at open: %v", boundRows)
		}
	})
	h.key(tcell.KeyCtrlR, tcell.ModNone)
	slowReload := await(t, requests)
	h.key(tcell.KeyF5, tcell.ModNone)
	latestReload := await(t, requests)
	latestReload <- detailResult{body: "reloaded"}
	h.wait("latest reload", func() bool { return h.table.detail.text.GetText(false) == "reloaded" })
	for len(h.frames) > 0 {
		<-h.frames
	}
	slowReload <- detailResult{err: errors.New("stale reload error")}
	await(t, h.frames)
	h.ui(func() {
		if h.table.detail.text.GetText(false) != "reloaded" || h.table.detail.status.GetText(false) != detailKeys {
			t.Error("older reload overwrote the latest result")
		}
	})
}

func TestDetailInvalidSelectionAndMissingCallbacks(t *testing.T) {
	var called bool
	h := newDetailHarness(t, WithDetailViewFunc(func(int, int, any) DetailView { called = true; return DetailView{} }))
	for _, tc := range []struct {
		name string
		data TableData
		r, c int
	}{
		{"no data", nil, 1, 0},
		{"empty header", TableData{{}, {"item"}}, 1, 0},
		{"header selected", TableData{{"KEY"}, {"item"}}, 0, 0},
		{"negative row", TableData{{"KEY"}, {"item"}}, -1, 0},
		{"row outside data", TableData{{"KEY"}, {"item"}}, 9, 0},
		{"negative column", TableData{{"KEY"}, {"item"}}, 1, -1},
		{"column outside header", TableData{{"KEY"}, {"item"}}, 1, 1},
		{"short row", TableData{{"KEY", "SUMMARY"}, {"item"}}, 1, 0},
	} {
		h.ui(func() {
			h.table.data = tc.data
			h.table.view.Select(tc.r, tc.c)
			h.table.openDetail()
			if called || h.table.detail != nil {
				t.Errorf("%s: invalid selection invoked detail callback", tc.name)
			}
		})
	}
	h.ui(func() { h.table.data = TableData{{"KEY"}, {"item"}}; h.table.view.Select(1, 0) })
	h.open()
	h.typeText("c")
	h.key(tcell.KeyF5, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.composer != nil || !strings.Contains(h.table.detail.status.GetText(false), "no loader") {
			t.Error("nil loader/submit not handled")
		}
	})
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.ui(func() { WithDetailViewFunc(nil)(h.table) })
	h.typeText("v")
	h.ui(func() {
		if h.table.detail != nil || !h.table.view.HasFocus() {
			t.Error("nil detail and legacy callbacks changed page")
		}
	})
}

func TestComposerCtrlCDraftAndConfirmation(t *testing.T) {
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil },
		SubmitComment: func(string) error { t.Error("cancel posted a comment"); return nil },
	}))
	h.open()
	h.typeText("c")
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.composer != nil || h.table.painter.HasPage("discard") || !h.table.detail.text.HasFocus() {
			t.Error("CtrlC did not close empty composer to detail")
		}
	})
	h.typeText("c")
	h.typeText("keep this draft")
	var c *commentComposer
	h.ui(func() { c = h.table.detail.composer })
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.ui(func() {
		if !h.table.painter.HasPage("discard") || h.table.detail.composer != c || c.draft.GetText() != "keep this draft" {
			t.Error("CtrlC did not confirm cancellation of a nonempty draft")
		}
	})
	h.key(tcell.KeyTab, tcell.ModNone)
	// Even with Discard selected, CtrlC must not activate it or quit.
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.ui(func() {
		if !h.table.painter.HasPage("discard") || h.table.detail.composer != c || c.draft.GetText() != "keep this draft" {
			t.Error("CtrlC bypassed discard confirmation")
		}
	})
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if h.table.painter.HasPage("discard") || !c.draft.HasFocus() || c.draft.GetText() != "keep this draft" {
			t.Error("CtrlC confirmation default did not safely keep editing")
		}
	})
}

func TestComposerCtrlCPendingPostAndFailureCancel(t *testing.T) {
	requests := make(chan commentRequest, 2)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "details", nil }, SubmitComment: commentCallback(requests),
	}))
	h.open()
	h.typeText("c")
	h.typeText("pending draft")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	request := await(t, requests)
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.ui(func() {
		c := h.table.detail.composer
		if c == nil || !c.posting || c.draft.GetText() != request.body || h.table.painter.HasPage("discard") {
			t.Error("CtrlC interrupted pending POST or changed its draft")
		}
	})
	request.result <- errors.New("post offline")
	h.wait("failure after interrupted POST", func() bool {
		return strings.Contains(h.table.detail.composer.status.GetText(false), "Post failed:")
	})
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.ui(func() {
		if !h.table.painter.HasPage("discard") || h.table.detail.composer.draft.GetText() != request.body {
			t.Error("failed POST draft did not require discard confirmation")
		}
	})
	h.key(tcell.KeyTab, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.composer != nil || !h.table.detail.text.HasFocus() {
			t.Error("explicit discard after failure did not return to detail")
		}
	})
	select {
	case <-requests:
		t.Error("CtrlC retried POST")
	default:
	}
}

func TestTableCtrlCOutsideComposerExits(t *testing.T) {
	for _, state := range []string{"generic list", "opted-in list", "detail", "empty composer closed"} {
		t.Run(state, func(t *testing.T) {
			var opts []TableOption
			if state != "generic list" {
				opts = append(opts, detailOption(DetailView{
					Load: func() (string, error) { return "details", nil }, SubmitComment: func(string) error { return nil },
				}))
			}
			h := newDetailHarness(t, opts...)
			if state == "detail" || state == "empty composer closed" {
				h.open()
			}
			if state == "empty composer closed" {
				h.typeText("c")
				h.key(tcell.KeyCtrlC, tcell.ModNone)
			}
			h.table.screen.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone))
			await(t, h.done)
			if h.table.screen.GetInputCapture() != nil {
				t.Error("application capture was not restored after exit")
			}
		})
	}
}

func TestTableComposerCapturePreservesPriorCapture(t *testing.T) {
	var interrupts atomic.Int32
	prior := func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyCtrlC {
			interrupts.Add(1)
		}
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'x' {
			return tcell.NewEventKey(tcell.KeyRune, 'X', tcell.ModNone)
		}
		if ev.Key() == tcell.KeyF6 {
			return nil
		}
		return ev
	}
	h := newDetailHarness(t,
		func(table *Table) { table.screen.SetInputCapture(prior) },
		detailOption(DetailView{
			Load: func() (string, error) { return "details", nil }, SubmitComment: func(string) error { return nil },
		}),
	)
	h.open()
	h.typeText("c")
	h.typeText("x")
	h.key(tcell.KeyF6, tcell.ModNone)
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.composer.draft.GetText() != "X" || interrupts.Load() != 0 {
			t.Error("prior capture was lost, or received a protected interrupt")
		}
	})
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.key(tcell.KeyTab, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.table.screen.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone))
	await(t, h.done)
	if interrupts.Load() != 1 {
		t.Errorf("outside-composer CtrlC not delegated exactly once: %d", interrupts.Load())
	}
	restored := h.table.screen.GetInputCapture()
	if restored == nil || restored(tcell.NewEventKey(tcell.KeyF6, 0, tcell.ModNone)) != nil {
		t.Error("prior capture was not restored after application exit")
	}
}

func TestStatusGuidancePrecedesLiteralVerboseErrors(t *testing.T) {
	loadError := "[red]LOAD\n" + strings.Repeat("load details\n", 10)
	postError := "[blue]POST\n" + strings.Repeat("post details\n", 10)
	h := newDetailHarness(t, detailOption(DetailView{
		Title: "TEST-1", Load: func() (string, error) { return "", errors.New(loadError) },
		SubmitComment: func(string) error { return errors.New(postError) },
	}))
	h.ui(func() { h.screen.SetSize(40, 24) })
	h.open()
	h.ui(func() {
		text := h.screenText()
		if !strings.Contains(text, "CtrlR/F5: retry loading.") || !strings.Contains(text, "[red]LOAD") || !strings.Contains(h.table.detail.status.GetText(false), loadError) {
			t.Error("load retry guidance or literal multiline error was lost")
		}
	})
	h.typeText("c")
	h.typeText("draft")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	h.wait("verbose post failure", func() bool {
		return strings.Contains(h.table.detail.composer.status.GetText(false), postError)
	})
	h.ui(func() {
		text := h.screenText()
		if !strings.Contains(text, "retry") || !strings.Contains(text, "cancel") || !strings.Contains(text, "[blue]POST") {
			t.Error("post retry/cancel guidance or literal error clipped before verbose details")
		}
	})
}

func TestTableDetailOptInPreservesLegacyPager(t *testing.T) {
	tee, err := exec.LookPath("tee")
	if err != nil {
		t.Skip("tee unavailable for external-pager check")
	}
	output := filepath.Join(t.TempDir(), "pager-output")
	t.Setenv("TERM", "xterm")
	t.Setenv("JIRA_PAGER", tee+" "+output)
	legacyCalls := make(chan struct{}, 2)
	legacy := WithViewModeFunc(func(row, col int, data interface{}) (func() interface{}, func(interface{}) (string, error)) {
		legacyCalls <- struct{}{}
		return func() interface{} { return data.(TableData).Get(row, 0) }, func(value interface{}) (string, error) { return value.(string), nil }
	})
	h := newDetailHarness(t, legacy)
	h.typeText("v")
	await(t, legacyCalls)
	h.wait("legacy pager returned", func() bool { name, _ := h.table.painter.GetFrontPage(); return name == "primary" })
	content, err := os.ReadFile(output)
	if err != nil || string(content) != "TEST-1" {
		t.Fatalf("external pager content = %q, error = %v", content, err)
	}
	h.ui(func() { detailOption(DetailView{Load: func() (string, error) { return "opted in", nil }})(h.table) })
	h.open()
	h.ui(func() {
		if h.table.detail.text.GetText(false) != "opted in" {
			t.Error("detail callback did not take precedence")
		}
	})
	select {
	case <-legacyCalls:
		t.Error("opted-in detail invoked legacy pager")
	default:
	}
}
