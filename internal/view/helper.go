package view

import (
	"strings"
	"time"
)

func formatDateTime(dt string) string {
	const rfc3339 = "2006-01-02T15:04:05-0700"

	t, err := time.Parse(rfc3339, dt)
	if err != nil {
		return dt
	}

	return t.Format("2006-01-02 15:04:05")
}

func prepareTitle(text string) string {
	text = strings.TrimSpace(text)

	// Single word within big brackets like [BE] is treated as a
	// tag and is not parsed by tview creating a gap in the text.
	//
	// We will handle this with a little trick by replacing
	// big brackets with similar-looking unicode characters.
	text = strings.ReplaceAll(text, "[", "⦗")
	text = strings.ReplaceAll(text, "]", "⦘")

	return text
}
