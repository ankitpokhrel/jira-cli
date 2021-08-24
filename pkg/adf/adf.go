package adf

import (
	"strings"
)

// Node types.
const (
	NodeTypeParent  = "parent"
	NodeTypeChild   = "child"
	NodeTypeUnknown = "unknown"

	NodeBlockquote  = "blockquote"
	NodeBulletList  = "bulletList"
	NodeCodeBlock   = "codeBlock"
	NodeHeading     = "heading"
	NodeOrderedList = "orderedList"
	NodePanel       = "panel"
	NodeParagraph   = "paragraph"
	NodeTable       = "table"
	NodeMedia       = "media"

	ChildNodeText        = "text"
	ChildNodeListItem    = "listItem"
	ChildNodeTableRow    = "tableRow"
	ChildNodeTableHeader = "tableHeader"
	ChildNodeTableCell   = "tableCell"

	InlineNodeCard      = "inlineCard"
	InlineNodeEmoji     = "emoji"
	InlineNodeMention   = "mention"
	InlineNodeHardBreak = "hardBreak"

	MarkEm     = "em"
	MarkLink   = "link"
	MarkCode   = "code"
	MarkStrike = "strike"
	MarkStrong = "strong"
)

// TagOpener is a tag opener.
type TagOpener interface {
	Open(c Connector, depth int) string
}

// TagCloser is a tag closer.
type TagCloser interface {
	Close(Connector) string
}

// TagOpenerCloser wraps tag opener and closer.
type TagOpenerCloser interface {
	TagOpener
	TagCloser
}

// Connector is a connector interface.
type Connector interface {
	GetType() string
	GetAttributes() interface{}
}

// ADF is an Atlassian document format object.
type ADF struct {
	Version int     `json:"version"`
	DocType string  `json:"type"`
	Content []*Node `json:"content"`
}

// ReplaceAll replaces all occurrences of an old string with a new.
func (a *ADF) ReplaceAll(old, new string) {
	if a == nil || len(a.Content) == 0 {
		return
	}
	for _, parent := range a.Content {
		a.replace(parent, 0, old, new)
	}
}

func (a *ADF) replace(n *Node, depth int, old, new string) {
	for _, child := range n.Content {
		a.replace(child, depth+1, old, new)
	}
	if n.NodeType == ChildNodeText {
		n.Text = strings.ReplaceAll(n.Text, old, new)
	}
}

// Node is an ADF content node.
type Node struct {
	NodeType   string      `json:"type"`
	Content    []*Node     `json:"content,omitempty"`
	Attributes interface{} `json:"attrs,omitempty"`
	NodeValue
}

// GetType gets node type.
func (n Node) GetType() string { return n.NodeType }

// GetAttributes gets node attributes.
func (n Node) GetAttributes() interface{} { return n.Attributes }

// NodeValue is an actual ADF node content.
type NodeValue struct {
	Text  string     `json:"text,omitempty"`
	Marks []MarkNode `json:"marks,omitempty"`
}

// MarkNode is a mark node type.
type MarkNode struct {
	MarkType   string      `json:"type,omitempty"`
	Attributes interface{} `json:"attrs,omitempty"`
}

// GetType gets node type.
func (n MarkNode) GetType() string { return n.MarkType }

// GetAttributes gets node attributes.
func (n MarkNode) GetAttributes() interface{} { return n.Attributes }

// Translator transforms ADF to a new format.
type Translator struct {
	doc *ADF
	tsl TagOpenerCloser
	buf *strings.Builder
}

// NewTranslator constructs an ADF translator.
func NewTranslator(adf *ADF, tr TagOpenerCloser) *Translator {
	return &Translator{
		doc: adf,
		tsl: tr,
		buf: new(strings.Builder),
	}
}

// ParentNodes returns supported ADF parent nodes.
func ParentNodes() []string {
	return []string{
		NodeBlockquote,
		NodeBulletList,
		NodeCodeBlock,
		NodeHeading,
		NodeOrderedList,
		NodePanel,
		NodeParagraph,
		NodeTable,
		NodeMedia,
	}
}

// ChildNodes returns supported ADF child nodes.
func ChildNodes() []string {
	return []string{
		ChildNodeText,
		ChildNodeListItem,
		ChildNodeTableRow,
		ChildNodeTableHeader,
		ChildNodeTableCell,
	}
}

// IsParentNode checks if the node is a parent node.
func IsParentNode(identifier string) bool {
	for _, n := range ParentNodes() {
		if n == identifier {
			return true
		}
	}
	return false
}

// IsChildNode checks if the node is a child node.
func IsChildNode(identifier string) bool {
	for _, n := range ChildNodes() {
		if n == identifier {
			return true
		}
	}
	return false
}

// NodeType returns the type of ADF node.
func NodeType(identifier string) string {
	if IsParentNode(identifier) {
		return NodeTypeParent
	}
	if IsChildNode(identifier) {
		return NodeTypeChild
	}
	return NodeTypeUnknown
}

// Translate translates ADF to a new format.
func (a *Translator) Translate() string {
	a.walk()
	return a.buf.String()
}

func (a *Translator) walk() {
	if a.doc == nil || len(a.doc.Content) == 0 {
		return
	}
	for _, parent := range a.doc.Content {
		a.visit(parent, 0)
	}
}

func (a *Translator) visit(n *Node, depth int) {
	a.buf.WriteString(a.tsl.Open(n, depth))

	for _, child := range n.Content {
		a.visit(child, depth+1)
	}

	if NodeType(n.NodeType) == NodeTypeChild {
		var tag strings.Builder

		opened := make([]MarkNode, 0, len(n.Marks))
		if n.NodeType == ChildNodeText {
			for _, m := range n.Marks {
				opened = append(opened, m)
				tag.WriteString(a.tsl.Open(m, depth))
			}
		}

		tag.WriteString(sanitize(n.Text))

		// Close tags in reverse order.
		for i := len(opened) - 1; i >= 0; i-- {
			m := opened[i]
			tag.WriteString(a.tsl.Close(m))
		}

		a.buf.WriteString(tag.String())
	}

	a.buf.WriteString(a.tsl.Close(n))
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "\n")
	s = strings.ReplaceAll(s, "<", "❬")
	s = strings.ReplaceAll(s, ">", "❭")
	return s
}
