package adf

import (
	"fmt"
	"slices"
	"strings"
)

type nodeTypeHook map[NodeType]func(Connector) string

// MarkdownTranslator is a markdown translator.
type MarkdownTranslator struct {
	table struct {
		rows int
		cols int
		ccol int // current column count
		sep  bool
	}
	list struct {
		ol, ul  map[int]bool
		depthO  int
		depthU  int
		counter map[int]int // each level starts with same numeric counter at the moment.
	}
	openHooks  nodeTypeHook
	closeHooks nodeTypeHook
}

// MarkdownTranslatorOption is a functional option for MarkdownTranslator.
type MarkdownTranslatorOption func(*MarkdownTranslator)

// NewMarkdownTranslator constructs markdown translator.
func NewMarkdownTranslator(opts ...MarkdownTranslatorOption) *MarkdownTranslator {
	tr := MarkdownTranslator{
		list: struct {
			ol, ul  map[int]bool
			depthO  int
			depthU  int
			counter map[int]int
		}{
			ol:      make(map[int]bool),
			ul:      make(map[int]bool),
			counter: make(map[int]int),
		},
	}

	for _, opt := range opts {
		opt(&tr)
	}

	return &tr
}

// WithMarkdownOpenHooks sets open hooks of a markdown translator.
func WithMarkdownOpenHooks(hooks nodeTypeHook) MarkdownTranslatorOption {
	return func(tr *MarkdownTranslator) {
		tr.openHooks = hooks
	}
}

// WithMarkdownCloseHooks sets close hooks of a markdown translator.
func WithMarkdownCloseHooks(hooks nodeTypeHook) MarkdownTranslatorOption {
	return func(tr *MarkdownTranslator) {
		tr.closeHooks = hooks
	}
}

// Open implements TagOpener interface.
//
//nolint:gocyclo
func (tr *MarkdownTranslator) Open(n Connector, _ int) string {
	var tag strings.Builder

	nt, attrs := n.GetType(), n.GetAttributes()

	if hook, ok := tr.openHooks[nt]; ok {
		tag.WriteString(hook(n))
	} else {
		switch nt {
		case NodeBlockquote:
			tag.WriteString("> ")
		case NodeCodeBlock:
			tag.WriteString("```")

			nl := true
			if attrs != nil {
				a := attrs.(map[string]any)
				for k := range a {
					if k == "language" {
						nl = false
						break
					}
				}
			}
			if nl {
				tag.WriteString("\n")
			}
		case NodePanel:
			tag.WriteString("---\n")
		case NodeTable:
			tag.WriteString("\n")
		case NodeMedia:
			tag.WriteString("\n[attachment]")
		case NodeBulletList:
			tr.list.depthU++
			tr.list.ul[tr.list.depthU] = true
		case NodeOrderedList:
			tr.list.depthO++
			tr.list.ol[tr.list.depthO] = true
		case ChildNodeListItem:
			if tr.list.ol[tr.list.depthO] {
				for range tr.list.depthO - 1 {
					tag.WriteString("\t")
				}
				tr.list.counter[tr.list.depthO]++
				tag.WriteString(fmt.Sprintf("%d. ", tr.list.counter[tr.list.depthO]))
			} else {
				for range tr.list.depthU - 1 {
					tag.WriteString("\t")
				}
				tag.WriteString("- ")
			}
		case ChildNodeTableHeader:
			if tr.table.cols != 0 {
				tag.WriteString(" | ")
			}
			tr.table.cols++
		case ChildNodeTableCell:
			if tr.table.ccol != 0 {
				tag.WriteString(" | ")
			}
			tr.table.ccol++
		case ChildNodeTableRow:
			tr.table.rows++
			if tr.table.rows == 1 && !tr.table.sep {
				tr.table.sep = true
			}
			tr.table.ccol = 0
		case InlineNodeHardBreak:
			tag.WriteString("\n\n")
		case InlineNodeMention:
			tag.WriteString(" @")
		case InlineNodeCard:
			tag.WriteString(" 📍 ")
		case MarkStrong:
			tag.WriteString(" **")
		case MarkEm:
			tag.WriteString(" _")
		case MarkCode:
			tag.WriteString(" `")
		case MarkStrike:
			tag.WriteString(" -")
		case MarkLink:
			tag.WriteString(" [")
		}
	}

	tag.WriteString(tr.setOpenTagAttributes(attrs))

	return tag.String()
}

