package tui

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ticketAction struct {
	kind, ticket, target, body string
}

// navigationContent makes each ticket's text and reply root distinguishable so mixed snapshots cannot pass navigation checks.
func navigationContent(ticket string) DetailContent {
	return DetailContent{
		Text:         "Ticket " + ticket + "\n" + strings.Repeat("details\n", 40) + "bottom " + ticket,
		ReplyTargets: []ReplyTarget{{ID: ticket + "-root", Author: ticket + " author", Preview: ticket + " thread"}},
	}
}

// navigationView captures ticket identity in every callback so tests detect partial descriptor rebinding.
func navigationView(ticket string, actions chan<- ticketAction) DetailView {
	return DetailView{
		Title: ticket,
		Load:  func() (DetailContent, error) { return navigationContent(ticket), nil },
		SubmitComment: func(body string) error {
			actions <- ticketAction{kind: "comment", ticket: ticket, body: body}
			return nil
		},
		SubmitReply: func(rootID, body string) error {
			actions <- ticketAction{kind: "reply", ticket: ticket, target: rootID, body: body}
			return nil
		},
		OpenTicket: func(input string) (DetailView, error) {
			actions <- ticketAction{kind: "open", ticket: ticket, target: input}
			return navigationView(input, actions), nil
		},
	}
}

// assertTicket requires descriptor, visible content, reply metadata, and posted status to belong to one atomic ticket snapshot.
func assertTicket(t *testing.T, h *detailHarness, ticket, posted string) {
	t.Helper()
	h.ui(func() {
		d := h.table.detail
		if d == nil {
			t.Error("detail was closed")
			return
		}
		if d.view.Title != ticket || d.text.GetTitle() != tview.Escape(ticket) || d.text.GetText(false) != tview.TranslateANSI(tview.Escape(navigationContent(ticket).Text)) {
			t.Errorf("descriptor/title/text were not kept together for %q", ticket)
		}
		if len(d.replyTargets) != 1 || d.replyTargets[0].ID != ticket+"-root" || d.replyUnavailable != "" {
			t.Errorf("reply metadata was not kept with ticket %q", ticket)
		}
		if d.posted != posted {
			t.Errorf("posted banner = %q, want %q", d.posted, posted)
		}
		status := detailKeys
		if posted != "" {
			status = posted + " posted. " + status
		}
		if d.status.GetText(false) != status {
			t.Error("ticket navigation changed or retained the wrong status banner")
		}
	})
}

// expectTicketAction checks the captured ticket and target as well as action kind to catch stale callback routing.
func expectTicketAction(t *testing.T, actions <-chan ticketAction, want ticketAction) {
	t.Helper()
	if got := await(t, actions); got != want {
		t.Errorf("callback = %#v, want %#v", got, want)
	}
}

// navigateTo waits for a committed descriptor rather than treating resolver completion as a successful ticket switch.
func navigateTo(h *detailHarness, input, title string) {
	h.t.Helper()
	h.typeText("/")
	h.typeText(input)
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.wait("ticket navigation", func() bool { return h.table.detail.prompt == nil && h.table.detail.view.Title == title })
}

type ticketContext struct {
	row, col, offsetRow, offsetCol, textRow, textCol int
}

// currentTicketContext captures list and detail offsets on the UI goroutine for cancellation and failure comparisons.
func currentTicketContext(h *detailHarness) ticketContext {
	var context ticketContext
	h.ui(func() {
		context.row, context.col = h.table.view.GetSelection()
		context.offsetRow, context.offsetCol = h.table.view.GetOffset()
		context.textRow, context.textCol = h.table.detail.text.GetScrollOffset()
	})
	return context
}

// assertPromptClosed rejects orphan prompt pages and requires focus to return to the active detail view.
func assertPromptClosed(t *testing.T, h *detailHarness) {
	t.Helper()
	h.ui(func() {
		if h.table.detail.prompt != nil || h.table.painter.HasPage("ticket-prompt") || !h.table.detail.text.HasFocus() {
			t.Error("navigation prompt did not close and restore detail focus")
		}
	})
}

// drainTicketFrames removes prior draw notifications so the next wait observes a released stale worker's update.
func drainTicketFrames(h *detailHarness) {
	for len(h.frames) > 0 {
		<-h.frames
	}
}

