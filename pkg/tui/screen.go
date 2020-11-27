package tui

import "github.com/rivo/tview"

// Screen is a terminal screen.
type Screen struct {
	*tview.Application
}

// NewScreen creates a new screen.
func NewScreen() *Screen {
	app := tview.NewApplication()

	return &Screen{app}
}
