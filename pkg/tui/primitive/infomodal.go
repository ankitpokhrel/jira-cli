package primitive

import (
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InfoModal is a centered message window used to inform the user or prompt them
// for an immediate decision. It needs to have at least one button (added via
// AddButtons()) or it will never disappear.
//
// This primitive is based on tview.Modal.
type InfoModal struct {
	*tview.Box

	// The body of the modal.
	info *tview.Frame

	// The message text (original, not word-wrapped).
	text string

	// The text color.
	textColor tcell.Color
}

// NewInfoModal returns a new modal message window.
func NewInfoModal() *InfoModal {
	m := &InfoModal{
		Box:       tview.NewBox(),
		textColor: tview.Styles.PrimaryTextColor,
	}
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	m.info = tview.NewFrame(tv)
	return m
}

// SetInfo sets the body of the modal.
func (i *InfoModal) SetInfo(info string) *InfoModal {
	i.text = info
	i.info.GetPrimitive().(*tview.TextView).SetText(info)
	return i
}

// SetTitle sets the title of the modal frame.
func (i *InfoModal) SetTitle(title string) *InfoModal {
	i.info.SetTitle(" " + title + " ")
	return i
}

// SetAlign sets the alignment of the text.
func (i *InfoModal) SetAlign(align int) *InfoModal {
	i.info.GetPrimitive().(*tview.TextView).SetTextAlign(align)
	return i
}

// Draw draws this primitive onto the screen.
//
//nolint:mnd
func (i *InfoModal) Draw(screen tcell.Screen) {
	screenWidth, screenHeight := screen.Size()
	width := screenWidth / 3

	// Reset the text and find out how wide it is.
	i.info.Clear()
	lines := strings.Split(i.text, "\n")
	for _, line := range lines {
		w := utf8.RuneCountInString(line)
		if w > width {
			width = w
		}
	}

	// Set the modal's position and size.
	height := len(lines) + 4
	width += 4
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	i.info.SetRect(x, y, width, height)

	i.info.SetBorder(true).SetTitleAlign(tview.AlignCenter)

	i.info.Draw(screen)
}
