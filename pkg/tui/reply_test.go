package tui

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func replyTestContent() DetailContent {
	return DetailContent{
		Text: "issue details",
		ReplyTargets: []ReplyTarget{
			{ID: "root-1", Author: "Alice", Created: "2026-11-23", Preview: "first thread"},
			{ID: "root-2", Author: "Bob", Created: "2026-11-24", Preview: "second thread"},
		},
	}
}

func replyTestView(t *testing.T, content DetailContent, submit func(string, string) error) DetailView {
	t.Helper()
	return DetailView{
		Title:       "TEST-1",
		Load:        func() (DetailContent, error) { return content, nil },
		SubmitReply: submit,
		SubmitComment: func(string) error {
			t.Error("reply fell back to top-level comment")
			return errors.New("unexpected top-level comment")
		},
	}
}

type replyRequest struct {
	rootID string
	body   string
	result chan error
}

func replyCallback(requests chan<- replyRequest) func(string, string) error {
	return func(rootID, body string) error {
		request := replyRequest{rootID: rootID, body: body, result: make(chan error, 1)}
		requests <- request
		return <-request.result
	}
}

func TestReplyPickerOptionsAndEscaping(t *testing.T) {
	content := replyTestContent()
	content.ReplyTargets[0] = ReplyTarget{
		ID: "root-[red]", Author: "\x1b[31mAlice[blue]\x1b[0m\nSmith\u202e", Created: "2026\r\n[green]",
		Preview: "\x1b]8;;https://example.com\a[red]preview\x1b]8;;\a\ttext",
	}
	content.ReplyTargets[1].Preview = strings.Repeat("p", replyPreviewLimit+10)
	h := newDetailHarness(t, detailOption(replyTestView(t, content, nil)))
	h.open()
	h.typeText("r")
	h.ui(func() {
		picker := h.table.detail.picker
		if picker == nil || picker.GetItemCount() != 2 || h.table.detail.composer != nil {
			t.Error("r must open a chooser, not an editor")
			return
		}
		if picker.GetTitle() != "Which thread do you want to reply to?" {
			t.Error("missing explicit thread question")
		}
		main, preview := picker.GetItemText(0)
		if main != tview.Escape("root-[red] · Alice[blue] Smith · 2026 [green]") || preview != tview.Escape("[red]preview text") {
			t.Errorf("unsafe or reordered option: %q / %q", main, preview)
		}
		if strings.ContainsFunc(main+preview, unicode.IsControl) {
			t.Error("picker contains terminal controls")
		}
		main, preview = picker.GetItemText(1)
		if !strings.HasPrefix(main, "root-2 · Bob") || preview != strings.Repeat("p", replyPreviewLimit)+"…" {
			t.Error("second root order or short preview changed")
		}
		if !strings.Contains(h.screenText(), "root-[red]") || !strings.Contains(h.screenText(), "2 threads") {
			t.Error("literal ID or root count not visible")
		}
	})
	h.key(tcell.KeyDown, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.picker.GetCurrentItem() != 1 {
			t.Error("Down did not select the next option")
		}
	})
	h.key(tcell.KeyUp, tcell.ModNone)
	h.typeText("rc ")
	h.ui(func() {
		if h.table.detail.picker.GetCurrentItem() != 0 || h.table.detail.composer != nil {
			t.Error("picker shortcuts bypassed explicit Enter selection")
		}
	})
}

func TestReplySingleRootRequiresExplicitSelection(t *testing.T) {
	content := replyTestContent()
	content.ReplyTargets = content.ReplyTargets[:1]
	h := newDetailHarness(t, detailOption(replyTestView(t, content, nil)))
	h.open()
	h.typeText("r")
	h.ui(func() {
		if h.table.detail.picker == nil || h.table.detail.composer != nil {
			t.Error("one root must still require a chooser")
		}
	})
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		d := h.table.detail
		c := d.composer
		if d.picker != nil || c == nil || c.replyID != "root-1" || c.draft.GetText() != "" {
			t.Error("selection leaked r/Enter or chose the wrong root")
			return
		}
		if c.draft.GetTitle() != "Reply to thread root-1 on TEST-1" {
			t.Error("reply target and issue not clearly labeled")
		}
		d.openReplyPicker()
		d.openComposer()
		if d.composer != c || d.picker != nil {
			t.Error("picker/composer opened twice")
		}
	})
	h.typeText("r")
	h.ui(func() {
		if h.table.detail.composer.draft.GetText() != "r" {
			t.Error("r is not literal in the editor")
		}
	})
}

