package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Screen is a shell screen.
type Screen struct {
	*tview.Application
}

// NewScreen creates a new screen.
func NewScreen() *Screen {
	app := tview.NewApplication()

	app.SetBeforeDrawFunc(func(s tcell.Screen) bool {
		s.Clear()
		return false
	})

	return &Screen{app}
}

// Paint paints UI to the screen.
func (s *Screen) Paint(root tview.Primitive) error {
	return s.SetRoot(root, true).SetFocus(root).Run()
}