// Close implements TagCloser interface.
//
//nolint:gocyclo
func (tr *MarkdownTranslator) Close(n Connector) string {
	var tag strings.Builder

	nt := n.GetType()

	if hook, ok := tr.closeHooks[nt]; ok {
		tag.WriteString(hook(n))
	} else {
		switch nt {
		case NodeBlockquote:
			tag.WriteString("\n")
		case NodeCodeBlock:
			tag.WriteString("\n```\n")
		case NodePanel:
			tag.WriteString("---\n")
		case NodeHeading:
			tag.WriteString("\n")
		case NodeBulletList:
			tr.list.ul[tr.list.depthU] = false
			tr.list.depthU--
		case NodeOrderedList:
			tr.list.ol[tr.list.depthO] = false
			tr.list.depthO--
		case NodeParagraph:
			if tr.list.ul[tr.list.depthU] || tr.list.ol[tr.list.depthO] {
				tag.WriteString("\n")
			} else if tr.table.rows == 0 {
				tag.WriteString("\n\n")
			}
		case NodeTable:
			tr.table.rows = 0
			tr.table.cols = 0
			tr.table.sep = false
		case ChildNodeTableRow:
			tag.WriteString("\n")
			if tr.table.sep {
				for i := 0; i < tr.table.cols; i++ {
					tag.WriteString("---")
					if i != tr.table.cols-1 {
						tag.WriteString(" | ")
					}
				}
				tr.table.sep = false
				tag.WriteString("\n")
			}
		case InlineNodeMention:
			tag.WriteString(" ")
		case InlineNodeEmoji:
			tag.WriteString(" ")
		case MarkStrong:
			tag.WriteString("** ")
		case MarkEm:
			tag.WriteString("_ ")
		case MarkCode:
			tag.WriteString("` ")
		case MarkStrike:
			tag.WriteString("- ")
		case MarkLink:
			tag.WriteString("]")
		}
	}

	tag.WriteString(tr.setCloseTagAttributes(n.GetAttributes()))

	return tag.String()
}

func (tr *MarkdownTranslator) setOpenTagAttributes(a any) string {
	if a == nil {
		return ""
	}

	var (
		tag strings.Builder
		nl  bool
	)

	attrs := a.(map[string]any)
	for k, v := range attrs {
		if tr.isValidAttr(k) {
			switch k {
			case "language":
				tag.WriteString(fmt.Sprintf("%s", v))
				nl = true
			case "level":
				for range int(v.(float64)) {
					tag.WriteString("#")
				}
				tag.WriteString(" ")
			case "text":
				tag.WriteString(fmt.Sprintf("%s", v))
				nl = false
			}
		}
		if nl {
			tag.WriteString("\n")
		}
	}

	return tag.String()
}

func (*MarkdownTranslator) setCloseTagAttributes(a any) string {
	if a == nil {
		return ""
	}

	var tag strings.Builder

	attrs := a.(map[string]any)
	if h, ok := attrs["href"]; ok {
		tag.WriteString(fmt.Sprintf("(%s) ", h))
	} else if h, ok := attrs["url"]; ok {
		tag.WriteString(fmt.Sprintf("%s ", h))
	}

	return tag.String()
}

func (*MarkdownTranslator) isValidAttr(attr string) bool {
	known := []string{"language", "level", "text"}
	return slices.Contains(known, attr)
}
