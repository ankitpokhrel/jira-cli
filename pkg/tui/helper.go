package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

func cmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
