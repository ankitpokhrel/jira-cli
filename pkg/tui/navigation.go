package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	ticketPromptCaption = "Which ticket would you like to view?"
	ticketPromptKeys    = "Enter: view ticket · Esc/CtrlC: cancel"
	ticketInputLimit    = 2048
	ticketUnavailable   = "Ticket navigation unavailable; no request sent."
)

// ticketPrompt owns a single navigation attempt; cancellation and request IDs
// prevent abandoned asynchronous results from replacing the active ticket.
type ticketPrompt struct {
	input     *tview.InputField
	status    *tview.TextView
	pending   bool
	requestID uint64
	cancel    chan struct{}
}

// openTicketPrompt keeps navigation exclusive with draft editing and thread
// selection, preserving the current detail page while another ticket is resolved.
func (d *tableDetail) openTicketPrompt() {
	if d.table.detail != d || d.prompt != nil || d.composer != nil || d.picker != nil {
		return
	}
	p := &ticketPrompt{
		input:  tview.NewInputField(),
		status: tview.NewTextView().SetText(ticketPromptKeys),
		cancel: make(chan struct{}),
	}
	d.prompt = p
	p.input.SetBorder(true).SetTitle(ticketPromptCaption)
	p.input.SetAcceptanceFunc(func(text string, _ rune) bool {
		if strings.ContainsAny(text, "\r\n") {
			p.status.SetText("Enter a single line; input was not changed.\n" + ticketPromptKeys)
			return false
		}
		if len(text) > ticketInputLimit {
			p.status.SetText(fmt.Sprintf("Ticket input exceeds %d bytes; input was not changed.\n%s", ticketInputLimit, ticketPromptKeys))
			return false
		}
		return true
	})
	p.input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc {
			d.closeTicketPrompt()
			return nil
		}
		if p.pending {
			return nil
		}
		if ev.Key() == tcell.KeyEnter {
			d.submitTicketPrompt()
			return nil
		}
		return ev
	})
	if d.view.OpenTicket == nil {
		p.status.SetText(ticketUnavailable + "\n" + ticketPromptKeys)
	}
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(p.input, 3, 0, true).
		AddItem(p.status, 3, 0, false).
		AddItem(nil, 0, 1, false)
	d.table.painter.AddPage("ticket-prompt", page, true, true)
	d.table.screen.SetFocus(p.input)
}

// closeTicketPrompt invalidates pending navigation without closing the details
// or changing the underlying list's selection and scroll position.
func (d *tableDetail) closeTicketPrompt() {
	if d.prompt == nil {
		return
	}
	close(d.prompt.cancel)
	d.prompt = nil
	d.table.painter.RemovePage("ticket-prompt")
	d.table.screen.SetFocus(d.text)
}

// submitTicketPrompt loads a complete candidate before atomically swapping the
// descriptor, title, text, and reply targets on the UI loop. Failed, cancelled,
// or superseded attempts leave the active ticket and its write callbacks intact.
func (d *tableDetail) submitTicketPrompt() {
	p := d.prompt
	if p == nil || p.pending {
		return
	}
	resolve := d.view.OpenTicket
	if resolve == nil {
		p.status.SetText(ticketUnavailable + "\n" + ticketPromptKeys)
		return
	}
	input := p.input.GetText()
	p.requestID++
	requestID, loadID := p.requestID, d.loadID
	p.pending = true
	p.input.SetDisabled(true)
	p.status.SetText("Loading ticket... Esc/CtrlC: cancel.")
	go func() {
		next, err := resolve(input)
		if err == nil && strings.TrimSpace(next.Title) == "" {
			err = errors.New("ticket resolver returned an empty title")
		}
		if err == nil && next.Load == nil {
			err = errors.New("ticket resolver returned no detail loader")
		}
		var content DetailContent
		if err == nil {
			select {
			case <-p.cancel:
				return
			default:
			}
			content, err = next.Load()
		}
		// Prepare the full snapshot before committing any part of the new ticket.
		text := tview.TranslateANSI(tview.Escape(content.Text))
		targets, unavailable := replySnapshot(content)
		d.table.screen.QueueUpdateDraw(func() {
			if d.table.detail != d || d.prompt != p || p.requestID != requestID || d.loadID != loadID {
				return
			}
			if err != nil {
				p.pending = false
				p.input.SetDisabled(false)
				p.status.SetText("Unable to view ticket. Edit and press Enter to retry; Esc/CtrlC: cancel.\n" + err.Error())
				d.table.screen.SetFocus(p.input)
				return
			}
			// An older reload may still be running for the previous descriptor.
			d.loadID++
			d.view = next
			d.text.SetTitle(tview.Escape(next.Title))
			d.text.SetText(text).ScrollToBeginning()
			d.replyTargets, d.replyUnavailable = targets, unavailable
			d.posted = ""
			d.status.SetText(detailKeys)
			d.closeTicketPrompt()
		})
	}()
}
