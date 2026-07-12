package tui

import (
	"bufio"
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/cli/safeexec"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-isatty"
	"github.com/rivo/tview"

	"github.com/rethab/jira-cli/pkg/tui/primitive"
)

func pad(in string, n uint) string {
	if in == "" {
		return in
	}

	var out strings.Builder

	for range n {
		out.WriteString(" ")
	}

	out.WriteString(in)

	for range n {
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
	modal := tview.NewModal()
	modal.SetText("\n\nProcessing. Please wait...").
		SetBackgroundColor(tcell.ColorSpecial).
		SetTextColor(tcell.ColorDefault)
	modal.Box.SetBackgroundColor(tcell.ColorSpecial)
	return modal
}

func getErrorModal() *tview.Modal {
	modal := tview.NewModal()
	modal.SetBackgroundColor(tcell.ColorSpecial).
		SetTextColor(tcell.ColorRed).
		AddButtons([]string{"OK"})
	modal.Box.SetBackgroundColor(tcell.ColorSpecial)
	return modal
}

func getActionModal() *primitive.ActionModal {
	return primitive.NewActionModal().
		SetBackgroundColor(tcell.ColorSpecial).
		SetButtonBackgroundColor(tcell.ColorDarkCyan).
		SetTextColor(tcell.ColorDefault)
}

// IsDumbTerminal checks TERM/WT_SESSION environment variable and returns true if they indicate a dumb terminal.
//
// Dumb terminal indicates terminal with limited capability. It may not provide support
// for special character sequences, e.g., no handling of ANSI escape sequences.
func IsDumbTerminal() bool {
	term := strings.ToLower(os.Getenv("TERM"))
	_, wtSession := os.LookupEnv("WT_SESSION")
	return !wtSession && (term == "" || term == "dumb")
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
	return cmp.Or(os.Getenv("JIRA_PAGER"), os.Getenv("PAGER"), "less")
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

	pagerEnv := slices.DeleteFunc(os.Environ(), func(env string) bool {
		return strings.HasPrefix(env, "PAGER=")
	})
	if _, ok := os.LookupEnv("LESS"); !ok {
		pagerEnv = append(pagerEnv, "LESS=R")
	}

	// Same as the editor: the pager belongs to the user, not to a request.
	cmd := exec.Command(pager, pagerArgs...) //nolint:noctx
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