func TestReplyPickerCancelPreservesContext(t *testing.T) {
	for _, cancel := range []tcell.Key{tcell.KeyEsc, tcell.KeyCtrlC} {
		t.Run(tcell.KeyNames[cancel], func(t *testing.T) {
			content := replyTestContent()
			content.Text = strings.Repeat("details\n", 60) + "bottom"
			h := newDetailHarness(t, detailOption(replyTestView(t, content, nil)))
			h.ui(func() { h.table.view.Select(35, 0).SetOffset(30, 1) })
			h.open()
			h.key(tcell.KeyEnd, tcell.ModNone)
			var row, col, offsetRow, offsetCol, scrollRow, scrollCol int
			h.ui(func() {
				row, col = h.table.view.GetSelection()
				offsetRow, offsetCol = h.table.view.GetOffset()
				scrollRow, scrollCol = h.table.detail.text.GetScrollOffset()
			})
			h.typeText("r")
			h.key(tcell.KeyDown, tcell.ModNone)
			h.key(cancel, tcell.ModNone)
			h.ui(func() {
				d := h.table.detail
				r, c := h.table.view.GetSelection()
				or, oc := h.table.view.GetOffset()
				sr, sc := d.text.GetScrollOffset()
				if d.picker != nil || d.composer != nil || !d.text.HasFocus() {
					t.Error("picker cancellation did not return to detail focus")
				}
				if r != row || c != col || or != offsetRow || oc != offsetCol || sr != scrollRow || sc != scrollCol {
					t.Error("picker cancellation changed detail scroll or list context")
				}
			})
		})
	}
}

func TestReplyUnavailableSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content DetailContent
		err     error
		want    string
	}{
		{name: "native disabled", content: DetailContent{ReplyTargets: replyTestContent().ReplyTargets, ReplyUnavailable: "Enable --comment-threads to reply."}, want: "Enable --comment-threads"},
		{name: "nil roots", want: "No reply threads loaded"},
		{name: "zero roots", content: DetailContent{ReplyTargets: []ReplyTarget{}}, want: "No reply threads loaded"},
		{name: "empty ID", content: DetailContent{ReplyTargets: []ReplyTarget{{ID: "root"}, {}}}, want: "invalid target ID"},
		{name: "spaces in ID", content: DetailContent{ReplyTargets: []ReplyTarget{{ID: " root "}}}, want: "invalid target ID"},
		{name: "control ID", content: DetailContent{ReplyTargets: []ReplyTarget{{ID: "root\x1b[31m"}}}, want: "invalid target ID"},
		{name: "format ID", content: DetailContent{ReplyTargets: []ReplyTarget{{ID: "root\u202e"}}}, want: "invalid target ID"},
		{name: "duplicate ID", content: DetailContent{ReplyTargets: []ReplyTarget{{ID: "root"}, {ID: "root"}}}, want: "duplicate target IDs"},
		{name: "load error with roots", content: replyTestContent(), err: errors.New("offline"), want: "detail load failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := replyTestView(t, tc.content, nil)
			view.Load = func() (DetailContent, error) { return tc.content, tc.err }
			h := newDetailHarness(t, detailOption(view))
			h.open()
			h.typeText("r")
			h.ui(func() {
				d := h.table.detail
				if d.picker != nil || d.composer != nil || len(d.replyTargets) != 0 || !strings.Contains(d.status.GetText(false), tc.want) {
					t.Errorf("unavailable snapshot accepted or reason missing: %q", d.status.GetText(false))
				}
			})
		})
	}
}

