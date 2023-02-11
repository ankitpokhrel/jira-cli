package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cli/safeexec"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-isatty"
	"github.com/rivo/tview"

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

func getInfoModal() *tview.Modal {
	return tview.NewModal().
		SetText("\n\nProcessing. Please wait...").
		SetBackgroundColor(tcell.ColorSpecial).
		SetTextColor(tcell.ColorDefault)
}

func getActionModal() *primitive.ActionModal {
	return primitive.NewActionModal().
		SetBackgroundColor(tcell.ColorSpecial).
		SetButtonBackgroundColor(tcell.ColorDarkCyan).
		SetTextColor(tcell.ColorDefault)
}

// IsDumbTerminal checks TERM environment variable and returns true if it is set to dumb.
//
// Dumb terminal indicates terminal with limited capability. It may not provide support
// for special character sequences, e.g., no handling of ANSI escape sequences.
func IsDumbTerminal() bool {
	term := strings.ToLower(os.Getenv("TERM"))
	return term == "" || term == "dumb"
}

// IsNotTTY returns true if the stdout file descriptor is not a TTY.
func IsNotTTY() bool {
	return !isatty.IsTerminal(os.Stdout.Fd())
}

// GetPager returns configured pager.
func GetPager() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	if IsDumbTerminal() {
		return "cat"
	}
	pager := os.Getenv("JIRA_PAGER")
	if pager == "" {
		pgr := os.Getenv("PAGER")
		if pgr == "" {
			pager = "less"
		} else {
			pager = pgr
		}
	}
	return pager
}

// PagerOut outputs to configured pager if possible.
func PagerOut(out string) error {
	pagerCmd := GetPager()
	if pagerCmd == "" {
		_, err := fmt.Print(out)
		return err
	}

	pa := strings.Split(pagerCmd, " ")
	pager, pagerArgs := pa[0], pa[1:]
	if err := cmdExists(pager); err != nil {
		return err
	}

	pagerEnv := os.Environ()
	for i := len(pagerEnv) - 1; i >= 0; i-- {
		if strings.HasPrefix(pagerEnv[i], "PAGER=") {
			pagerEnv = append(pagerEnv[0:i], pagerEnv[i+1:]...)
		}
	}
	if _, ok := os.LookupEnv("LESS"); !ok {
		pagerEnv = append(pagerEnv, "LESS=R")
	}

	cmd := exec.Command(pager, pagerArgs...)
	cmd.Env = pagerEnv
	cmd.Stdin = strings.NewReader(out)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func cmdExists(cmd string) error {
	_, err := safeexec.LookPath(cmd)
	return err
}

func customTUIStyle(style TableStyle) tcell.Style {
	bg, ok := tcell.ColorNames[style.SelectionBackground]
	if !ok {
		bg = tcell.ColorDefault
	}
	fg, ok := tcell.ColorNames[style.SelectionForeground]
	if !ok {
		fg = tcell.ColorDarkOliveGreen
	}
	return tcell.StyleDefault.
		Background(bg).
		Foreground(fg).
		Bold(style.SelectionTextIsBold)
}
