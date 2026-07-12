package md

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	cf "github.com/kentaro-m/blackfriday-confluence"
	bf "github.com/russross/blackfriday/v2"

	"github.com/rethab/jira-cli/pkg/md/jirawiki"
)

// mentionPattern matches Jira user mentions, e.g. [~displayname] or [~display name].
var mentionPattern = regexp.MustCompile(`\[~[^\[\]\n]+\]`)

// nonceSize is the number of random bytes used to build a mention placeholder nonce.
const nonceSize = 8

// ToJiraMD translates CommonMark to Jira flavored markdown.
func ToJiraMD(md string) string {
	if md == "" {
		return md
	}

	// blackfriday-confluence's escaper backslash-escapes `[`, `~` and `]`,
	// which mangles a Jira mention like "[~displayname]" into
	// "\[\~displayname\]" and stops it from pinging the user. Swap mentions
	// out for placeholders that survive the render untouched, then restore
	// them afterwards.
	nonce := placeholderNonce()
	var mentions []string
	protected := mentionPattern.ReplaceAllStringFunc(md, func(mention string) string {
		mentions = append(mentions, mention)
		return placeholder(nonce, len(mentions)-1)
	})

	renderer := &cf.Renderer{Flags: cf.IgnoreMacroEscaping}
	r := bf.New(bf.WithRenderer(renderer), bf.WithExtensions(bf.CommonExtensions))

	var buf bytes.Buffer
	ast := r.Parse([]byte(protected))
	ast.Walk(func(node *bf.Node, entering bool) bf.WalkStatus {
		// Jira wiki image syntax (!url!) has no place for alt text, so the
		// image node's children (the alt text) must be skipped, otherwise
		// they get concatenated into the same !...! markers by the renderer.
		// SkipChildren also suppresses the walker's "leaving" visit for this
		// node, so both markers are emitted up front while entering. They go
		// through RenderNode rather than the buffer directly because the
		// renderer tracks how much it last wrote to decide whether a
		// following block needs a leading newline; writing behind its back
		// leaves that stale and glues the next table or list onto this line.
		if node.Type == bf.Image {
			if entering {
				renderer.RenderNode(&buf, node, true)
				renderer.RenderNode(&buf, node, false)
			}
			return bf.SkipChildren
		}
		return renderer.RenderNode(&buf, node, entering)
	})

	out := buf.String()
	for i, mention := range mentions {
		out = strings.ReplaceAll(out, placeholder(nonce, i), mention)
	}

	return out
}

// placeholderNonce returns a random token used to make mention placeholders
// practically impossible to collide with real input text.
func placeholderNonce() string {
	b := make([]byte, nonceSize)
	// Since Go 1.24, crypto/rand.Read never returns an error: it panics on
	// unrecoverable OS failure instead.
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}

// placeholder returns a marker that is unlikely to appear in real markdown
// and contains no characters that the confluence renderer would escape. The
// nonce also terminates the marker so that no placeholder is a prefix of
// another: otherwise restoring "<nonce>1" would corrupt "<nonce>10".
func placeholder(nonce string, i int) string {
	return fmt.Sprintf("%s%d%s", nonce, i, nonce)
}

// FromJiraMD translates Jira flavored markdown to CommonMark.
func FromJiraMD(jfm string) string {
	return jirawiki.Parse(jfm)
}
