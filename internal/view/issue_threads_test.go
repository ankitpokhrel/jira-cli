package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/md"
)

func TestSelectCommentThreads(t *testing.T) {
	t.Parallel()
	comments := []jira.IssueComment{
		threadTestComment("older", "", "2026-01-01T01:00:00Z", "Older root"),
		threadTestComment("newest-reply", "older", "2026-01-04T01:00:00Z", "Latest reply on older root"),
		threadTestComment("b", "", "2026-01-02T02:00:00+0100", "Tied root B"),
		threadTestComment("nested", "early", "2026-01-03T02:00:00Z", "Nested reply"),
		threadTestComment("late", "a", "2026-01-03T01:00:00Z", "Later reply"),
		threadTestComment("a", "", "2026-01-02T01:00:00Z", "Tied root A"),
		threadTestComment("early", "a", "2026-01-02T02:00:00Z", "Earlier reply"),
	}
	for _, tc := range []struct {
		name  string
		limit uint
		ids   []string
		depth []int
		shown int
	}{
		{name: "none", limit: 0},
		{name: "one complete thread", limit: 1, ids: []string{"a", "early", "nested", "late"}, depth: []int{0, 1, 2, 1}, shown: 1},
		{name: "two roots", limit: 2, ids: []string{"a", "early", "nested", "late", "b"}, depth: []int{0, 1, 2, 1, 0}, shown: 2},
		{name: "uint maximum", limit: ^uint(0), ids: []string{"a", "early", "nested", "late", "b", "older", "newest-reply"}, depth: []int{0, 1, 2, 1, 0, 0, 1}, shown: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := selectCommentThreads(comments, tc.limit)
			if got.total != 3 || got.shown != tc.shown || got.invalid != 0 {
				t.Fatalf("counts = total %d, shown %d, invalid %d", got.total, got.shown, got.invalid)
			}
			var ids []string
			var depths []int
			for _, entry := range got.comments {
				ids = append(ids, comments[entry.index].ID)
				depths = append(depths, entry.depth)
			}
			if !reflect.DeepEqual(ids, tc.ids) || !reflect.DeepEqual(depths, tc.depth) {
				t.Errorf("selection = %v at %v, want %v at %v", ids, depths, tc.ids, tc.depth)
			}
		})
	}
}

func TestCommentThreadOrderingIsRepeatable(t *testing.T) {
	t.Parallel()
	comments := []jira.IssueComment{
		threadTestComment("b", "", "2026-01-01T01:00:00Z", "B"),
		threadTestComment("reply-b", "a", "2026-01-02T01:00:00Z", "Reply B"),
		threadTestComment("a", "", "2026-01-01T01:00:00Z", "A"),
		threadTestComment("reply-a", "a", "2026-01-02T01:00:00Z", "Reply A"),
	}
	rng := rand.New(rand.NewSource(1))
	const attempts = 20
	for range attempts {
		rng.Shuffle(len(comments), func(a, b int) { comments[a], comments[b] = comments[b], comments[a] })
		before := append([]jira.IssueComment(nil), comments...)
		got := selectCommentThreads(comments, ^uint(0))
		ids := make([]string, 0, len(got.comments))
		for _, entry := range got.comments {
			ids = append(ids, comments[entry.index].ID)
		}
		if want := []string{"a", "reply-a", "reply-b", "b"}; !reflect.DeepEqual(ids, want) {
			t.Fatalf("unstable order %v, want %v", ids, want)
		}
		if !reflect.DeepEqual(before, comments) {
			t.Fatal("selection mutated the source comments")
		}
	}
}

