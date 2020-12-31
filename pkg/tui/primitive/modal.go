package primitive

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Modal is a centered message window used to inform the user.
//
// This primitive is based on https://github.com/rivo/tview/wiki/Modal.
type Modal struct {
	*tview.Box

	// The frame embedded in the modal.
	frame *tview.Frame
	// The form embedded in the modal's frame.
	form *tview.Form
	// The message text (original, not word-wrapped).
	text string
	// The text color.
	textColor tcell.Color
}

// NewModal returns a new modal message window.
func NewModal() *Modal {
	m := Modal{
		Box:       tview.NewBox(),
		form:      tview.NewForm(),
		textColor: tview.Styles.PrimaryTextColor,
	}
	m.form.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(0, 0, 0, 0)

	m.frame = tview.NewFrame(m.form).SetBorders(0, 0, 1, 0, 0, 0)
	m.frame.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)

	return &m
}

// SetBackgroundColor sets the color of the modal frame background.
func (m *Modal) SetBackgroundColor(color tcell.Color) *Modal {
	m.form.SetBackgroundColor(color)
	m.frame.SetBackgroundColor(color)
	return m
}

// SetTextColor sets the color of the message text.
func (m *Modal) SetTextColor(color tcell.Color) *Modal {
	m.textColor = color
	return m
}

// SetBorder sets the flag indicating whether or not the frame should have a border.
func (m *Modal) SetBorder(show bool) *Modal {
	m.frame.SetBorder(show)
	return m
}

// SetBorderColor sets the frame's border color.
func (m *Modal) SetBorderColor(color tcell.Color) *Modal {
	m.frame.SetBorderColor(color)
	return m
}

// SetText sets the message text of the window. The text may contain line
// breaks. Note that words are wrapped, too, based on the final size of the
// window.
func (m *Modal) SetText(text string) *Modal {
	m.text = text
	return m
}

// Focus is called when this primitive receives focus.
func (m *Modal) Focus(delegate func(p tview.Primitive)) {
	delegate(m.form)
}

// HasFocus returns whether or not this primitive has focus.
func (m *Modal) HasFocus() bool {
	return m.form.HasFocus()
}

// Draw draws this primitive onto the screen.
//nolint:gomnd
func (m *Modal) Draw(screen tcell.Screen) {
	// Calculate the width of this modal.
	screenWidth, screenHeight := screen.Size()
	width := screenWidth / 3

	// Reset the text and find out how wide it is.
	m.frame.Clear()
	lines := tview.WordWrap(m.text, width)
	for _, line := range lines {
		m.frame.AddText(line, true, tview.AlignCenter, m.textColor)
	}

	// Set the modal's position and size.
	height := len(lines) + 6
	width += 4
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	m.SetRect(x, y, width, height)

	// Draw the frame.
	m.frame.SetRect(x, y, width, height)
	m.frame.Draw(screen)
}
