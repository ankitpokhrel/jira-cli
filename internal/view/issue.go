package view

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const defaultSummaryLength = 73 // +1 to take ellipsis '…' into account.

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

type issueComment struct {
	meta string
	body string
}

// IssueOption is filtering options for an issue.
type IssueOption struct {
	NumComments uint
}

// Issue is a list view for issues.
type Issue struct {
	Server  string
	Data    *jira.Issue
	Display DisplayFormat
	Options IssueOption
}

// Render renders the view.
func (i Issue) Render() error {
	if i.Display.Plain || tui.IsDumbTerminal() || tui.IsNotTTY() {
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
	var s strings.Builder

	s.WriteString(i.header())

	desc := i.description()
	if desc != "" {
		s.WriteString(fmt.Sprintf("\n\n%s\n\n%s", i.separator("Description"), desc))
	}
	if len(i.Data.Fields.Subtasks) > 0 {
		s.WriteString(
			fmt.Sprintf(
				"\n\n%s\n\n%s\n",
				i.separator(fmt.Sprintf("%d Subtasks", len(i.Data.Fields.Subtasks))),
				i.subtasks(),
			),
		)
	}
	if len(i.Data.Fields.IssueLinks) > 0 {
		s.WriteString(fmt.Sprintf("\n\n%s\n\n%s\n", i.separator("Linked Issues"), i.linkedIssues()))
	}
	total := i.Data.Fields.Comment.Total
	if total > 0 && i.Options.NumComments > 0 {
		sep := fmt.Sprintf("%d Comments", total)
		s.WriteString(fmt.Sprintf("\n\n%s", i.separator(sep)))
		for _, comment := range i.comments() {
			s.WriteString(fmt.Sprintf("\n\n%s\n\n%s\n", comment.meta, comment.body))
		}
	}
	s.WriteString(i.footer())

	return s.String()
}

func (i Issue) fragments() []fragment {
	scraps := []fragment{
		{Body: i.header(), Parse: true},
	}

	desc := i.description()
	if desc != "" {
		scraps = append(
			scraps,
			newBlankFragment(1),
			fragment{Body: i.separator("Description")},
			newBlankFragment(2),
			fragment{Body: desc, Parse: true},
		)
	}

	if len(i.Data.Fields.Subtasks) > 0 {
		scraps = append(
			scraps,
			newBlankFragment(1),
			fragment{Body: i.separator(fmt.Sprintf("%d Subtasks", len(i.Data.Fields.Subtasks)))},
			newBlankFragment(2),
			fragment{Body: i.subtasks()},
			newBlankFragment(1),
		)
	}

	if len(i.Data.Fields.IssueLinks) > 0 {
		scraps = append(
			scraps,
			newBlankFragment(1),
			fragment{Body: i.separator("Linked Issues")},
			newBlankFragment(2),
			fragment{Body: i.linkedIssues()},
			newBlankFragment(1),
		)
	}

	if i.Data.Fields.Comment.Total > 0 && i.Options.NumComments > 0 {
		scraps = append(
			scraps,
			newBlankFragment(1),
			fragment{Body: i.separator(fmt.Sprintf("%d Comments", i.Data.Fields.Comment.Total))},
			newBlankFragment(2),
		)
		for _, comment := range i.comments() {
			scraps = append(
				scraps,
				fragment{Body: comment.meta},
				newBlankFragment(1),
				fragment{Body: comment.body, Parse: true},
			)
		}
	}

	return append(scraps, newBlankFragment(1), fragment{Body: i.footer()}, newBlankFragment(2))
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
	sep := "————————————————————————"
	if msg == "" {
		return gray(fmt.Sprintf("%s%s", sep, sep))
	}
	return gray(fmt.Sprintf("%s%s%s", sep, pad(msg), sep))
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
		"%s %s  %s %s  ⌛ %s  👷 %s  🔑️ %s  💭 %d comments  \U0001F9F5 %d linked\n# %s\n⏱️  %s  🔎 %s  🚀 %s  📦 %s  🏷️  %s  👀 %s",
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

func (i Issue) subtasks() string {
	if len(i.Data.Fields.Subtasks) == 0 {
		return ""
	}

	var (
		subtasks       strings.Builder
		summaryLen     = defaultSummaryLength
		maxKeyLen      int
		maxSummaryLen  int
		maxStatusLen   int
		maxPriorityLen int
	)

	for idx := range i.Data.Fields.Subtasks {
		task := i.Data.Fields.Subtasks[idx]

		maxKeyLen = max(len(task.Key), maxKeyLen)
		maxSummaryLen = max(len(task.Fields.Summary), maxSummaryLen)
		maxStatusLen = max(len(task.Fields.Status.Name), maxStatusLen)
		maxPriorityLen = max(len(task.Fields.Priority.Name), maxPriorityLen)
	}

	if maxSummaryLen < summaryLen {
		summaryLen = maxSummaryLen
	}

	subtasks.WriteString(
		fmt.Sprintf("\n %s\n\n", coloredOut("SUBTASKS", color.FgWhite, color.Bold)),
	)
	for idx := range i.Data.Fields.Subtasks {
		task := i.Data.Fields.Subtasks[idx]
		subtasks.WriteString(
			fmt.Sprintf(
				"  %s %s • %s • %s\n",
				coloredOut(pad(task.Key, maxKeyLen), color.FgGreen, color.Bold),
				shortenAndPad(task.Fields.Summary, summaryLen),
				pad(task.Fields.Priority.Name, maxPriorityLen),
				pad(task.Fields.Status.Name, maxStatusLen),
			),
		)
	}

	return subtasks.String()
}

func (i Issue) linkedIssues() string {
	if len(i.Data.Fields.IssueLinks) == 0 {
		return ""
	}

	var (
		linked         strings.Builder
		keys           = make([]string, 0)
		linkMap        = make(map[string][]*jira.Issue, len(i.Data.Fields.IssueLinks))
		summaryLen     = defaultSummaryLength
		maxKeyLen      int
		maxSummaryLen  int
		maxTypeLen     int
		maxStatusLen   int
		maxPriorityLen int
	)

	for _, link := range i.Data.Fields.IssueLinks {
		var (
			linkType    string
			linkedIssue *jira.Issue
		)

		if link.InwardIssue != nil {
			linkType = link.LinkType.Inward
			linkedIssue = link.InwardIssue
		} else if link.OutwardIssue != nil {
			linkType = link.LinkType.Outward
			linkedIssue = link.OutwardIssue
		}

		if linkedIssue == nil {
			continue
		}

		if _, ok := linkMap[linkType]; !ok {
			keys = append(keys, linkType)
		}
		linkMap[linkType] = append(linkMap[linkType], linkedIssue)

		maxKeyLen = max(len(linkedIssue.Key), maxKeyLen)
		maxSummaryLen = max(len(linkedIssue.Fields.Summary), maxSummaryLen)
		maxTypeLen = max(len(linkedIssue.Fields.IssueType.Name), maxTypeLen)
		maxStatusLen = max(len(linkedIssue.Fields.Status.Name), maxStatusLen)
		maxPriorityLen = max(len(linkedIssue.Fields.Priority.Name), maxPriorityLen)
	}

	if maxSummaryLen < summaryLen {
		summaryLen = maxSummaryLen
	}

	// We are sorting keys to respect the order we see in the UI.
	sort.Strings(keys)

	for _, k := range keys {
		linked.WriteString(
			fmt.Sprintf("\n %s\n\n", coloredOut(strings.ToUpper(k), color.FgWhite, color.Bold)),
		)
		for _, iss := range linkMap[k] {
			linked.WriteString(
				fmt.Sprintf(
					"  %s %s • %s • %s • %s\n",
					coloredOut(pad(iss.Key, maxKeyLen), color.FgGreen, color.Bold),
					shortenAndPad(iss.Fields.Summary, summaryLen),
					pad(iss.Fields.IssueType.Name, maxTypeLen),
					pad(iss.Fields.Priority.Name, maxPriorityLen),
					pad(iss.Fields.Status.Name, maxStatusLen),
				),
			)
		}
	}

	return linked.String()
}

func (i Issue) comments() []issueComment {
	total := i.Data.Fields.Comment.Total
	comments := make([]issueComment, 0, total)

	if total == 0 {
		return comments
	}

	limit := int(i.Options.NumComments)
	if limit > total {
		limit = total
	}

	for idx := total - 1; idx >= total-limit; idx-- {
		c := i.Data.Fields.Comment.Comments[idx]
		var body string
		if adfNode, ok := c.Body.(*adf.ADF); ok {
			body = adf.NewTranslator(adfNode, adf.NewMarkdownTranslator()).Translate()
		} else {
			body = c.Body.(string)
			body = md.FromJiraMD(body)
		}
		meta := fmt.Sprintf(
			"\n %s • %s",
			coloredOut(c.Author.Name, color.FgWhite, color.Bold),
			coloredOut(cmdutil.FormatDateTimeHuman(c.Created, jira.RFC3339), color.FgWhite, color.Bold),
		)
		if idx == total-1 {
			meta += fmt.Sprintf(" • %s", coloredOut("Latest comment", color.FgCyan, color.Bold))
		}
		comments = append(comments, issueComment{
			meta: meta,
			body: body,
		})
	}

	return comments
}

func (i Issue) footer() string {
	var out strings.Builder

	nc := int(i.Options.NumComments)
	if i.Data.Fields.Comment.Total > 0 && nc > 0 && nc < i.Data.Fields.Comment.Total {
		if i.Display.Plain {
			out.WriteString("\n")
		}
		out.WriteString(fmt.Sprintf("%s\n", gray("Use --comments <limit> with `jira issue view` to load more comments")))
	}
	if i.Display.Plain {
		out.WriteString("\n")
	}
	out.WriteString(gray(fmt.Sprintf("View this issue on Jira: %s/browse/%s", i.Server, i.Data.Key)))

	return out.String()
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
	_, err = fmt.Fprint(w, out)
	return err
}