// TestTicketPromptInputAndSinglePendingRequest keeps typing inert, locks duplicate submissions, and commits only after the new content loads.
func TestTicketPromptInputAndSinglePendingRequest(t *testing.T) {
	inputs := make(chan string, 2)
	resolve := make(chan DetailView, 1)
	loadStarted := make(chan struct{})
	loaded := make(chan DetailContent, 1)
	actions := make(chan ticketAction, 10)
	view := navigationView("OLD", actions)
	view.OpenTicket = func(input string) (DetailView, error) {
		inputs <- input
		return <-resolve, nil
	}
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("/")
	var prompt *ticketPrompt
	h.ui(func() {
		prompt = h.table.detail.prompt
		if prompt == nil || prompt.input.GetText() != "" || prompt.input.GetTitle() != ticketPromptCaption || !prompt.input.HasFocus() {
			t.Error("slash did not open a focused empty ticket prompt")
			return
		}
		h.table.detail.openTicketPrompt()
		h.table.detail.openComposer()
		h.table.detail.openReplyPicker()
		if h.table.detail.prompt != prompt || h.table.detail.composer != nil || h.table.detail.picker != nil {
			t.Error("prompt opened twice or allowed a competing child view")
		}
	})
	h.typeText("q/c/r")
	select {
	case <-inputs:
		t.Error("typing called the resolver")
	default:
	}
	h.key(tcell.KeyEnter, tcell.ModNone)
	if got := await(t, inputs); got != "q/c/r" {
		t.Errorf("captured input = %q", got)
	}
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.typeText("ignored")
	h.paste("also ignored")
	h.ui(func() {
		if !prompt.pending || prompt.input.GetText() != "q/c/r" {
			t.Error("pending prompt accepted edits or duplicate Enter")
		}
	})
	next := navigationView("NEW", actions)
	next.Load = func() (DetailContent, error) { close(loadStarted); return <-loaded, nil }
	resolve <- next
	await(t, loadStarted)
	assertTicket(t, h, "OLD", "")
	select {
	case <-inputs:
		t.Error("duplicate navigation request")
	default:
	}
	loaded <- navigationContent("NEW")
	h.wait("new ticket loaded", func() bool { return h.table.detail.prompt == nil })
	assertTicket(t, h, "NEW", "")
	assertPromptClosed(t, h)
}

// TestTicketSwitchRebindsWholeDescriptorAcrossTwoJumps moves all write callbacks and reply metadata together without replacing the original list context.
func TestTicketSwitchRebindsWholeDescriptorAcrossTwoJumps(t *testing.T) {
	actions := make(chan ticketAction, 20)
	h := newDetailHarness(t, detailOption(navigationView("OLD", actions)))
	h.ui(func() { h.table.view.Select(35, 0).SetOffset(30, 1) })
	h.open()
	h.key(tcell.KeyEnd, tcell.ModNone)
	before := currentTicketContext(h)
	h.ui(func() {
		h.table.detail.posted = "Comment"
		h.table.detail.status.SetText("Comment posted. " + detailKeys)
	})
	navigateTo(h, "NEW[blue]", "NEW[blue]")
	expectTicketAction(t, actions, ticketAction{kind: "open", ticket: "OLD", target: "NEW[blue]"})
	assertTicket(t, h, "NEW[blue]", "")
	h.ui(func() {
		row, col := h.table.detail.text.GetScrollOffset()
		if row != 0 || col != 0 || h.table.detail.status.GetText(false) != detailKeys {
			t.Error("new ticket did not start at the top with the old posted banner cleared")
		}
	})
	h.typeText("c")
	h.typeText("new comment")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "comment", ticket: "NEW[blue]", body: "new comment"})
	h.wait("new comment posted", func() bool {
		return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Comment posted. "+detailKeys
	})
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.typeText("new reply")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "reply", ticket: "NEW[blue]", target: "NEW[blue]-root", body: "new reply"})
	h.wait("new reply posted", func() bool {
		return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Reply posted. "+detailKeys
	})
	navigateTo(h, "THIRD", "THIRD")
	expectTicketAction(t, actions, ticketAction{kind: "open", ticket: "NEW[blue]", target: "THIRD"})
	assertTicket(t, h, "THIRD", "")
	h.typeText("c")
	h.typeText("third comment")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "comment", ticket: "THIRD", body: "third comment"})
	h.wait("third comment posted", func() bool {
		return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Comment posted. "+detailKeys
	})
	h.typeText("q")
	h.ui(func() {
		row, col := h.table.view.GetSelection()
		or, oc := h.table.view.GetOffset()
		if h.table.detail != nil || !h.table.view.HasFocus() || row != before.row || col != before.col || or != before.offsetRow || oc != before.offsetCol {
			t.Error("jump history replaced the original list context")
		}
		if len(h.table.data) != 61 || h.table.data[35][0] != "TEST-35" {
			t.Error("navigation added or replaced list data")
		}
	})
}

