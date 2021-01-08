package view

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/pkg/browser"

	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	wordWrap = 120
	helpText = `USAGE
	-----
	
	The layout contains 2 sections, viz: Sidebar and Contents screen.  
	
	You can use up and down arrow keys or 'j' and 'k' letters to navigate through the sidebar.
	Press 'w' to toggle focus between the sidebar and the contents screen. 
	
	On contents screen:
	  - Use arrow keys or 'j', 'k', 'h', and 'l' letters to navigate through the issue list.
	  - Use 'g' and 'SHIFT+G' to quickly navigate to the top and bottom respectively.
	  - Press 'v' to view selected issue details.
	  - Hit ENTER to open the selected issue in a browser.
	
	Press 'q' / ESC / CTRL+C to quit.`
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

// GetPager returns configured pager.
func GetPager() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	pager := os.Getenv("PAGER")
	if pager == "" && cmdExists("less") {
		pager = "less -r"
	}
	return pager
}

// PagerOut outputs to configured pager if possible.
func PagerOut(out string) error {
	pager := GetPager()
	if pager == "" {
		_, err := fmt.Print(out)
		return err
	}
	pa := strings.Split(pager, " ")
	cmd := exec.Command(pa[0], pa[1:]...)
	cmd.Stdin = strings.NewReader(out)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// MDRenderer constructs markdown renderer.
func MDRenderer() (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
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

func navigate(server string) tui.SelectedFunc {
	return func(r, c int, d interface{}) {
		var path string

		switch data := d.(type) {
		case tui.TableData:
			path = data[r][1]
		case tui.PreviewData:
			path = data.Key
		}

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

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
