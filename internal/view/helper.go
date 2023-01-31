package view

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/mgutz/ansi"

	"github.com/ankitpokhrel/jira-cli/pkg/browser"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	wordWrap = 120
	tabWidth = 8
	helpText = `USAGE
	-----
	
	The layout contains 2 sections, viz: Sidebar and Contents screen.  
	
	You can use up and down arrow keys or 'j' and 'k' letters to navigate through the sidebar.
	Press 'w' or Tab to toggle focus between the sidebar and the contents screen.
	
	On contents screen:
	  - Use arrow keys or 'j', 'k', 'h', and 'l' letters to navigate through the issue list.
	  - Use 'g' and 'SHIFT+G' to quickly navigate to the top and bottom respectively.
	  - Press 'v' to view selected issue details.
	  - Press 'c' to copy issue URL to the system clipboard.
	  - Press 'CTRL+K' to copy issue key to the system clipboard.
	  - Hit ENTER to open the selected issue in a browser.
	
	Press 'q' / ESC / CTRL+C to quit.`

	tableHelpText = `[default]ACTIONS AVAILABLE IN THE TUI
----------------------------

* [yellow]← → ↑ ↓ / j, k, h, l[default] to navigate through the list
* [yellow]g[default] to quickly navigate to the top of the list
* [yellow]G[default] to quickly navigate to the bottom of the list
* [yellow]CTRL + f[default] to scroll through a page downwards
* [yellow]CTRL + b[default] to scroll through a page upwards
* [yellow]v[default] to view selected issue details
* [yellow]m[default] to move/transition selected issue
* [yellow]CTRL + r / F5[default] to refresh the issues list
* [yellow]ENTER[default] to open the selected issue in the browser
* [yellow]c[default] to copy issue URL to the system clipboard
* [yellow]CTRL + k[default] to copy issue key to the system clipboard
* [yellow]q / ESC / CTRL + c[default] to quit the app
* [yellow]?[default] to view this help page`
)

// ValidIssueColumns returns valid columns for issue list.
func ValidIssueColumns() []string {
	return []string{
		fieldType,
		fieldKey,
		fieldSummary,
		fieldStatus,
		fieldAssignee,
		fieldReporter,
		fieldPriority,
		fieldResolution,
		fieldCreated,
		fieldUpdated,
		fieldLabels,
	}
}

// ValidSprintColumns returns valid columns for sprint list.
func ValidSprintColumns() []string {
	return []string{
		fieldID,
		fieldName,
		fieldStartDate,
		fieldEndDate,
		fieldCompleteDate,
		fieldState,
	}
}

// MDRenderer constructs markdown renderer.
func MDRenderer() (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(wordWrap),
	)
}

func formatDateTime(dt, format string) string {
	t, err := time.Parse(format, dt)
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

func issueKeyFromTuiData(r int, d interface{}) string {
	var path string

	switch data := d.(type) {
	case tui.TableData:
		path = data.Get(r, data.GetIndex(fieldKey))
	case tui.PreviewData:
		path = data.Key
	}

	return path
}

func jiraURLFromTuiData(server string, r int, d interface{}) string {
	return fmt.Sprintf("%s/browse/%s", server, issueKeyFromTuiData(r, d))
}

func navigate(server string) tui.SelectedFunc {
	return func(r, c int, d interface{}) {
		_ = browser.Browse(jiraURLFromTuiData(server, r, d))
	}
}

func copyURL(server string) tui.CopyFunc {
	return func(r, c int, d interface{}) {
		_ = clipboard.WriteAll(jiraURLFromTuiData(server, r, d))
	}
}

func copyKey() tui.CopyKeyFunc {
	return func(r, c int, d interface{}) {
		_ = clipboard.WriteAll(issueKeyFromTuiData(r, d))
	}
}

func renderPlain(w io.Writer, data tui.TableData) error {
	for _, items := range data {
		n := len(items)
		for j, v := range items {
			fmt.Fprintf(w, "%s", v)
			if j != n-1 {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintln(w)
	}

	if _, ok := w.(*tabwriter.Writer); ok {
		return w.(*tabwriter.Writer).Flush()
	}
	return nil
}

func coloredOut(msg string, clr color.Attribute, attrs ...color.Attribute) string {
	c := color.New(clr).Add(attrs...)
	return c.Sprint(msg)
}

func xterm256() bool {
	term := os.Getenv("TERM")
	return strings.Contains(term, "-256color")
}

func gray(msg string) string {
	if xterm256() {
		return gray256(msg)
	}
	return ansi.ColorFunc("black+h")(msg)
}

func gray256(msg string) string {
	return fmt.Sprintf("\x1b[38;5;242m%s\x1b[m", msg)
}

func shortenAndPad(msg string, limit int) string {
	if limit > 1 && len(msg) > limit {
		return msg[0:limit-1] + "…"
	}
	return pad(msg, limit)
}

func pad(msg string, limit int) string {
	var out strings.Builder
	out.WriteString(msg)
	for i := len(msg); i < limit; i++ {
		out.WriteRune(' ')
	}
	return out.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