func TestInvalidCommentThreadsRemainVisible(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		comments []jira.IssueComment
		want     string
	}{
		{name: "orphan", comments: []jira.IssueComment{{ID: "orphan", ParentID: "absent"}}, want: "missing parent absent"},
		{name: "missing ID", comments: []jira.IssueComment{{}}, want: "missing comment ID"},
		{name: "duplicate", comments: []jira.IssueComment{{ID: "dup"}, {ID: "dup"}}, want: "duplicate comment ID"},
		{name: "duplicate parent", comments: []jira.IssueComment{{ID: "dup"}, {ID: "dup"}, {ID: "child", ParentID: "dup"}}, want: "ambiguous duplicate parent dup"},
		{name: "self cycle", comments: []jira.IssueComment{{ID: "self", ParentID: "self"}}, want: "cyclic parent chain"},
		{name: "cycle", comments: []jira.IssueComment{{ID: "a", ParentID: "b"}, {ID: "b", ParentID: "a"}}, want: "cyclic parent chain"},
		{name: "invalid ancestor", comments: []jira.IssueComment{{ID: "child", ParentID: "orphan"}, {ID: "orphan", ParentID: "absent"}}, want: "unresolved parent chain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			comments := append([]jira.IssueComment{threadTestComment("root", "", "2026-01-01T01:00:00Z", "Valid root")}, tc.comments...)
			issue := newThreadTestIssue(comments, 1)
			got := selectCommentThreads(comments, 1)
			if got.total != 1 || got.shown != 1 || got.invalid != len(tc.comments) || len(got.comments) != len(comments) {
				t.Fatalf("lost or invented threads/comments: %+v", got)
			}
			for _, entry := range got.comments[1:] {
				if entry.depth != 0 || entry.warning == "" {
					t.Errorf("invalid entry attached to an invented edge: %+v", entry)
				}
			}
			text := issue.String()
			if !strings.Contains(text, "Invalid thread data:") || !strings.Contains(text, tc.want) {
				t.Errorf("missing invalid-data label %q in %q", tc.want, text)
			}
			if strings.Count(text, "Invalid thread data:") != len(tc.comments) {
				t.Errorf("did not label every invalid comment: %q", text)
			}
			if !strings.Contains(text, "Showing 1 of 1 threads") || !strings.Contains(text, "unthreaded comments with invalid thread data") {
				t.Errorf("misleading footer: %q", issue.footer())
			}
			issue.Options.NumComments = 0
			if len(issue.comments()) != 0 || strings.Contains(issue.String(), "Invalid thread data:") {
				t.Fatal("zero limit must hide invalid comment bodies too")
			}
		})
	}
}

func TestCommentThreadRendering(t *testing.T) {
	t.Parallel()
	wiki := "First *bold* paragraph\n\nSecond paragraph\n\n* One\n* Two\n\n{code:go}\nfmt.Println(\"test\")\n{code}"
	doc := &adf.ADF{
		Version: 1, DocType: "doc",
		Content: []*adf.Node{{NodeType: "paragraph", Content: []*adf.Node{{NodeType: "text", NodeValue: adf.NodeValue{Text: "ADF reply paragraph"}}}}},
	}
	comments := []jira.IssueComment{
		threadTestComment("root", "", "2026-01-01T01:00:00Z", "Root body"),
		threadTestComment("reply", "root", "2026-01-02T01:00:00Z", doc),
		threadTestComment("nested", "reply", "2026-01-03T01:00:00Z", wiki),
	}
	issue := newThreadTestIssue(comments, 1)
	bodies := issue.comments()
	if len(bodies) != len(comments) || bodies[1].body != "ADF reply paragraph\n\n" || bodies[2].body != md.FromJiraMD(wiki) {
		t.Fatalf("body translation changed: %+v", bodies)
	}
	if !strings.Contains(issue.String(), "\n    First **bold** paragraph") || !strings.Contains(issue.String(), "\n     Tester •") {
		t.Errorf("String output lacks nested body/metadata indentation: %q", issue.String())
	}
	for _, style := range []string{"notty", "dark"} {
		t.Run(style, func(t *testing.T) {
			t.Parallel()
			renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(wordWrap))
			if err != nil {
				t.Fatal(err)
			}
			out, err := issue.RenderedOut(renderer)
			if err != nil {
				t.Fatal(err)
			}
			for _, fragment := range issue.fragments() {
				if fragment.Indent == 0 {
					continue
				}
				if fragment.Parse {
					if fragment.Indent == 2 && fragment.Body != md.FromJiraMD(wiki) {
						t.Errorf("nested Markdown was indented before rendering: %q", fragment.Body)
					}
					body, renderErr := renderer.Render(fragment.Body)
					if renderErr != nil {
						t.Fatal(renderErr)
					}
					if !strings.Contains(out, indentComment(body, fragment.Indent)) {
						t.Errorf("body was not indented after Markdown rendering: %q", out)
					}
				}
			}
			if !strings.Contains(out, "Reply nested to reply") || !strings.Contains(out, "Showing 1 of 1 threads") {
				t.Errorf("missing nesting metadata/counts: %q", out)
			}
		})
	}
}

