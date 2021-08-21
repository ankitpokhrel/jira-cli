package view

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

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
	out, err := r.Render(i.String())
	if err != nil {
		return err
	}
	return PagerOut(out)
}

func (i Issue) String() string {
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
	if len(i.Data.Fields.Labels) > 0 {
	}
	it, iti := i.Data.Fields.IssueType.Name, "⭐"
	if it == "Bug" {
		iti = "🐞"
	}
	desc := ""
	if i.Data.Fields.Description != nil {
		tr := adf.NewTranslator(i.Data.Fields.Description.(*adf.ADF), adf.NewMarkdownTranslator())
		desc = tr.Translate()
	}
	return fmt.Sprintf(
		"%s %s  %s %s  ⌛ %s  👷 %s  🔑️ %s\n# %s\n⏱️  %s  🔎 %s  🚀 %s  📦 %s  🏷️  %s\n\n-----------\n%s",
		iti, it, sti, st, cmdutil.FormatDateTimeHuman(i.Data.Fields.Updated, jira.RFC3339), as, i.Data.Key,
		i.Data.Fields.Summary,
		cmdutil.FormatDateTimeHuman(i.Data.Fields.Created, jira.RFC3339), i.Data.Fields.Reporter.Name,
		i.Data.Fields.Priority.Name, cmpt, lbl,
		desc,
	)
}

func (i Issue) data() tui.TextData {
	return tui.TextData(i.String())
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
