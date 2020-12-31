package tui

import (
	"bufio"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func pad(in string, n uint) string {
	if in == "" {
		return in
	}

	var (
		i   uint
		out strings.Builder
	)

	for i = 0; i < n; i++ {
		out.WriteString(" ")
	}

	out.WriteString(in)

	for i = 0; i < n; i++ {
		out.WriteString(" ")
	}

	return out.String()
}

func splitText(s string) []string {
	var lines []string

	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	return lines
}

func getInfoModal() *tview.Modal {
	return tview.NewModal().
		SetText("\n\nProcessing. Please wait...").
		SetBackgroundColor(tcell.ColorSpecial)
}
