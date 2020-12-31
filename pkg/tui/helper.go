package tui

import (
	"bufio"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/ankitpokhrel/jira-cli/pkg/tui/primitive"
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

func getInfoModal() *primitive.Modal {
	return primitive.NewModal().
		SetText("\n\nProcessing. Please wait...").
		SetBackgroundColor(tcell.ColorSpecial).
		SetTextColor(tcell.ColorDefault).
		SetBorderColor(tcell.ColorDefault)
}