func TestReplyWithoutBackendIsPreviewOnly(t *testing.T) {
	h := newDetailHarness(t, detailOption(replyTestView(t, replyTestContent(), nil)))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.typeText("draft\ttext")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModShift)
	h.key(tcell.KeyBacktab, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		c := h.table.detail.composer
		if !c.post.IsDisabled() || c.posting || c.draft.GetText() != "draft\ttext" || c.replyID != "root-1" {
			t.Error("nil backend enabled posting or lost reply draft")
		}
		if !strings.Contains(c.status.GetText(false), replyPostingUnavailable) {
			t.Error("preview pretended that posting was available")
		}
	})
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if !h.table.detail.composer.draft.HasFocus() {
			t.Error("preview draft cancellation did not default to keep editing")
		}
	})
}

func TestReplySubmissionPathsAndImmutableBinding(t *testing.T) {
	for _, method := range []string{"CtrlS", "ShiftEnter", "Post button"} {
		t.Run(method, func(t *testing.T) {
			requests := make(chan replyRequest, 2)
			content := replyTestContent()
			h := newDetailHarness(t, detailOption(replyTestView(t, content, replyCallback(requests))))
			h.open()
			h.typeText("r")
			h.key(tcell.KeyDown, tcell.ModNone)
			h.key(tcell.KeyEnter, tcell.ModNone)
			var composer *commentComposer
			h.ui(func() {
				composer = h.table.detail.composer
				if composer.draft.GetText() != "" || composer.replyID != "root-2" {
					t.Error("wrong selection or initial key leaked into editor")
				}
				content.ReplyTargets[1].ID = "changed-source"
				h.table.view.Select(12, 0)
				h.table.detail.view.SubmitReply = func(string, string) error {
					t.Error("submission callback was not bound before posting")
					return errors.New("wrong callback")
				}
			})
			h.typeText(" rcvqm? ")
			h.key(tcell.KeyTab, tcell.ModNone)
			h.key(tcell.KeyEnter, tcell.ModNone)
			h.paste("a\tb\nnext")
			want := " rcvqm? \t\na\tb\nnext"
			switch method {
			case "CtrlS":
				h.key(tcell.KeyCtrlS, tcell.ModNone)
			case "ShiftEnter":
				h.key(tcell.KeyEnter, tcell.ModShift)
			case "Post button":
				h.key(tcell.KeyBacktab, tcell.ModNone)
				h.key(tcell.KeyEnter, tcell.ModNone)
			}
			request := await(t, requests)
			if request.rootID != "root-2" || request.body != want {
				t.Errorf("reply request = %q, %q", request.rootID, request.body)
			}
			h.key(tcell.KeyCtrlS, tcell.ModNone)
			h.key(tcell.KeyCtrlC, tcell.ModNone)
			h.key(tcell.KeyEsc, tcell.ModNone)
			h.paste("not an edit")
			h.ui(func() {
				if !composer.posting || composer.draft.GetText() != want || h.table.detail.composer != composer {
					t.Error("pending reply was cancelled or edited")
				}
			})
			select {
			case <-requests:
				t.Error("duplicate reply")
			default:
			}
			request.result <- nil
			h.wait("reply posted and refreshed", func() bool {
				return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Reply posted. "+detailKeys
			})
			h.ui(func() {
				if composer.draft.GetText() != "" || !h.table.detail.text.HasFocus() {
					t.Error("confirmed reply did not clear draft and restore focus")
				}
			})
		})
	}
}

func TestReplyRejectsWhitespaceWithoutLosingTarget(t *testing.T) {
	requests := make(chan replyRequest, 1)
	h := newDetailHarness(t, detailOption(replyTestView(t, replyTestContent(), replyCallback(requests))))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	body := " \t\n\u2003 "
	h.paste(body)
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	h.ui(func() {
		c := h.table.detail.composer
		if c.posting || c.replyID != "root-1" || c.draft.GetText() != body || !strings.Contains(c.status.GetText(false), "Reply must not be empty") {
			t.Error("whitespace reply was accepted or lost its target/draft")
		}
	})
	select {
	case <-requests:
		t.Error("whitespace reply reached the callback")
	default:
	}
}