func TestCommentThreadPlainRendering(t *testing.T) {
	t.Parallel()
	issue := newThreadTestIssue([]jira.IssueComment{
		threadTestComment("root", "", "2026-01-01T01:00:00Z", "Root body"),
		threadTestComment("reply", "root", "2026-01-02T01:00:00Z", "First *bold* paragraph\n\nSecond paragraph\n\n* One\n* Two\n\n{code}\ncode line\n{code}"),
	}, 1)
	var plain bytes.Buffer
	if err := issue.renderPlain(&plain); err != nil {
		t.Fatal(err)
	}
	renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle("notty"), glamour.WithWordWrap(wordWrap))
	if err != nil {
		t.Fatal(err)
	}
	want, err := issue.RenderedOut(renderer)
	if err != nil {
		t.Fatal(err)
	}
	if plain.String() != want {
		t.Fatal("plain rendering bypassed the shared fragment renderer")
	}
}

func TestCommentThreadLatestAndFooter(t *testing.T) {
	t.Parallel()
	comments := []jira.IssueComment{
		threadTestComment("old", "", "2026-01-01T01:00:00Z", "Old root"),
		threadTestComment("latest", "old", "2026-01-03T01:00:00Z", "Newest reply on hidden thread"),
		threadTestComment("new", "", "2026-01-02T01:00:00Z", "Newest root"),
	}
	issue := newThreadTestIssue(comments, 1)
	if strings.Contains(issue.String(), "Latest comment") || strings.Contains(issue.String(), "Old root") {
		t.Fatal("newest root was incorrectly labeled latest comment or older thread leaked")
	}
	if footer := issue.footer(); !strings.Contains(footer, "Showing 1 of 2 threads") || !strings.Contains(footer, "load more threads") || strings.Contains(footer, "load more comments") {
		t.Errorf("thread footer = %q", footer)
	}
	issue.Options.NumComments = 0
	if strings.Contains(issue.String(), "Newest root") || strings.Contains(issue.String(), " Comments ---") {
		t.Fatal("zero comments displayed comment content")
	}
	issue.Options.NumComments = ^uint(0)
	bodies := issue.comments()
	if strings.Contains(bodies[0].meta, "Latest comment") || !strings.Contains(bodies[2].meta, "Latest comment") {
		t.Errorf("latest-comment marker is not on the newest reply: %+v", bodies)
	}
	if !strings.Contains(issue.footer(), "Showing 2 of 2 threads") || strings.Contains(issue.footer(), "load more") {
		t.Errorf("unbounded footer = %q", issue.footer())
	}
	comments[0].Created = "not a timestamp"
	if text := issue.String(); !strings.Contains(text, "Invalid comment timestamp") || strings.Contains(text, "Latest comment") {
		t.Errorf("unknown dates were not labeled or latest claim was invented: %q", text)
	}
}

