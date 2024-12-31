package primitive

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ActionModal is a centered message window used to inform the user or prompt them
// for an immediate decision. It needs to have at least one button (added via
// AddButtons()) or it will never disappear.
//
// This primitive is based on tview.Modal.
type ActionModal struct {
	*tview.Box

	// The frame embedded in the modal.
	frame *tview.Frame

	// The form embedded in the modal's frame.
	form *tview.Form

	// The footer embedded in the modal's frame.
	footer *tview.TextView

	// The message text (original, not word-wrapped).
	text string

	// The text color.
	textColor tcell.Color

	// The optional callback for when the user clicked one of the buttons. It
	// receives the index of the clicked button and the button's label.
	done func(buttonIndex int, buttonLabel string)
}

// NewActionModal returns a new modal message window.
func NewActionModal() *ActionModal {
	m := &ActionModal{
		Box:       tview.NewBox(),
		textColor: tview.Styles.PrimaryTextColor,
	}
	m.form = tview.NewForm().
		SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetButtonTextColor(tview.Styles.PrimaryTextColor)
	m.form.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	m.form.SetCancelFunc(func() {
		if m.done != nil {
			m.done(-1, "")
		}
	})
	m.footer = tview.NewTextView()
	m.footer.SetTitleAlign(tview.AlignCenter)
	m.footer.SetTextAlign(tview.AlignCenter).SetWordWrap(true)
	m.footer.SetTextStyle(tcell.StyleDefault.Italic(true))
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(m.form, 0, 3, true).
		AddItem(m.footer, 2, 1, false)
	m.frame = tview.NewFrame(flex).SetBorders(0, 0, 1, 0, 0, 0)
	m.frame.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)
	return m
}

// GetFooter returns the footer of the modal.
func (m *ActionModal) GetFooter() *tview.TextView {
	return m.footer
}

// SetBackgroundColor sets the color of the modal frame background.
func (m *ActionModal) SetBackgroundColor(color tcell.Color) *ActionModal {
	m.form.SetBackgroundColor(color)
	m.frame.SetBackgroundColor(color)
	return m
}

// SetTextColor sets the color of the message text.
func (m *ActionModal) SetTextColor(color tcell.Color) *ActionModal {
	m.textColor = color
	return m
}

// SetButtonBackgroundColor sets the background color of the buttons.
func (m *ActionModal) SetButtonBackgroundColor(color tcell.Color) *ActionModal {
	m.form.SetButtonBackgroundColor(color)
	return m
}

// SetButtonTextColor sets the color of the button texts.
func (m *ActionModal) SetButtonTextColor(color tcell.Color) *ActionModal {
	m.form.SetButtonTextColor(color)
	return m
}

// SetDoneFunc sets a handler which is called when one of the buttons was
// pressed. It receives the index of the button as well as its label text. The
// handler is also called when the user presses the Escape key. The index will
// then be negative and the label text an emptry string.
func (m *ActionModal) SetDoneFunc(handler func(buttonIndex int, buttonLabel string)) *ActionModal {
	m.done = handler
	return m
}

// SetText sets the message text of the window. The text may contain line
// breaks but color tag states will not transfer to following lines. Note that
// words are wrapped, too, based on the final size of the window.
func (m *ActionModal) SetText(text string) *ActionModal {
	m.text = text
	return m
}

// AddButtons adds buttons to the window. There must be at least one button and
// a "done" handler so the window can be closed again.
func (m *ActionModal) AddButtons(labels []string) *ActionModal {
	for index, label := range labels {
		func(i int, l string) {
			m.form.AddButton(label, func() {
				if m.done != nil {
					m.done(i, l)
				}
			})
			button := m.form.GetButton(m.form.GetButtonCount() - 1)
			button.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyDown, tcell.KeyRight:
					return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
				case tcell.KeyUp, tcell.KeyLeft:
					return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
				}
				return event
			})
		}(index, label)
	}
	return m
}

// ClearButtons removes all buttons from the window.
func (m *ActionModal) ClearButtons() *ActionModal {
	m.form.ClearButtons()
	return m
}

// SetFocus shifts the focus to the button with the given index.
func (m *ActionModal) SetFocus(index int) *ActionModal {
	m.form.SetFocus(index)
	return m
}

// Focus is called when this primitive receives focus.
func (m *ActionModal) Focus(delegate func(p tview.Primitive)) {
	delegate(m.form)
}

// HasFocus returns whether or not this primitive has focus.
func (m *ActionModal) HasFocus() bool {
	return m.form.HasFocus()
}

// Draw draws this primitive onto the screen.
//
//nolint:mnd
func (m *ActionModal) Draw(screen tcell.Screen) {
	// Calculate the width of this modal.
	buttonsWidth := 0
	for i := 0; i < m.form.GetButtonCount(); i++ {
		button := m.form.GetButton(i)
		buttonsWidth += tview.TaggedStringWidth(button.GetLabel()) + 4 + 2
	}
	buttonsWidth -= 2
	screenWidth, screenHeight := screen.Size()
	width := screenWidth / 3
	if width < buttonsWidth {
		width = buttonsWidth
	}
	// width is now without the box border.

	// Reset the text and find out how wide it is.
	m.frame.Clear()
	var lines []string
	for _, line := range strings.Split(m.text, "\n") {
		if len(line) == 0 {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, tview.WordWrap(line, width)...)
	}

	for _, line := range lines {
		m.frame.AddText(line, true, tview.AlignCenter, m.textColor)
	}

	// Set the modal's position and size.
	height := len(lines) + 9
	width += 4
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	m.SetRect(x, y, width, height)

	// Draw the frame.
	m.frame.SetRect(x, y, width, height)
	m.frame.Draw(screen)
}

// MouseHandler returns the mouse handler for this primitive.
func (m *ActionModal) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return m.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		// Pass mouse events on to the form.
		consumed, capture = m.form.MouseHandler()(action, event, setFocus)
		if !consumed && action == tview.MouseLeftDown && m.InRect(event.Position()) {
			setFocus(m)
			consumed = true
		}
		return
	})
}

// InputHandler returns the handler for this primitive.
func (m *ActionModal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return m.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if m.frame.HasFocus() {
			if handler := m.frame.InputHandler(); handler != nil {
				handler(event, setFocus)
				return
			}
		}
	})
}
