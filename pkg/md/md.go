package md

import (
	"regexp"
	"strings"

	cf "github.com/kentaro-m/blackfriday-confluence"
	bf "github.com/russross/blackfriday/v2"

	"github.com/ankitpokhrel/jira-cli/pkg/md/jirawiki"
)

// jiraNestedListPattern matches Jira wiki nested list syntax: lines starting with
// two or more asterisks followed by a space (e.g., "** item", "*** sub-item").
// Single asterisk lines are handled correctly by BlackFriday, so we only need
// to convert nested lists (2+ asterisks).
var jiraNestedListPattern = regexp.MustCompile(`^(\*{2,})\s`)

// convertJiraNestedLists converts Jira wiki nested list syntax to CommonMark.
// Jira uses "** item" for nested lists, but BlackFriday interprets "**" as bold.
// This function converts "** item" to "\t- item" (tab-indented CommonMark list).
func convertJiraNestedLists(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		if match := jiraNestedListPattern.FindStringSubmatch(line); match != nil {
			depth := len(match[1]) // Number of asterisks
			indent := strings.Repeat("\t", depth-1)
			rest := strings.TrimPrefix(line, match[0])
			lines[i] = indent + "- " + rest
		}
	}
	return strings.Join(lines, "\n")
}

// ToJiraMD translates CommonMark to Jira flavored markdown.
func ToJiraMD(md string) string {
	if md == "" {
		return md
	}

	// Preprocess: convert Jira wiki nested lists to CommonMark format
	// so BlackFriday can parse them correctly.
	md = convertJiraNestedLists(md)

	renderer := &cf.Renderer{Flags: cf.IgnoreMacroEscaping}
	r := bf.New(bf.WithRenderer(renderer), bf.WithExtensions(bf.CommonExtensions))

	return string(renderer.Render(r.Parse([]byte(md))))
}

// FromJiraMD translates Jira flavored markdown to CommonMark.
func FromJiraMD(jfm string) string {
	return jirawiki.Parse(jfm)
}