func TestCommentThreadRichADF(t *testing.T) {
	t.Parallel()
	const source = `{"version":1,"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"Bold paragraph","marks":[{"type":"strong"}]}]},
		{"type":"paragraph","content":[{"type":"text","text":"Second paragraph"}]},
		{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Bullet item"}]}]}]},
		{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"firstCodeLine()\nsecondCodeLine()"}]}
	]}`
	var doc adf.ADF
	if err := json.Unmarshal([]byte(source), &doc); err != nil {
		t.Fatal(err)
	}
	issue := newThreadTestIssue([]jira.IssueComment{
		threadTestComment("root", "", "2026-01-01T01:00:00Z", "Root"),
		threadTestComment("reply", "root", "2026-01-02T01:00:00Z", &doc),
	}, 1)
	body := issue.comments()[1].body
	for _, want := range []string{"**Bold paragraph**", "Second paragraph", "- Bullet item", "```go\nfirstCodeLine()\nsecondCodeLine()"} {
		if !strings.Contains(body, want) {
			t.Errorf("ADF translation lost formatting %q: %q", want, body)
		}
	}
	renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle("notty"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := issue.RenderedOut(renderer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n    • Bullet item") || !strings.Contains(out, "\n      firstCodeLine()") || !strings.Contains(out, "\n      secondCodeLine()") {
		t.Errorf("nested ADF was treated as raw text/code instead of Markdown: %q", out)
	}
}

func TestCommentThreadMalformedBodies(t *testing.T) {
	t.Parallel()
	for _, body := range []any{nil, (*adf.ADF)(nil), 42, map[string]any{"unexpected": "body"}} {
		issue := newThreadTestIssue([]jira.IssueComment{threadTestComment("root", "", "2026-01-01T01:00:00Z", body)}, 1)
		text := issue.String()
		if !strings.Contains(text, "Comment body unavailable") && !strings.Contains(text, "Unsupported comment body:") {
			t.Errorf("malformed body silently disappeared: %q", text)
		}
	}
}

func TestCommentThreadDeepChain(t *testing.T) {
	t.Parallel()
	const length = 1000
	comments := make([]jira.IssueComment, length)
	for idx := range comments {
		parent := ""
		if idx > 0 {
			parent = fmt.Sprint(idx - 1)
		}
		comments[idx] = threadTestComment(fmt.Sprint(idx), parent, "2026-01-01T01:00:00Z", "Body")
	}
	issue := newThreadTestIssue(comments, 1)
	got := issue.comments()
	if len(got) != length || got[length-1].depth != length-1 || !strings.Contains(got[length-1].meta, "depth 999") {
		t.Fatalf("deep thread was truncated or depth lost: %d entries", len(got))
	}
	if want := strings.Repeat("  ", maxCommentIndentDepth) + "Body"; indentComment("Body", length) != want {
		t.Fatal("deep nesting produced unbounded indentation")
	}
}

func TestFlatCommentTotalAndUintBounds(t *testing.T) {
	t.Parallel()
	issue := newThreadTestIssue([]jira.IssueComment{threadTestComment("root", "", "2026-01-01T01:00:00Z", "Flat body")}, ^uint(0))
	issue.Options.CommentThreads = false
	issue.Data.Fields.Comment.Total = 1000
	if got := issue.comments(); len(got) != 1 || !strings.Contains(got[0].body, "Flat body") || strings.Contains(got[0].meta, "Latest comment") {
		t.Fatalf("unsafe partial flat comments: %+v", got)
	}
	if strings.Contains(issue.footer(), "load more") {
		t.Fatal("maximum uint overflowed into a limited flat count")
	}
	issue.Data.Fields.Comment.Total = -1
	if got := issue.comments(); len(got) != 0 {
		t.Errorf("negative total returned comments: %+v", got)
	}
}

func TestReplyTargetsSelection(t *testing.T) {
	t.Parallel()
	comments := []jira.IssueComment{
		threadTestComment("old", "", "2026-01-01T01:00:00Z", "Old root"),
		threadTestComment("latest", "old", "2026-01-04T01:00:00Z", "Latest reply"),
		threadTestComment("b", "", "2026-01-02T01:00:00Z", "Root B"),
		threadTestComment("a", "", "2026-01-02T01:00:00Z", "Root A"),
		threadTestComment("reply", "a", "2026-01-03T01:00:00Z", "Reply to A"),
	}
	for _, tc := range []struct {
		name     string
		comments []jira.IssueComment
		threads  bool
		limit    uint
		ids      []string
		reason   string
	}{
		{name: "flat", comments: comments, limit: 2, reason: "--comment-threads"},
		{name: "zero", comments: comments, threads: true, reason: "greater than 0"},
		{name: "empty", threads: true, limit: 1, reason: "No valid native root threads"},
		{name: "one", comments: comments, threads: true, limit: 1, ids: []string{"a"}},
		{name: "two", comments: comments, threads: true, limit: 2, ids: []string{"a", "b"}},
		{name: "all", comments: comments, threads: true, limit: ^uint(0), ids: []string{"a", "b", "old"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			issue := newThreadTestIssue(tc.comments, tc.limit)
			issue.Options.CommentThreads = tc.threads
			targets, unavailable := issue.replyTargets()
			var ids []string
			for _, target := range targets {
				ids = append(ids, target.ID)
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Errorf("selectable roots = %v, want %v", ids, tc.ids)
			}
			if tc.reason == "" {
				if unavailable != "" {
					t.Errorf("valid roots marked unavailable: %q", unavailable)
				}
			} else if !strings.Contains(unavailable, tc.reason) {
				t.Errorf("availability guidance = %q, want %q", unavailable, tc.reason)
			}
		})
	}
}

func TestReplyTargetsRejectInvalidGraphs(t *testing.T) {
	t.Parallel()
	invalid := []jira.IssueComment{
		{ID: "orphan", ParentID: "absent"},
		{ID: "orphan-child", ParentID: "orphan"},
		{ID: "cycle-a", ParentID: "cycle-b"},
		{ID: "cycle-b", ParentID: "cycle-a"},
		{ID: "self", ParentID: "self"},
		{ID: "duplicate"},
		{ID: "duplicate"},
		{ID: "ambiguous-child", ParentID: "duplicate"},
		{},
	}
	issue := newThreadTestIssue(invalid, ^uint(0))
	targets, unavailable := issue.replyTargets()
	if len(targets) != 0 || !strings.Contains(unavailable, "No valid native root threads") {
		t.Fatalf("invalid graph is selectable: targets %+v, reason %q", targets, unavailable)
	}
	issue.Data.Fields.Comment.Comments = append(append([]jira.IssueComment(nil), invalid...),
		threadTestComment("valid", "", "2026-01-01T01:00:00Z", "Valid root"),
		threadTestComment("valid-reply", "valid", "2026-01-02T01:00:00Z", "Valid reply"))
	targets, unavailable = issue.replyTargets()
	if len(targets) != 1 || targets[0].ID != "valid" || unavailable != "" {
		t.Errorf("invalid entries leaked into the valid-root picker: targets %+v, reason %q", targets, unavailable)
	}
}

func TestReplyTargetsMetadataAndPreview(t *testing.T) {
	t.Parallel()
	doc := &adf.ADF{
		Version: 1, DocType: "doc",
		Content: []*adf.Node{{NodeType: "paragraph", Content: []*adf.Node{{NodeType: "text", NodeValue: adf.NodeValue{Text: "ADF first paragraph"}}}}},
	}
	for _, tc := range []struct {
		name    string
		author  jira.User
		body    any
		preview string
	}{
		{name: "display name", author: jira.User{DisplayName: "Display author", Name: "account-name"}, body: "\n\nFirst *bold* line\n\nSecond\tparagraph", preview: "First **bold** line Second paragraph"},
		{name: "account fallback", author: jira.User{Name: "account-name"}, body: doc, preview: "ADF first paragraph"},
		{name: "empty", body: " \n\t ", preview: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			comment := threadTestComment("root", "", "2026-01-01T01:00:00.000+0000", tc.body)
			comment.Author = tc.author
			issue := newThreadTestIssue([]jira.IssueComment{comment}, 1)
			targets, reason := issue.replyTargets()
			if len(targets) != 1 || reason != "" {
				t.Fatalf("missing valid target: %+v, %q", targets, reason)
			}
			author := tc.author.DisplayName
			if author == "" {
				author = tc.author.Name
			}
			if target := targets[0]; target.ID != "root" || target.Author != author || target.Created != "Thu, 01 Jan 26" || target.Preview != tc.preview {
				t.Errorf("target metadata/preview = %+v", target)
			}
		})
	}
}