// TestTicketNavigationErrorsKeepCurrentSnapshotAndInput leaves the active ticket, scroll, and editable prompt unchanged after resolution or load failure.
func TestTicketNavigationErrorsKeepCurrentSnapshotAndInput(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		resolve           func(string) (DetailView, error)
	}{
		{name: "nil resolver", input: "NEW", want: ticketUnavailable},
		{name: "blank input", want: "ticket is required", resolve: func(input string) (DetailView, error) {
			if input != "" {
				t.Errorf("blank input was changed: %q", input)
			}
			return DetailView{}, errors.New("ticket is required")
		}},
		{name: "invalid input", input: "q/c/r", want: "[red]invalid\nmore", resolve: func(string) (DetailView, error) { return DetailView{}, errors.New("[red]invalid\nmore") }},
		{name: "empty title", input: "NEW", want: "empty title", resolve: func(string) (DetailView, error) {
			return DetailView{Load: func() (DetailContent, error) { t.Error("invalid descriptor loaded"); return DetailContent{}, nil }}, nil
		}},
		{name: "whitespace title", input: "NEW", want: "empty title", resolve: func(string) (DetailView, error) { return DetailView{Title: " \t "}, nil }},
		{name: "nil loader", input: "NEW", want: "no detail loader", resolve: func(string) (DetailView, error) { return DetailView{Title: "NEW"}, nil }},
		{name: "load or renderer error", input: "NEW", want: "render failed", resolve: func(string) (DetailView, error) {
			return DetailView{Title: "NEW", Load: func() (DetailContent, error) { return navigationContent("NEW"), errors.New("render failed") }}, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actions := make(chan ticketAction, 10)
			view := navigationView("OLD", actions)
			view.OpenTicket = tc.resolve
			h := newDetailHarness(t, detailOption(view))
			h.open()
			h.key(tcell.KeyEnd, tcell.ModNone)
			h.ui(func() {
				h.table.detail.posted = "Reply"
				h.table.detail.status.SetText("Reply posted. " + detailKeys)
			})
			before := currentTicketContext(h)
			h.typeText("/")
			h.typeText(tc.input)
			h.key(tcell.KeyEnter, tcell.ModNone)
			h.wait("prompt error", func() bool {
				p := h.table.detail.prompt
				return p != nil && !p.pending && strings.Contains(p.status.GetText(false), tc.want)
			})
			assertTicket(t, h, "OLD", "Reply")
			if after := currentTicketContext(h); after != before {
				t.Error("failed navigation changed list or detail scroll")
			}
			h.ui(func() {
				p := h.table.detail.prompt
				if p.input.GetText() != tc.input || !p.input.HasFocus() {
					t.Error("error lost input or editable focus")
				}
				if tc.name == "invalid input" && !strings.Contains(h.screenText(), "[red]invalid") {
					t.Error("provider error was not displayed literally")
				}
			})
		})
	}
}

// TestTicketNavigationFailureRetainsAllActionBindings prevents a failed target render from rebinding comments, replies, or later navigation.
func TestTicketNavigationFailureRetainsAllActionBindings(t *testing.T) {
	actions := make(chan ticketAction, 10)
	view := navigationView("OLD", actions)
	view.OpenTicket = func(input string) (DetailView, error) {
		actions <- ticketAction{kind: "open", ticket: "OLD", target: input}
		next := navigationView("WRONG", actions)
		next.Load = func() (DetailContent, error) { return navigationContent("WRONG"), errors.New("render failed") }
		return next, nil
	}
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("/")
	h.typeText("NEXT")
	h.key(tcell.KeyEnter, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "open", ticket: "OLD", target: "NEXT"})
	h.wait("failed target render", func() bool { return strings.Contains(h.table.detail.prompt.status.GetText(false), "render failed") })
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.typeText("c")
	h.typeText("old comment")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "comment", ticket: "OLD", body: "old comment"})
	h.wait("old comment posted", func() bool {
		return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Comment posted. "+detailKeys
	})
	h.typeText("r")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.typeText("old reply")
	h.key(tcell.KeyCtrlS, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "reply", ticket: "OLD", target: "OLD-root", body: "old reply"})
	h.wait("old reply posted", func() bool {
		return h.table.detail.composer == nil && h.table.detail.status.GetText(false) == "Reply posted. "+detailKeys
	})
	h.typeText("/")
	h.typeText("ANOTHER")
	h.key(tcell.KeyEnter, tcell.ModNone)
	expectTicketAction(t, actions, ticketAction{kind: "open", ticket: "OLD", target: "ANOTHER"})
	h.wait("second failed target render", func() bool { return strings.Contains(h.table.detail.prompt.status.GetText(false), "render failed") })
	assertTicket(t, h, "OLD", "Reply")
}