func TestReplyComposerEscapesTargetTitle(t *testing.T) {
	content := replyTestContent()
	content.ReplyTargets[0].ID = "root-[red]"
	view := replyTestView(t, content, nil)
	view.Title = "TEST-1 [blue]title"
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		want := "Reply to thread root-[red] on TEST-1 [blue]title"
		if h.table.detail.composer.draft.GetTitle() != tview.Escape(want) || !strings.Contains(h.screenText(), want) {
			t.Error("reply target title was not rendered literally")
		}
	})
}

func TestReplyFailureKeepsTargetAndRequiresExplicitRetry(t *testing.T) {
	requests := make(chan replyRequest, 2)
	h := newDetailHarness(t, detailOption(replyTestView(t, replyTestContent(), replyCallback(requests))))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyDown, tcell.ModNone)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.paste("  body\nkept\t ")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	first := await(t, requests)
	first.result <- errors.New("unknown outcome [red]")
	h.wait("reply not confirmed", func() bool {
		return strings.Contains(h.table.detail.composer.status.GetText(false), "not confirmed; check thread before retrying")
	})
	h.ui(func() {
		c := h.table.detail.composer
		if c.replyID != "root-2" || c.draft.GetText() != first.body || c.posting || c.post.IsDisabled() || !c.draft.HasFocus() {
			t.Error("reply failure lost target or editable body")
		}
		h.table.detail.replyTargets[1].ID = "changed-selection"
	})
	select {
	case <-requests:
		t.Error("reply retried automatically")
	default:
	}
	h.typeText("!")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	second := await(t, requests)
	if second.rootID != first.rootID || second.body != first.body+"!" {
		t.Error("retry did not retain the captured target/body")
	}
	second.result <- nil
	h.wait("reply retry success", func() bool { return h.table.detail.composer == nil })
}

func TestReplySuccessWithReloadFailureNeverRestoresDraft(t *testing.T) {
	requests := make(chan replyRequest, 2)
	var loads atomic.Int32
	view := replyTestView(t, replyTestContent(), replyCallback(requests))
	view.Load = func() (DetailContent, error) {
		if loads.Add(1) == 2 {
			return replyTestContent(), errors.New("refresh offline")
		}
		return replyTestContent(), nil
	}
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.typeText("posted once")
	var c *commentComposer
	h.ui(func() { c = h.table.detail.composer })
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	request := await(t, requests)
	request.result <- nil
	h.wait("confirmed reply with failed reload", func() bool {
		return strings.Contains(h.table.detail.status.GetText(false), "Reply posted. Reload failed (do not repost): refresh offline")
	})
	h.ui(func() {
		if h.table.detail.composer != nil || c.draft.GetText() != "" || len(h.table.detail.replyTargets) != 0 {
			t.Error("refresh failure restored submitted draft or stale targets")
		}
	})
	h.typeText("r")
	h.ui(func() {
		status := h.table.detail.status.GetText(false)
		if h.table.detail.picker != nil || !strings.Contains(status, "detail load failed") || !strings.HasPrefix(status, "Reply posted.") {
			t.Error("failed snapshot allowed selection or lost the confirmed POST outcome")
		}
	})
	h.key(tcell.KeyF5, tcell.ModNone)
	h.wait("explicit reply refresh", func() bool { return h.table.detail.status.GetText(false) == "Reply posted. "+detailKeys })
	select {
	case <-requests:
		t.Error("reload reposted reply")
	default:
	}
}

type replyLoadResult struct {
	content DetailContent
	err     error
}

