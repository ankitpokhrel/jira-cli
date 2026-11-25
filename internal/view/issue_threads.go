package view

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

type threadComment struct {
	index   int
	depth   int
	warning string
}

type commentThreadSelection struct {
	comments []threadComment
	total    int
	shown    int
	invalid  int
}

func selectCommentThreads(comments []jira.IssueComment, limit uint) commentThreadSelection {
	parents, warnings := commentParents(comments)
	resolveParentWarnings(parents, warnings)
	children := make([][]int, len(comments))
	roots := make([]int, 0)
	invalid := make([]int, 0)
	for idx := range comments {
		switch {
		case warnings[idx] != "":
			invalid = append(invalid, idx)
		case parents[idx] < 0:
			roots = append(roots, idx)
		default:
			children[parents[idx]] = append(children[parents[idx]], idx)
		}
	}
	timestamps := make([]time.Time, len(comments))
	for idx, comment := range comments {
		timestamps[idx], _ = commentTimestamp(comment.Created)
	}
	less := func(a, b int, newest bool) bool {
		if !timestamps[a].Equal(timestamps[b]) {
			if newest {
				return timestamps[a].After(timestamps[b])
			}
			return timestamps[a].Before(timestamps[b])
		}
		return comments[a].ID < comments[b].ID
	}
	sort.SliceStable(roots, func(a, b int) bool { return less(roots[a], roots[b], true) })
	for _, replies := range children {
		sort.SliceStable(replies, func(a, b int) bool { return less(replies[a], replies[b], false) })
	}
	sort.SliceStable(invalid, func(a, b int) bool { return less(invalid[a], invalid[b], false) })
	selection := commentThreadSelection{
		total: len(roots), shown: int(min(limit, uint(len(roots)))), invalid: len(invalid),
	}
	if limit == 0 {
		return selection
	}
	// An explicit stack avoids recursion on untrusted, deeply nested parent chains.
	for _, root := range roots[:selection.shown] {
		stack := []threadComment{{index: root}}
		for len(stack) > 0 {
			entry := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			selection.comments = append(selection.comments, entry)
			replies := children[entry.index]
			for idx := len(replies) - 1; idx >= 0; idx-- {
				stack = append(stack, threadComment{index: replies[idx], depth: entry.depth + 1})
			}
		}
	}
	// Invalid entries remain visible without pretending that they are native roots.
	for _, idx := range invalid {
		selection.comments = append(selection.comments, threadComment{index: idx, warning: warnings[idx]})
	}
	return selection
}

func commentParents(comments []jira.IssueComment) ([]int, []string) {
	indices := make(map[string]int, len(comments))
	counts := make(map[string]int, len(comments))
	for idx, comment := range comments {
		indices[comment.ID] = idx
		counts[comment.ID]++
	}
	parents := make([]int, len(comments))
	warnings := make([]string, len(comments))
	for idx, comment := range comments {
		parents[idx] = -1
		switch {
		case comment.ID == "":
			warnings[idx] = "missing comment ID"
		case counts[comment.ID] > 1:
			warnings[idx] = "duplicate comment ID"
		case comment.ParentID == "":
			continue
		case counts[comment.ParentID] == 0:
			warnings[idx] = "missing parent " + comment.ParentID
		case counts[comment.ParentID] > 1:
			warnings[idx] = "ambiguous duplicate parent " + comment.ParentID
		default:
			parents[idx] = indices[comment.ParentID]
		}
	}
	return parents, warnings
}

func resolveParentWarnings(parents []int, warnings []string) {
	const (
		unvisited = iota
		visiting
		resolved
	)
	states := make([]int, len(parents))
	for start := range parents {
		if states[start] != unvisited {
			continue
		}
		var path []int
		idx := start
		for idx >= 0 && states[idx] == unvisited {
			states[idx] = visiting
			path = append(path, idx)
			if warnings[idx] != "" {
				break
			}
			idx = parents[idx]
		}
		reason := ""
		if idx >= 0 {
			if warnings[idx] != "" {
				reason = "unresolved parent chain"
			} else if states[idx] == visiting {
				reason = "cyclic parent chain"
			}
		}
		for _, entry := range path {
			if warnings[entry] == "" {
				warnings[entry] = reason
			}
			states[entry] = resolved
		}
	}
}

