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
		counter int // each level has same numeric counter at the moment.
	}
}

// Open implements TagOpener interface.
func (tr *MarkdownTranslator) Open(n Connector, d int) string {
	var tag strings.Builder

	nt := n.GetType()
	tag.WriteString(tr.levelUp(nt, d))

	switch nt {
	case NodeBlockquote:
		tag.WriteString("> ")
	case NodeBulletList:
		tr.list.ol = false
		tr.list.ul = true
	case NodeOrderedList:
		tr.list.ol = true
		tr.list.ul = false
	case ChildNodeListItem:
		if tr.list.ol {
			tr.list.counter++
			tag.WriteString(fmt.Sprintf("%d. ", tr.list.counter))
		} else {
			tag.WriteString("- ")
		}
	case NodeCodeBlock:
		tag.WriteString("```")
	case NodePanel:
		tag.WriteString("```\n")
	case NodeTable:
		tag.WriteString("\n")
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
	case MarkStrong:
		tag.WriteString("**")
	case MarkEm:
		tag.WriteString("_")
	case MarkStrike:
		tag.WriteString("~")
	case MarkLink:
		tag.WriteString("[")
	}

	tag.WriteString(tr.setOpenTagAttributes(n.GetAttributes()))

	return tag.String()
}

// Close implements TagCloser interface.
func (tr *MarkdownTranslator) Close(n Connector) string {
	var tag strings.Builder

	switch n.GetType() {
	case NodeBlockquote:
		tag.WriteString("\n")
	case NodeCodeBlock:
		tag.WriteString("```\n")
	case NodePanel:
		tag.WriteString("```\n")
	case NodeHeading:
		tag.WriteString("\n")
	case NodeBulletList:
		fallthrough
	case NodeOrderedList:
		tr.list.ol = false
		tr.list.ul = false
		tr.list.counter = 0
	case NodeParagraph:
		if tr.table.rows == 0 {
			tag.WriteString("\n")
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
	case MarkStrong:
		tag.WriteString("**")
	case MarkEm:
		tag.WriteString("_")
	case MarkStrike:
		tag.WriteString("~")
	case MarkLink:
		tag.WriteString("]")
	}

	tag.WriteString(tr.setCloseTagAttributes(n.GetAttributes()))

	return tag.String()
}

func (tr *MarkdownTranslator) levelUp(nt string, depth int) string {
	if !IsParentNode(nt) {
		return ""
	}
	var tag strings.Builder
	if nt == NodeBulletList || nt == NodeBlockquote || nt == NodeOrderedList {
		for i := 0; i < (depth / 2); i++ {
			tag.WriteString("\t")
		}
	}
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
			}
		}
	}
	if nl {
		tag.WriteString("\n")
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
		tag.WriteString(fmt.Sprintf("(%s)", h))
	}
	return tag.String()
}

func (tr *MarkdownTranslator) isValidAttr(attr string) bool {
	known := []string{"language", "level"}
	for _, k := range known {
		if k == attr {
			return true
		}
	}
	return false
}