// TestTicketPromptCanEditAndRetryResolverAndRenderErrors preserves editable input through multiple failures until an explicitly retried load succeeds.
func TestTicketPromptCanEditAndRetryResolverAndRenderErrors(t *testing.T) {
	actions := make(chan ticketAction, 10)
	inputs := make(chan string, 3)
	view := navigationView("OLD", actions)
	view.OpenTicket = func(input string) (DetailView, error) {
		inputs <- input
		if input == "BAD" {
			return DetailView{}, errors.New("invalid ticket")
		}
		next := navigationView(input, actions)
		if input == "GOOD" {
			next.Load = func() (DetailContent, error) { return DetailContent{}, errors.New("render failed") }
		}
		return next, nil
	}
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("/")
	h.typeText("BAD")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.wait("validation error", func() bool { return strings.Contains(h.table.detail.prompt.status.GetText(false), "invalid ticket") })
	for i := 0; i < 3; i++ {
		h.key(tcell.KeyBackspace2, tcell.ModNone)
	}
	h.typeText("GOOD")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.wait("render error", func() bool { return strings.Contains(h.table.detail.prompt.status.GetText(false), "render failed") })
	assertTicket(t, h, "OLD", "")
	h.typeText("-RETRY")
	h.key(tcell.KeyEnter, tcell.ModNone)
	h.wait("retry success", func() bool { return h.table.detail.prompt == nil })
	assertTicket(t, h, "GOOD-RETRY", "")
	for _, want := range []string{"BAD", "GOOD", "GOOD-RETRY"} {
		if got := await(t, inputs); got != want {
			t.Errorf("attempt input = %q, want %q", got, want)
		}
	}
}

// TestTicketPromptCancellationKeepsContext makes Escape and CtrlC preserve the current ticket and ignore any late pending result.
func TestTicketPromptCancellationKeepsContext(t *testing.T) {
	for _, pending := range []bool{false, true} {
		for _, key := range []tcell.Key{tcell.KeyEsc, tcell.KeyCtrlC} {
			name := tcell.KeyNames[key]
			if pending {
				name += " pending"
			}
			t.Run(name, func(t *testing.T) {
				actions := make(chan ticketAction, 10)
				started := make(chan struct{})
				loaded := make(chan DetailContent, 1)
				view := navigationView("OLD", actions)
				view.OpenTicket = func(string) (DetailView, error) {
					next := navigationView("LATE", actions)
					next.Load = func() (DetailContent, error) { close(started); return <-loaded, nil }
					return next, nil
				}
				h := newDetailHarness(t, detailOption(view))
				h.ui(func() { h.table.view.Select(35, 0).SetOffset(30, 1) })
				h.open()
				h.key(tcell.KeyEnd, tcell.ModNone)
				before := currentTicketContext(h)
				h.typeText("/")
				h.typeText("NEXT")
				if pending {
					h.key(tcell.KeyEnter, tcell.ModNone)
					await(t, started)
				}
				h.key(key, tcell.ModNone)
				assertPromptClosed(t, h)
				if after := currentTicketContext(h); after != before {
					t.Error("cancel changed ticket context")
				}
				if pending {
					drainTicketFrames(h)
					loaded <- navigationContent("LATE")
					await(t, h.frames)
					assertTicket(t, h, "OLD", "")
					assertPromptClosed(t, h)
				}
			})
		}
	}
}

