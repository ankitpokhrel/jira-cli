package view

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pkg/browser"

	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const helpText = `Use up and down arrow keys or 'j' and 'k' letters to navigate through the list.

	Press 'w' to toggle focus between the sidebar and the contents screen. On contents screen,
	you can use arrow keys or 'j', 'k', 'h', and 'l' letters to navigate through the epic issue list.

	Press ENTER to open the selected issue in the browser.

	Press 'q' / ESC / CTRL+C to quit.`

func formatDateTime(dt, format string) string {
	t, err := time.Parse(format, dt)
	if err != nil {
		return dt
	}

	return t.Format("2006-01-02 15:04:05")
}

func formatDateTimeHuman(dt, format string) string {
	t, err := time.Parse(format, dt)
	if err != nil {
		return dt
	}

	return t.Format("Mon, 02 Jan 06")
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

func navigate(server string) tui.SelectedFunc {
	return func(r, c int, path string) {
		if path == "" {
			return
		}

		url := fmt.Sprintf("%s/browse/%s", server, path)

		_ = browser.OpenURL(url)
	}
}

func renderPlain(w io.Writer, data tui.TableData) error {
	for _, items := range data {
		n := len(items)

		for j, v := range items {
			_, _ = fmt.Fprintf(w, "%s", v)
			if j != n-1 {
				_, _ = fmt.Fprintf(w, "\t")
			}
		}

		_, _ = fmt.Fprintln(w)
	}

	if _, ok := w.(*tabwriter.Writer); ok {
		return w.(*tabwriter.Writer).Flush()
	}

	return nil
}
