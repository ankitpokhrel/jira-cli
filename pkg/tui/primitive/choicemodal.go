package primitive

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChoiceModal struct {
	*tview.Box
	frame  *tview.Frame
	text   string
	list   *tview.List
	footer *tview.TextView
	done   func(index int, label string)
}

func NewChoiceModal() *ChoiceModal {
	m := &ChoiceModal{Box: tview.NewBox()}

	m.list = tview.NewList().
		ShowSecondaryText(false).
		SetMainTextColor(tcell.ColorDefault)

	m.footer = tview.NewTextView()
	m.footer.SetTitleAlign(tview.AlignCenter)
	m.footer.SetTextAlign(tview.AlignCenter)
	m.footer.SetTextStyle(tcell.StyleDefault.Italic(true))
	m.footer.SetBorderPadding(1, 0, 0, 0)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(m.list, 0, 1, true).
		AddItem(m.footer, 2, 0, false)

	m.frame = tview.NewFrame(flex).SetBorders(0, 0, 1, 0, 0, 0)
	m.frame.SetBorder(true).SetBorderPadding(1, 1, 1, 1)

	return m
}

func (m *ChoiceModal) SetText(text string) {
	m.text = text
}

func (m *ChoiceModal) SetDoneFunc(doneFunc func(index int, label string)) *ChoiceModal {
	m.done = doneFunc
	return m
}

func (m *ChoiceModal) SetChoices(choices []string) *ChoiceModal {
	m.list.Clear()
	for _, choice := range choices {
		m.list.AddItem(choice, "", 0, nil)
	}
	return m
}

func (m *ChoiceModal) SetSelected(index int) *ChoiceModal {
	m.list.SetCurrentItem(index)
	return m
}

func (m *ChoiceModal) GetFooter() *tview.TextView {
	return m.footer
}

func (m *ChoiceModal) Focus(delegate func(p tview.Primitive)) {
	delegate(m.list)
}

func (m *ChoiceModal) HasFocus() bool {
	return m.list.HasFocus()
}

func (m *ChoiceModal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return m.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyEnter:
			if m.done != nil {
				index := m.list.GetCurrentItem()
				label, _ := m.list.GetItemText(index)
				m.done(index, label)
			}
		default:
			if handler := m.frame.InputHandler(); handler != nil {
				handler(event, setFocus)
			}
		}
	})
}

func (m *ChoiceModal) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
	return m.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if handler := m.frame.MouseHandler(); handler != nil {
			return handler(action, event, setFocus)
		}
		return false, nil
	})
}

const (
	verticalMargin   = 3
	frameExtraHeight = 7
)

func (m *ChoiceModal) Draw(screen tcell.Screen) {
	screenWidth, screenHeight := screen.Size()
	width := 70

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
		m.frame.AddText(line, true, tview.AlignCenter, tcell.ColorDefault)
	}

	height := len(lines) + m.list.GetItemCount() + frameExtraHeight
	maxHeight := screenHeight - verticalMargin*2
	if height > maxHeight {
		height = maxHeight
	}

	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	m.frame.SetRect(x, y, width, height)
	m.frame.Draw(screen)
}