// TestTicketCancelledRequestCannotOverwriteNextPrompt prevents an older cancelled navigation from overwriting a newer committed ticket.
func TestTicketCancelledRequestCannotOverwriteNextPrompt(t *testing.T) {
	actions := make(chan ticketAction, 10)
	loads := make(chan chan DetailContent, 2)
	view := navigationView("OLD", actions)
	view.OpenTicket = func(input string) (DetailView, error) {
		next := navigationView(input, actions)
		next.Load = func() (DetailContent, error) {
			result := make(chan DetailContent, 1)
			loads <- result
			return <-result, nil
		}
		return next, nil
	}
	h := newDetailHarness(t, detailOption(view))
	h.open()
	h.typeText("/")
	h.typeText("A")
	h.key(tcell.KeyEnter, tcell.ModNone)
	a := await(t, loads)
	h.key(tcell.KeyEsc, tcell.ModNone)
	h.typeText("/")
	h.typeText("B")
	h.key(tcell.KeyEnter, tcell.ModNone)
	b := await(t, loads)
	b <- navigationContent("B")
	h.wait("B committed", func() bool { return h.table.detail.prompt == nil })
	drainTicketFrames(h)
	a <- navigationContent("A")
	await(t, h.frames)
	assertTicket(t, h, "B", "")
	assertPromptClosed(t, h)
}

// TestTicketSwitchAndOldReloadOrdering keeps the new atomic snapshot authoritative regardless of when an old ticket reload completes.
func TestTicketSwitchAndOldReloadOrdering(t *testing.T) {
	for _, oldFirst := range []bool{false, true} {
		name := "old reload finishes last"
		if oldFirst {
			name = "old reload finishes behind prompt"
		}
		t.Run(name, func(t *testing.T) {
			actions := make(chan ticketAction, 10)
			oldStarted, newStarted := make(chan struct{}), make(chan struct{})
			oldResult, newResult := make(chan DetailContent, 1), make(chan DetailContent, 1)
			var calls atomic.Int32
			view := navigationView("OLD", actions)
			view.Load = func() (DetailContent, error) {
				if calls.Add(1) == 1 {
					return navigationContent("OLD"), nil
				}
				close(oldStarted)
				return <-oldResult, nil
			}
			view.OpenTicket = func(string) (DetailView, error) {
				next := navigationView("NEW", actions)
				next.Load = func() (DetailContent, error) { close(newStarted); return <-newResult, nil }
				return next, nil
			}
			h := newDetailHarness(t, detailOption(view))
			h.open()
			h.key(tcell.KeyCtrlR, tcell.ModNone)
			await(t, oldStarted)
			h.typeText("/")
			h.typeText("NEW")
			h.key(tcell.KeyEnter, tcell.ModNone)
			await(t, newStarted)
			if oldFirst {
				oldResult <- navigationContent("OLD-RELOADED")
				h.wait("old reload behind prompt", func() bool { return h.table.detail.text.GetText(false) == navigationContent("OLD-RELOADED").Text })
				h.ui(func() {
					if h.table.detail.view.Title != "OLD" || !h.table.detail.prompt.pending {
						t.Error("old reload prematurely committed new descriptor")
					}
				})
			}
			newResult <- navigationContent("NEW")
			h.wait("new ticket committed", func() bool { return h.table.detail.prompt == nil })
			if !oldFirst {
				drainTicketFrames(h)
				oldResult <- navigationContent("OLD-RELOADED")
				await(t, h.frames)
			}
			assertTicket(t, h, "NEW", "")
		})
	}
}

// TestTicketReloadOrCloseInvalidatesPendingNavigation prevents detached prompt workers from replacing reloaded or reopened detail state.
func TestTicketReloadOrCloseInvalidatesPendingNavigation(t *testing.T) {
	for _, operation := range []string{"reload", "close and reopen"} {
		t.Run(operation, func(t *testing.T) {
			actions := make(chan ticketAction, 10)
			started, result := make(chan struct{}), make(chan DetailContent, 1)
			view := navigationView("OLD", actions)
			view.OpenTicket = func(string) (DetailView, error) {
				next := navigationView("LATE", actions)
				next.Load = func() (DetailContent, error) { close(started); return <-result, nil }
				return next, nil
			}
			h := newDetailHarness(t, detailOption(view))
			h.open()
			h.typeText("/")
			h.typeText("NEXT")
			h.key(tcell.KeyEnter, tcell.ModNone)
			await(t, started)
			h.ui(func() {
				if operation == "reload" {
					h.table.detail.load()
				} else {
					h.table.detail.close()
				}
				if h.table.painter.HasPage("ticket-prompt") {
					t.Error("reload/close left an orphan prompt")
				}
			})
			if operation == "reload" {
				h.wait("old reload completed", func() bool { return len(h.table.detail.replyTargets) == 1 })
			} else {
				h.open()
			}
			drainTicketFrames(h)
			result <- navigationContent("LATE")
			await(t, h.frames)
			assertTicket(t, h, "OLD", "")
			assertPromptClosed(t, h)
		})
	}
}

