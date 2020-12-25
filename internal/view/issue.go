package view

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const wordWrap = 120

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

	data := i.data()
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		return err
	}
	out, err := r.Render(string(data))
	if err != nil {
		return err
	}
	return PagerOut(out)
}

func (i Issue) data() tui.TextData {
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
	it, iti := i.Data.Fields.IssueType.Name, "⭐"
	if it == "Bug" {
		iti = "🐞"
	}
	tr := adf.NewTranslator(i.Data.Fields.Description, &adf.MarkdownTranslator{})
	dt := fmt.Sprintf(
		"%s %s  %s %s  ⌛ %s  👷 %s\n# %s\n⏱️  %s  🔎 %s  🚀 %s  🏷️  %s\n\n-----------\n%s",
		iti, it, sti, st, formatDateTimeHuman(i.Data.Fields.Updated, jira.RFC3339), as,
		i.Data.Fields.Summary,
		formatDateTimeHuman(i.Data.Fields.Created, jira.RFC3339), i.Data.Fields.Reporter.Name,
		i.Data.Fields.Priority.Name, lbl,
		tr.Translate(),
	)

	return tui.TextData(dt)
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
	out, err := r.Render(string(i.data()))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, out)
	return err
}
