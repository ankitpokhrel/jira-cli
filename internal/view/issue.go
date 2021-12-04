package view

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

type fragment struct {
	Body  string
	Parse bool
}

func newBlankFragment(n int) fragment {
	var buf strings.Builder
	for i := 0; i < n; i++ {
		buf.WriteRune('\n')
	}
	return fragment{
		Body:  buf.String(),
		Parse: false,
	}
}

// Issue is a list view for issues.
type Issue struct {
	Data    *jira.Issue
	Display DisplayFormat
}

// Render renders the view.
func (i Issue) Render() error {
	if i.Display.Plain {
		return i.renderPlain(os.Stdout)
	}
	r, err := MDRenderer()
	if err != nil {
		return err
	}
	out, err := i.RenderedOut(r)
	if err != nil {
		return err
	}
	return tui.PagerOut(out)
}

// RenderedOut translates raw data to the format we want to display in.
func (i Issue) RenderedOut(renderer *glamour.TermRenderer) (string, error) {
	var res strings.Builder

	for _, p := range i.fragments() {
		if p.Parse {
			out, err := renderer.Render(p.Body)
			if err != nil {
				return "", err
			}
			res.WriteString(out)
		} else {
			res.WriteString(p.Body)
		}
	}

	return res.String(), nil
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		i.header(),
		i.separator("Description"),
		i.description(),
	)
}

func (i Issue) fragments() []fragment {
	return []fragment{
		{Body: i.header(), Parse: true},
		newBlankFragment(1),
		{Body: i.separator("Description"), Parse: false},
		newBlankFragment(2),
		{Body: i.description(), Parse: true},
	}
}

func (i Issue) separator(msg string) string {
	pad := func(m string) string {
		if m != "" {
			return fmt.Sprintf(" %s ", m)
		}
		return m
	}

	if i.Display.Plain {
		sep := "------------------------"
		return fmt.Sprintf("%s%s%s", sep, pad(msg), sep)
	}
	cyan := color.New(color.FgHiCyan)
	sep := cyan.Sprintf("————————————————————————")
	return fmt.Sprintf("%s%s%s", sep, cyan.Sprint(pad(msg)), sep)
}

func (i Issue) header() string {
	as := i.Data.Fields.Assignee.Name
	if as == "" {
		as = "Unassigned"
	}
	st, sti := i.Data.Fields.Status.Name, "🚧"
	if st == "Done" {
		sti = "✅"
	}
	lbl := "None"
	if len(i.Data.Fields.Labels) > 0 {
		lbl = strings.Join(i.Data.Fields.Labels, ", ")
	}
	components := make([]string, 0, len(i.Data.Fields.Components))
	for _, c := range i.Data.Fields.Components {
		components = append(components, c.Name)
	}
	cmpt := "None"
	if len(components) > 0 {
		cmpt = strings.Join(components, ", ")
	}
	it, iti := i.Data.Fields.IssueType.Name, "⭐"
	if it == "Bug" {
		iti = "🐞"
	}
	wch := fmt.Sprintf("%d watchers", i.Data.Fields.Watches.WatchCount)
	if i.Data.Fields.Watches.WatchCount == 1 && i.Data.Fields.Watches.IsWatching {
		wch = "You are watching"
	} else if i.Data.Fields.Watches.IsWatching {
		wch = fmt.Sprintf("You + %d watchers", i.Data.Fields.Watches.WatchCount-1)
	}
	return fmt.Sprintf(
		"%s %s  %s %s  ⌛ %s  👷 %s  🔑️ %s  💭 %d comments  \U0001F9F5 %d linked issues\n# %s\n⏱️  %s  🔎 %s  🚀 %s  📦 %s  🏷️  %s  👀 %s",
		iti, it, sti, st, cmdutil.FormatDateTimeHuman(i.Data.Fields.Updated, jira.RFC3339), as, i.Data.Key,
		i.Data.Fields.Comment.Total, len(i.Data.Fields.IssueLinks),
		i.Data.Fields.Summary,
		cmdutil.FormatDateTimeHuman(i.Data.Fields.Created, jira.RFC3339), i.Data.Fields.Reporter.Name,
		i.Data.Fields.Priority.Name, cmpt, lbl, wch,
	)
}

func (i Issue) description() string {
	if i.Data.Fields.Description == nil {
		return ""
	}

	var desc string

	if adfNode, ok := i.Data.Fields.Description.(*adf.ADF); ok {
		desc = adf.NewTranslator(adfNode, adf.NewMarkdownTranslator()).Translate()
	} else {
		desc = i.Data.Fields.Description.(string)
		desc = md.FromJiraMD(desc)
	}

	return desc
}

// renderPlain renders the issue in plain view.
func (i Issue) renderPlain(w io.Writer) error {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("notty"),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		return err
	}
	out, err := r.Render(i.String())
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, out)
	return err
}