// TestTicketSlashIsLiteralInCommentAndReplyEditors prevents navigation from stealing editor input or competing with a picker or pending post.
func TestTicketSlashIsLiteralInCommentAndReplyEditors(t *testing.T) {
	for _, mode := range []string{"comment", "reply"} {
		t.Run(mode, func(t *testing.T) {
			var resolutions atomic.Int32
			started, result := make(chan string, 1), make(chan error, 1)
			submit := func(body string) error { started <- body; return <-result }
			view := replyTestView(t, replyTestContent(), func(_, body string) error { return submit(body) })
			view.SubmitComment = submit
			view.OpenTicket = func(string) (DetailView, error) {
				resolutions.Add(1)
				return DetailView{}, errors.New("unexpected navigation")
			}
			h := newDetailHarness(t, detailOption(view))
			h.typeText("/")
			h.ui(func() {
				if h.table.detail != nil {
					t.Error("list slash opened navigation")
				}
			})
			h.open()
			if mode == "reply" {
				h.typeText("r")
				h.typeText("/")
				h.ui(func() {
					h.table.detail.openTicketPrompt()
					if h.table.detail.prompt != nil {
						t.Error("picker allowed navigation")
					}
				})
				h.key(tcell.KeyEnter, tcell.ModNone)
			} else {
				h.typeText("c")
			}
			body := "/paths/q/c/r"
			h.typeText(body)
			h.ui(func() { h.table.detail.openTicketPrompt() })
			h.key(tcell.KeyCtrlS, tcell.ModNone)
			if got := await(t, started); got != body {
				t.Errorf("editor body = %q", got)
			}
			h.typeText("/")
			h.ui(func() {
				h.table.detail.openTicketPrompt()
				if h.table.detail.prompt != nil || !h.table.detail.composer.posting || resolutions.Load() != 0 {
					t.Error("editor or pending post allowed ticket navigation")
				}
			})
			result <- nil
			h.wait("editor submission complete", func() bool { return h.table.detail.composer == nil })
		})
	}
}

// TestTicketPromptRejectsMultilineAndOversizedInputWithoutTruncation preserves existing input and avoids resolution when a paste exceeds prompt constraints.
func TestTicketPromptRejectsMultilineAndOversizedInputWithoutTruncation(t *testing.T) {
	actions := make(chan ticketAction, 10)
	h := newDetailHarness(t, detailOption(navigationView("OLD", actions)))
	h.open()
	h.typeText("/")
	h.typeText("KEEP")
	h.paste("\nOTHER")
	h.ui(func() {
		p := h.table.detail.prompt
		if p.input.GetText() != "KEEP" || !strings.Contains(p.status.GetText(false), "single line") {
			t.Error("multiline paste changed the input")
		}
	})
	h.paste(strings.Repeat("x", ticketInputLimit))
	h.ui(func() {
		p := h.table.detail.prompt
		if p.input.GetText() != "KEEP" || !strings.Contains(p.status.GetText(false), "input was not changed") {
			t.Error("oversized paste was silently truncated")
		}
	})
	select {
	case <-actions:
		t.Error("input validation called a resolver")
	default:
	}
}

// TestTicketPromptCtrlCPreservesPriorCaptureAndNormalExit consumes prompt cancellation but restores normal interrupt delegation afterward.
func TestTicketPromptCtrlCPreservesPriorCaptureAndNormalExit(t *testing.T) {
	var interrupts atomic.Int32
	actions := make(chan ticketAction, 10)
	h := newDetailHarness(t,
		func(table *Table) {
			table.screen.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
				if ev.Key() == tcell.KeyCtrlC {
					interrupts.Add(1)
				}
				return ev
			})
		},
		detailOption(navigationView("OLD", actions)),
	)
	h.open()
	h.typeText("/")
	h.key(tcell.KeyCtrlC, tcell.ModNone)
	assertPromptClosed(t, h)
	if interrupts.Load() != 0 {
		t.Error("prompt CtrlC reached the prior application quit handler")
	}
	h.table.screen.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone))
	await(t, h.done)
	if interrupts.Load() != 1 {
		t.Error("normal CtrlC was not restored outside the prompt")
	}
}
