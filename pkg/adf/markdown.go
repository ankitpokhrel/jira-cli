package adf

import (
	"fmt"
	"strings"
)

// MarkdownTranslator is a markdown translator.
type MarkdownTranslator struct {
	table struct {
		rows int
		cols int
		ccol int // current column count
		sep  bool
	}
	list struct {
		ol, ul  bool
		depth   int
		counter int // each level has same numeric counter at the moment.
	}
}

// Open implements TagOpener interface.
//nolint:gocyclo
func (tr *MarkdownTranslator) Open(n Connector, d int) string {
	var tag strings.Builder

	nt, attrs := n.GetType(), n.GetAttributes()

	switch nt {
	case NodeBlockquote:
		tag.WriteString("> ")
	case NodeCodeBlock:
		tag.WriteString("```")

		nl := true
		if attrs != nil {
			a := attrs.(map[string]interface{})
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
		tr.list.ol = false
		tr.list.ul = true
		tr.list.depth = d
	case NodeOrderedList:
		tr.list.ol = true
		tr.list.ul = false
		tr.list.depth = d
	case ChildNodeListItem:
		for i := 0; i < tr.list.depth/2; i++ {
			tag.WriteString("\t")
		}
		if tr.list.ol {
			tr.list.counter++
			tag.WriteString(fmt.Sprintf("%d. ", tr.list.counter))
		} else {
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
		tag.WriteString(" ~")
	case MarkLink:
		tag.WriteString(" [")
	}

	tag.WriteString(tr.setOpenTagAttributes(attrs))

	return tag.String()
}

// Close implements TagCloser interface.
//nolint:gocyclo
func (tr *MarkdownTranslator) Close(n Connector) string {
	var tag strings.Builder

	switch n.GetType() {
	case NodeBlockquote:
		tag.WriteString("\n")
	case NodeCodeBlock:
		tag.WriteString("\n```\n")
	case NodePanel:
		tag.WriteString("---\n")
	case NodeHeading:
		tag.WriteString("\n")
	case NodeBulletList:
		fallthrough
	case NodeOrderedList:
		tr.list.depth = 0
	case NodeParagraph:
		if tr.table.rows == 0 {
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
		tag.WriteString("~ ")
	case MarkLink:
		tag.WriteString("]")
	}

	tag.WriteString(tr.setCloseTagAttributes(n.GetAttributes()))

	return tag.String()
}

func (tr *MarkdownTranslator) setOpenTagAttributes(a interface{}) string {
	if a == nil {
		return ""
	}

	var (
		tag strings.Builder
		nl  bool
	)

	attrs := a.(map[string]interface{})
	for k, v := range attrs {
		if tr.isValidAttr(k) {
			switch k {
			case "language":
				tag.WriteString(fmt.Sprintf("%s", v))
				nl = true
			case "level":
				for i := 0; i < int(v.(float64)); i++ {
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

func (tr *MarkdownTranslator) setCloseTagAttributes(a interface{}) string {
	if a == nil {
		return ""
	}
	var tag strings.Builder
	attrs := a.(map[string]interface{})
	if h, ok := attrs["href"]; ok {
		tag.WriteString(fmt.Sprintf("(%s) ", h))
	} else if h, ok := attrs["url"]; ok {
		tag.WriteString(fmt.Sprintf("%s ", h))
	}
	return tag.String()
}

func (tr *MarkdownTranslator) isValidAttr(attr string) bool {
	known := []string{"language", "level", "text"}
	for _, k := range known {
		if k == attr {
			return true
		}
	}
	return false
}
