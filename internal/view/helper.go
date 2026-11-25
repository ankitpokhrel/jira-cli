// Package view adapts Jira responses for terminal, plain-text, and CSV output.
package view

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/fatih/color"
	"github.com/mgutz/ansi"
	"github.com/muesli/termenv"
	"github.com/rivo/tview"
	"golang.org/x/term"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
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

	issueTableHelpText = tableHelpText + `

IN ISSUE DETAILS
----------------
* [yellow]/[default] to enter another ticket key (or a number in the current project)
* [yellow]c[default] to write a top-level multiline comment
* [yellow]r[default] to choose a displayed native thread and reply
* [yellow]CTRL + r / F5[default] to reload the ticket
* [yellow]q / ESC[default] to return to the list

Opt in with --comment-threads (experimental Jira Cloud, API-token Basic auth).
--comments N then shows N complete root threads with all replies; 0 hides comments.
Roots are newest-first, replies oldest-first. c creates top-level comments; r replies to a chosen thread.
Replies initially support unrestricted Software/Business threads only.

IN THE COMMENT EDITOR
---------------------
* [yellow]ENTER[default] to insert a newline
* [yellow]CTRL + s[default] or the Post button to submit
* [yellow]SHIFT + ENTER[default] to submit if supported by your terminal
* [yellow]TAB[default] to insert a tab character in the comment
* [yellow]SHIFT + TAB[default] to focus Post (press ENTER to submit)
* [yellow]ESC / CTRL + c[default] to cancel (confirms before discarding a draft)

If SHIFT + ENTER is indistinguishable from ENTER, use CTRL + s or Post.
Optional terminal mappings are documented in the README.`
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

// mdRendererFactory gives overlapping detail loads independent stateful renderers.
type mdRendererFactory func() (*glamour.TermRenderer, error)

// newMDRendererFactory freezes terminal-dependent style selection before tcell
// takes stdin, so background loads cannot consume navigation or composer input.
func newMDRendererFactory() mdRendererFactory {
	style := os.Getenv("GLAMOUR_STYLE")
	if style == "" || style == styles.AutoStyle {
		style = styles.NoTTYStyle
		if term.IsTerminal(int(os.Stdout.Fd())) {
			style = styles.LightStyle
			if termenv.HasDarkBackground() {
				style = styles.DarkStyle
			}
		}
	}
	// Resolve auto before tcell owns stdin; loads must never query the terminal.
	return func() (*glamour.TermRenderer, error) {
		return glamour.NewTermRenderer(
			glamour.WithStylePath(style),
			glamour.WithWordWrap(wordWrap),
		)
	}
}

// formatDateTime preserves the source on invalid dates or timezones; an empty
// timezone keeps the parsed offset instead of assuming the machine's local zone.
func formatDateTime(dt, format, tz string) string {
	t, err := time.Parse(format, dt)
	if err != nil {
		return dt
	}
	if tz == "" {
		return t.Format("2006-01-02 15:04:05")
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return dt
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

// prepareTitle escapes trimmed Jira text so bracketed content cannot become tview styling.
func prepareTitle(text string) string {
	text = strings.TrimSpace(text)
	return tview.Escape(text)
}

// issueKeyFromTuiData supports both table and preview callbacks; unknown payloads
// return an empty key rather than assuming a table layout.
func issueKeyFromTuiData(r int, d any) string {
	var path string

	switch data := d.(type) {
	case tui.TableData:
		path = data.Get(r, data.GetIndex(fieldKey))
	case tui.PreviewData:
		path = data.Key
	}

	return path
}

// jiraURLFromTuiData builds a browser-facing URL rather than a REST endpoint,
// sharing key extraction between table and preview actions.
func jiraURLFromTuiData(server string, r int, d any) string {
	return cmdutil.GenerateServerBrowseURL(server, issueKeyFromTuiData(r, d))
}

// navigate captures the server but resolves the selected issue at invocation;
// browser failures are best-effort because the selection callback has no error channel.
func navigate(server string) tui.SelectedFunc {
	return func(r, _ int, d any) {
		_ = browser.Browse(jiraURLFromTuiData(server, r, d))
	}
}

// copyURL shares browser URL generation with navigation; clipboard failures do
// not interrupt the TUI because the callback cannot return an error.
func copyURL(server string) tui.CopyFunc {
	return func(r, _ int, d any) {
		_ = clipboard.WriteAll(jiraURLFromTuiData(server, r, d))
	}
}

// copyKey copies the selected issue's identity without its server URL; clipboard
// errors are best-effort, as with the URL-copy callback.
func copyKey() tui.CopyKeyFunc {
	return func(r, _ int, d any) {
		_ = clipboard.WriteAll(issueKeyFromTuiData(r, d))
	}
}

// renderPlain removes tview escapes without quoting delimiter-bearing cells.
// Only a tabwriter flush error is returned; individual write errors are not propagated.
func renderPlain(w io.Writer, data tui.TableData, delimiter string) error {
	for _, items := range data {
		n := len(items)
		for j, v := range items {
			_, _ = fmt.Fprintf(w, "%s", unescape(v))
			if j != n-1 {
				_, _ = fmt.Fprintf(w, "%s", delimiter)
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	if _, ok := w.(*tabwriter.Writer); ok {
		return w.(*tabwriter.Writer).Flush()
	}
	return nil
}

// renderCSV delegates quoting to encoding/csv and checks buffered errors after
// flushing, so a failed final write is not reported as successful export.
func renderCSV(w io.Writer, data tui.TableData) error {
	csvwrt := csv.NewWriter(w)

	for _, items := range data {
		if err := csvwrt.Write(items); err != nil {
			return err
		}
	}

	csvwrt.Flush()
	if err := csvwrt.Error(); err != nil {
		return err
	}
	return nil
}

// unescape reverses tview's bracket escapes for plain output without stripping
// the literal bracketed text originally supplied by Jira.
func unescape(s string) string {
	pattern := regexp.MustCompile(`(\[[a-zA-Z0-9_,;: \-\."#]+\[*)\[\]`)
	return pattern.ReplaceAllString(s, "$1]")
}

func coloredOut(msg string, clr color.Attribute, attrs ...color.Attribute) string {
	c := color.New(clr).Add(attrs...)
	return c.Sprint(msg)
}

func xterm256() bool {
	term := os.Getenv("TERM")
	return strings.Contains(term, "-256color")
}

// gray falls back to bright black when TERM does not advertise a 256-color palette.
func gray(msg string) string {
	if xterm256() {
		return gray256(msg)
	}
	return ansi.ColorFunc("black+h")(msg)
}

func gray256(msg string) string {
	return fmt.Sprintf("\x1b[38;5;242m%s\x1b[m", msg)
}

// shortenAndPad uses byte-based widths for legacy table alignment; truncation
// reserves one byte position for the ellipsis, not a Unicode display-cell width.
func shortenAndPad(msg string, limit int) string {
	if limit > 1 && len(msg) > limit {
		return msg[0:limit-1] + "…"
	}
	return pad(msg, limit)
}

// pad applies a minimum byte length without truncation or interpreting ANSI codes;
// callers needing terminal-cell widths must account for those separately.
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