func TestReplyLoadingInvalidatesPickerAndStaleMetadata(t *testing.T) {
	requests := make(chan chan replyLoadResult, 4)
	view := replyTestView(t, DetailContent{}, nil)
	view.Load = func() (DetailContent, error) {
		result := make(chan replyLoadResult, 1)
		requests <- result
		loaded := <-result
		return loaded.content, loaded.err
	}
	h := newDetailHarness(t, detailOption(view))
	h.typeText("v")
	first := await(t, requests)
	h.typeText("r")
	h.ui(func() {
		if h.table.detail.picker != nil || !strings.Contains(h.table.detail.status.GetText(false), "loading") {
			t.Error("loading snapshot allowed selection or hid the reason")
		}
	})
	first <- replyLoadResult{content: replyTestContent()}
	h.wait("first reply snapshot", func() bool { return len(h.table.detail.replyTargets) == 2 })
	h.typeText("r")
	var oldPicker *tview.List
	h.ui(func() { oldPicker = h.table.detail.picker })
	h.key(tcell.KeyF5, tcell.ModNone)
	slow := await(t, requests)
	h.ui(func() {
		if h.table.detail.picker != nil || len(h.table.detail.replyTargets) != 0 || !h.table.detail.text.HasFocus() {
			t.Error("reload did not invalidate the picker and its targets")
		}
		oldPicker.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) { h.table.screen.SetFocus(p) })
		if h.table.detail.composer != nil {
			t.Error("superseded picker selected a stale target")
		}
	})
	h.key(tcell.KeyCtrlR, tcell.ModNone)
	latest := await(t, requests)
	latest <- replyLoadResult{content: DetailContent{Text: "new snapshot", ReplyTargets: []ReplyTarget{{ID: "new-root"}}}}
	h.wait("new reply metadata", func() bool { return h.table.detail.text.GetText(false) == "new snapshot" })
	for len(h.frames) > 0 {
		<-h.frames
	}
	slow <- replyLoadResult{content: DetailContent{Text: "stale", ReplyUnavailable: "stale disabled reason"}}
	await(t, h.frames)
	h.ui(func() {
		d := h.table.detail
		if d.text.GetText(false) != "new snapshot" || d.replyUnavailable != "" || len(d.replyTargets) != 1 || d.replyTargets[0].ID != "new-root" {
			t.Error("stale completion overwrote text or reply metadata")
		}
	})
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.ui(func() {
		if h.table.detail.composer.replyID != "new-root" {
			t.Error("picker did not use the latest atomic snapshot")
		}
	})
}

func TestReplyComposerExpiresWhenSnapshotReloads(t *testing.T) {
	requests := make(chan replyRequest, 1)
	h := newDetailHarness(t, detailOption(replyTestView(t, replyTestContent(), replyCallback(requests))))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.typeText("preserve this draft")
	h.ui(func() { h.table.detail.load() })
	h.wait("reloaded under composer", func() bool { return len(h.table.detail.replyTargets) == 2 })
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	h.ui(func() {
		c := h.table.detail.composer
		if !c.post.IsDisabled() || c.draft.GetText() != "preserve this draft" || c.status.GetText(false) != replySelectionExpired {
			t.Error("expired reply selection was not locked with its draft preserved")
		}
	})
	select {
	case <-requests:
		t.Error("expired target was posted")
	default:
	}
}

func TestReplySnapshotCopiesTargetMetadata(t *testing.T) {
	content := replyTestContent()
	h := newDetailHarness(t, detailOption(replyTestView(t, content, nil)))
	h.open()
	h.ui(func() {
		content.ReplyTargets[0].ID = "mutated-source"
		content.ReplyTargets[0].Author = "mutated author"
	})
	h.typeText("r")
	h.ui(func() {
		main, _ := h.table.detail.picker.GetItemText(0)
		if !strings.HasPrefix(main, "root-1 · Alice") {
			t.Error("loaded target metadata still aliases the loader's mutable slice")
		}
	})
}

func TestReplyDoesNotChangeTopLevelCommentAction(t *testing.T) {
	requests := make(chan commentRequest, 1)
	view := replyTestView(t, replyTestContent(), func(string, string) error {
		t.Error("c called the reply callback")
		return nil
	})
	view.SubmitComment = commentCallback(requests)
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("r")
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.typeText("c")
	h.typeText("top level")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	request := await(t, requests)
	if request.body != "top level" {
		t.Errorf("top-level body = %q", request.body)
	}
	request.result <- nil
	h.wait("comment still posted", func() bool {
		return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Comment posted. "+detailKeys
	})
}
