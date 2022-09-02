package root

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/kr/text"
	"github.com/spf13/cobra"
)

type helpEntry struct {
	Title string
	Body  string
}

func helpFunc(cmd *cobra.Command, _ []string) {
	entries := getEntries(cmd)

	out := cmd.OutOrStdout()
	for _, e := range entries {
		if e.Title != "" {
			fmt.Fprintf(out, "\033[1m%s\033[0m\n", e.Title)
			fmt.Fprintln(out, text.Indent(strings.Trim(e.Body, "\r\n"), "  "))
		} else {
			fmt.Fprintln(out, e.Body)
		}
		fmt.Fprintln(out)
	}
}

func getEntries(cmd *cobra.Command) []helpEntry {
	mainCmds, otherCmds := groupCommands(cmd.Commands())

	var entries []helpEntry

	appendIfNotEmpty := func(b interface{}, t string) {
		switch d := b.(type) {
		case string:
			if b != "" {
				entries = append(entries, helpEntry{Title: t, Body: d})
			}
		case []string:
			if len(d) > 0 {
				entries = append(entries, helpEntry{Title: t, Body: strings.Join(d, "\n")})
			}
		}
	}

	if cmd.Long != "" {
		entries = append(entries, helpEntry{"", cmd.Long})
	} else if cmd.Short != "" {
		entries = append(entries, helpEntry{"", cmd.Short})
	}

	appendIfNotEmpty(cmd.UseLine(), "USAGE")
	appendIfNotEmpty(mainCmds, "MAIN COMMANDS")
	appendIfNotEmpty(otherCmds, "OTHER COMMANDS")
	appendIfNotEmpty(outdent(cmd.LocalFlags().FlagUsages()), "FLAGS")
	appendIfNotEmpty(outdent(cmd.InheritedFlags().FlagUsages()), "INHERITED FLAGS")
	if _, ok := cmd.Annotations["help:args"]; ok {
		appendIfNotEmpty(cmd.Annotations["help:args"], "ARGUMENTS")
	}
	appendIfNotEmpty(cmd.Example, "EXAMPLES")
	appendIfNotEmpty(cmd.Aliases, "ALIASES")
	entries = append(entries, helpEntry{
		"LEARN MORE",
		`Use 'jira <command> <subcommand> --help' for more information about a command.
Read the doc or get help at https://github.com/ankitpokhrel/jira-cli`,
	})

	return entries
}

func groupCommands(cmds []*cobra.Command) ([]string, []string) {
	var primary, secondary []string

	for _, c := range cmds {
		if c.Short == "" {
			continue
		}
		if c.Hidden {
			continue
		}

		s := rpad(c.Name(), c.NamePadding()) + c.Short
		if _, ok := c.Annotations["cmd:main"]; ok {
			primary = append(primary, s)
		} else {
			secondary = append(secondary, s)
		}
	}

	if len(primary) == 0 {
		primary = secondary
		secondary = []string{}
	}

	return primary, secondary
}

func outdent(s string) string {
	lines, minIndent := strings.Split(s, "\n"), -1

	for _, l := range lines {
		if l == "" {
			continue
		}

		indent := len(l) - len(strings.TrimLeft(l, " "))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	var buf bytes.Buffer
	for _, l := range lines {
		fmt.Fprintln(&buf, strings.TrimPrefix(l, strings.Repeat(" ", minIndent)))
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func rpad(s string, pad int) string {
	template := fmt.Sprintf("%%-%ds ", pad)
	return fmt.Sprintf(template, s)
}