func TestReplyTargetPreviewUTF8(t *testing.T) {
	t.Parallel()
	const limit = 100
	for _, count := range []int{limit - 1, limit, limit + 1} {
		body := strings.Repeat("界", count)
		issue := newThreadTestIssue([]jira.IssueComment{threadTestComment("root", "", "2026-01-01T01:00:00Z", body)}, 1)
		targets, _ := issue.replyTargets()
		preview := targets[0].Preview
		want := body
		if count > limit {
			want = strings.Repeat("界", limit-1) + "…"
		}
		if !utf8.ValidString(preview) || utf8.RuneCountInString(preview) > limit || preview != want {
			t.Errorf("unsafe/unexpected preview at %d runes: %q", count, preview)
		}
	}
}

func threadTestComment(id, parent, created string, body any) jira.IssueComment {
	return jira.IssueComment{ID: id, ParentID: parent, Created: created, Body: body, Author: jira.User{DisplayName: "Tester"}}
}

func newThreadTestIssue(comments []jira.IssueComment, limit uint) Issue {
	data := &jira.Issue{Key: "TEST-1"}
	data.Fields.Comment.Comments = comments
	data.Fields.Comment.Total = len(comments)
	return Issue{Server: "https://test.invalid", Data: data, Display: DisplayFormat{Plain: true}, Options: IssueOption{NumComments: limit, CommentThreads: true}}
}
