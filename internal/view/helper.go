package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/browser"

	"github.com/ankitpokhrel/jira-cli/pkg/tui"
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

func open(server, key string) error {
	if key == "" {
		return nil
	}

	url := fmt.Sprintf("%s/browse/%s", server, key)

	return browser.OpenURL(url)
}

func navigate(server string) tui.SelectedFunc {
	return func(r, c int, path string) {
		_ = open(server, path)
	}
}