func commentTimestamp(created string) (time.Time, error) {
	stamp, err := time.Parse(jira.RFC3339, created)
	if err == nil {
		return stamp, nil
	}
	return time.Parse(time.RFC3339Nano, created)
}

func (i Issue) threadedComments() []issueComment {
	comments := i.Data.Fields.Comment.Comments
	selection := selectCommentThreads(comments, i.Options.NumComments)
	latest := -1
	for idx, comment := range comments {
		stamp, err := commentTimestamp(comment.Created)
		if err != nil {
			// An unknown timestamp makes a global "latest" claim unsafe.
			latest = -1
			break
		}
		if latest < 0 {
			latest = idx
			continue
		}
		previous, _ := commentTimestamp(comments[latest].Created)
		if stamp.After(previous) || (stamp.Equal(previous) && comment.ID < comments[latest].ID) {
			latest = idx
		}
	}
	out := make([]issueComment, 0, len(selection.comments))
	for _, entry := range selection.comments {
		comment := comments[entry.index]
		author := comment.Author.DisplayName
		if author == "" {
			author = comment.Author.Name
		}
		meta := fmt.Sprintf("\n %s • %s", coloredOut(author, color.FgWhite, color.Bold),
			coloredOut(cmdutil.FormatDateTimeHuman(comment.Created, jira.RFC3339), color.FgWhite, color.Bold))
		switch {
		case entry.warning != "":
			meta += fmt.Sprintf(" • Invalid thread data: %s (comment %q, parent %q)", entry.warning, comment.ID, comment.ParentID)
		case comment.ParentID == "":
			meta += " • Thread " + comment.ID
		default:
			meta += fmt.Sprintf(" • Reply %s to %s", comment.ID, comment.ParentID)
		}
		if _, err := commentTimestamp(comment.Created); err != nil {
			meta += " • Invalid comment timestamp"
		}
		if entry.index == latest {
			meta += " • " + coloredOut("Latest comment", color.FgCyan, color.Bold)
		}
		if entry.depth > maxCommentIndentDepth {
			meta += fmt.Sprintf(" • depth %d", entry.depth)
		}
		out = append(out, issueComment{meta: meta, body: threadedCommentBody(comment.Body), depth: entry.depth})
	}
	return out
}

func threadedCommentBody(body any) string {
	switch value := body.(type) {
	case *adf.ADF:
		if value != nil {
			return adf.NewTranslator(value, adf.NewMarkdownTranslator()).Translate()
		}
	case string:
		return md.FromJiraMD(value)
	case nil:
		return "[Comment body unavailable]"
	default:
		return fmt.Sprintf("[Unsupported comment body: %T]", body)
	}
	return "[Comment body unavailable]"
}

func (i Issue) replyTargets() ([]tui.ReplyTarget, string) {
	if !i.Options.CommentThreads {
		return nil, "Enable --comment-threads to choose a native thread"
	}
	if i.Options.NumComments == 0 {
		return nil, "Use --comments <limit> greater than 0 to load native threads"
	}
	comments := i.Data.Fields.Comment.Comments
	selection := selectCommentThreads(comments, i.Options.NumComments)
	targets := make([]tui.ReplyTarget, 0, selection.shown)
	for _, entry := range selection.comments {
		comment := comments[entry.index]
		if entry.depth != 0 || entry.warning != "" || comment.ParentID != "" {
			continue
		}
		author := comment.Author.DisplayName
		if author == "" {
			author = comment.Author.Name
		}
		targets = append(targets, tui.ReplyTarget{
			ID:      comment.ID,
			Author:  author,
			Created: cmdutil.FormatDateTimeHuman(comment.Created, jira.RFC3339),
			Preview: commentReplyPreview(threadedCommentBody(comment.Body)),
		})
	}
	if len(targets) == 0 {
		return nil, "No valid native root threads are available in the loaded comments"
	}
	return targets, ""
}

func commentReplyPreview(body string) string {
	const previewRunes = 100
	preview := []rune(strings.Join(strings.Fields(body), " "))
	if len(preview) > previewRunes {
		return string(preview[:previewRunes-1]) + "…"
	}
	return string(preview)
}

const maxCommentIndentDepth = 12

func indentComment(body string, depth int) string {
	if depth <= 0 {
		return body
	}
	indent := strings.Repeat("  ", min(depth, maxCommentIndentDepth))
	lines := strings.Split(body, "\n")
	for idx, line := range lines {
		if line != "" {
			lines[idx] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}
