package md

import (
	"errors"
	"fmt"
	"html"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	bf "github.com/russross/blackfriday/v2"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

// BlackFriday silently truncates at nesting 16; reject before reaching it.
const maxADFDepth = 16

var (
	adfReference     = regexp.MustCompile(`(?m)^([ \t>]*(?:(?:[-+*]|[0-9]+\.)[ \t]+)?\[[^\]\r\n]+\]):`)
	adfOrdered       = regexp.MustCompile(`(?m)^([ \t>]*(?:[-+*][ \t]+)?)([0-9]+)\.([ \t]+)([^\n]*)`)
	adfOrderTag      = regexp.MustCompile("\x00([0-9]+)\x00")
	adfEntity        = regexp.MustCompile(`&(?:#[0-9]+|#[xX][0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);`)
	adfRowTag        = regexp.MustCompile("\x00row[0-9]+\x00")
	adfEndEscape     = regexp.MustCompile(`(\\[ \t]*)(\n[ \t]*(?:\n|\z)|\z)`)
	adfUnparsedFence = regexp.MustCompile("(?m)^[ \\t]*(?:`{3,}|~{3,})")
	adfTitleEscape   = regexp.MustCompile(`\\[[:punct:]]|` + adfEntity.String())
)

// ToADF preserves composer line breaks while producing native Jira document content.
func ToADF(markdown string) (*adf.ADF, error) {
	if !utf8.ValidString(markdown) {
		return nil, errors.New("markdown to ADF: invalid UTF-8")
	}
	if strings.ContainsRune(markdown, 0) {
		return nil, errors.New("markdown to ADF: unsupported NUL character")
	}

	// References disappear from BlackFriday's AST, including unused definitions.
	// Protect their delimiters so unsupported definitions cannot silently vanish.
	markdown = strings.ReplaceAll(strings.ReplaceAll(markdown, "\r\n", "\n"), "\r", "\n")
	input := adfReference.ReplaceAllString(markdown, "$1\x00ref\x00:")
	// BlackFriday otherwise consumes an unpaired backslash at paragraph end.
	input = adfEndEscape.ReplaceAllString(input, "$1\x00end\x00$2")
	root := parseADFMarkdown(input)
	orders, err := adfListOrders(input, root)
	if err != nil {
		return nil, err
	}
	content, err := adfBlocks(root, orders, 0)
	if err != nil {
		return nil, err
	}
	return &adf.ADF{Version: 1, DocType: "doc", Content: content}, nil
}

func parseADFMarkdown(input string) *bf.Node {
	return bf.New(bf.WithExtensions(bf.CommonExtensions | bf.Footnotes)).Parse([]byte(input))
}

func adfListOrders(input string, root *bf.Node) (map[*bf.Node]int, error) {
	var lists []*bf.Node
	var hasTables bool
	var depth int
	var walkErr error
	root.Walk(func(n *bf.Node, entering bool) bf.WalkStatus {
		if !entering {
			depth--
			return bf.GoToNext
		}
		if depth >= maxADFDepth {
			walkErr = fmt.Errorf("markdown to ADF: %s exceeds supported nesting depth", n.Type)
			return bf.Terminate
		}
		if n.IsContainer() {
			depth++
		}
		if n.Type == bf.Table {
			hasTables = true
		}
		if n.IsFootnotesList {
			walkErr = errors.New("markdown to ADF: unsupported footnote List")
			return bf.Terminate
		}
		if n.Type == bf.List && n.ListFlags&bf.ListTypeOrdered != 0 {
			lists = append(lists, n)
		}
		return bf.GoToNext
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if hasTables {
		if err := adfCheckTableRows(input); err != nil {
			return nil, err
		}
	}
	orders := make(map[*bf.Node]int, len(lists))
	if len(lists) == 0 {
		return orders, nil
	}

	// Only this metadata parse is annotated: the content AST is never modified.
	// A suffix keeps fences/headings at the start of list items parseable.
	tagged := adfOrdered.ReplaceAllStringFunc(input, func(line string) string {
		return adfTagLine(line, "\x00"+adfOrdered.FindStringSubmatch(line)[2]+"\x00")
	})
	index := 0
	parseADFMarkdown(tagged).Walk(func(n *bf.Node, entering bool) bf.WalkStatus {
		if !entering || n.Type != bf.List || n.ListFlags&bf.ListTypeOrdered == 0 {
			return bf.GoToNext
		}
		if index >= len(lists) {
			walkErr = errors.New("markdown to ADF: unsupported ordered List structure")
			return bf.Terminate
		}
		order, err := adfTaggedOrder(n.FirstChild)
		if err != nil {
			walkErr = err
			return bf.Terminate
		}
		orders[lists[index]] = order
		index++
		return bf.GoToNext
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if index != len(lists) {
		return nil, errors.New("markdown to ADF: unsupported ordered List structure")
	}
	return orders, nil
}

func adfCheckTableRows(input string) error {
	// BlackFriday discards excess cells. A tag in the final cell must survive.
	count := 0
	lines := strings.Split(input, "\n")
	inTable, previousPipe := false, false
	for i, line := range lines {
		if !strings.Contains(line, "|") {
			inTable, previousPipe = false, false
			continue
		}
		if !inTable && previousPipe && strings.Trim(line, " \t>|:-") == "" {
			inTable = true
			continue
		}
		previousPipe = true
		count++
		lines[i] = adfTagLine(line, fmt.Sprintf("\x00row%d\x00", count))
	}
	tagged := strings.Join(lines, "\n")
	seen := make(map[string]bool, count)
	parseADFMarkdown(tagged).Walk(func(n *bf.Node, entering bool) bf.WalkStatus {
		if entering {
			for _, value := range []string{string(n.Info), n.HeadingID, string(n.Literal), string(n.Destination), string(n.Title)} {
				for _, tag := range adfRowTag.FindAllString(value, -1) {
					seen[tag] = true
				}
			}
		}
		return bf.GoToNext
	})
	if len(seen) != count {
		return errors.New("markdown to ADF: unsupported Table row with discarded content")
	}
	return nil
}

func adfTagLine(line, tag string) string {
	end := len(strings.TrimRight(line, " \t"))
	if end > 0 && line[end-1] == '|' {
		backslashes := 0
		for i := end - 2; i >= 0 && line[i] == '\\'; i-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			end--
		}
	}
	return line[:end] + " " + tag + line[end:]
}

func adfTaggedOrder(item *bf.Node) (int, error) {
	var tag []string
	item.Walk(func(n *bf.Node, entering bool) bf.WalkStatus {
		if !entering {
			return bf.GoToNext
		}
		if n.Type == bf.List {
			return bf.SkipChildren
		}
		for _, value := range []string{string(n.Info), n.HeadingID, string(n.Literal), string(n.Destination), string(n.Title)} {
			if tag = adfOrderTag.FindStringSubmatch(value); tag != nil {
				return bf.Terminate
			}
		}
		return bf.GoToNext
	})
	if tag == nil {
		return 0, errors.New("markdown to ADF: cannot preserve ordered List start")
	}
	order, err := strconv.ParseInt(tag[1], 10, 32)
	if err != nil {
		return 0, errors.New("markdown to ADF: ordered List start exceeds supported range")
	}
	return int(order), nil
}

func adfBlocks(parent *bf.Node, orders map[*bf.Node]int, depth int) ([]*adf.Node, error) {
	content := make([]*adf.Node, 0)
	for n := parent.FirstChild; n != nil; n = n.Next {
		if depth >= maxADFDepth {
			return nil, fmt.Errorf("markdown to ADF: %s exceeds supported nesting depth", n.Type)
		}
		node, err := adfBlock(n, orders, depth+1)
		if err != nil {
			return nil, err
		}
		if parent.Type == bf.Item && node.NodeType != adf.NodeParagraph && node.NodeType != adf.NodeBulletList && node.NodeType != adf.NodeOrderedList && node.NodeType != adf.NodeCodeBlock {
			return nil, fmt.Errorf("markdown to ADF: unsupported %s inside list Item", n.Type)
		}
		if parent.Type == bf.BlockQuote {
			switch node.NodeType {
			case adf.NodeParagraph, adf.NodeBulletList, adf.NodeOrderedList, adf.NodeCodeBlock:
			default:
				return nil, fmt.Errorf("markdown to ADF: unsupported %s inside BlockQuote", n.Type)
			}
		}
		content = append(content, node)
	}
	return content, nil
}

func adfBlock(n *bf.Node, orders map[*bf.Node]int, depth int) (*adf.Node, error) {
	node := &adf.Node{}
	var err error
	switch n.Type {
	case bf.Paragraph, bf.Heading:
		node.NodeType = adf.NodeParagraph
		if n.Type == bf.Heading {
			if n.HeadingID != "" {
				return nil, errors.New("markdown to ADF: unsupported Heading ID")
			}
			node.NodeType = adf.NodeHeading
			node.Attributes = map[string]any{"level": n.Level}
		}
		node.Content, err = adfInlines(n, nil, depth)
	case bf.BlockQuote, bf.List, bf.Item:
		return adfContainer(n, orders, depth)
	case bf.CodeBlock:
		node.NodeType = adf.NodeCodeBlock
		info := strings.TrimSpace(adfRestoreCode(string(n.Info)))
		if len(strings.Fields(info)) > 1 {
			return nil, errors.New("markdown to ADF: unsupported CodeBlock fence metadata")
		}
		if info != "" {
			node.Attributes = map[string]any{"language": info}
		}
		node.Content = adfCodeText(adfRestoreCode(string(n.Literal)), nil)
	case bf.HorizontalRule:
		node.NodeType = adf.NodeType("rule")
	case bf.Table:
		node.NodeType = adf.NodeTable
		for section := n.FirstChild; section != nil; section = section.Next {
			rows, rowsErr := adfBlocks(section, orders, depth)
			if rowsErr != nil {
				return nil, rowsErr
			}
			node.Content = append(node.Content, rows...)
		}
	case bf.TableRow:
		node.NodeType = adf.ChildNodeTableRow
		node.Content, err = adfBlocks(n, orders, depth)
	case bf.TableCell:
		if n.Align != 0 {
			return nil, errors.New("markdown to ADF: unsupported TableCell alignment")
		}
		node.NodeType = adf.ChildNodeTableCell
		if n.IsHeader {
			node.NodeType = adf.ChildNodeTableHeader
		}
		var inlines []*adf.Node
		inlines, err = adfInlines(n, nil, depth)
		node.Content = []*adf.Node{{NodeType: adf.NodeParagraph, Content: inlines}}
	default:
		return nil, fmt.Errorf("markdown to ADF: unsupported %s", n.Type)
	}
	if err != nil {
		return nil, err
	}
	return node, nil
}

func adfContainer(n *bf.Node, orders map[*bf.Node]int, depth int) (*adf.Node, error) {
	node := &adf.Node{}
	switch n.Type {
	case bf.BlockQuote:
		node.NodeType = adf.NodeBlockquote
	case bf.List:
		if n.ListFlags&bf.ListTypeDefinition != 0 {
			return nil, errors.New("markdown to ADF: unsupported definition List")
		}
		node.NodeType = adf.NodeBulletList
		if n.ListFlags&bf.ListTypeOrdered != 0 {
			node.NodeType = adf.NodeOrderedList
			node.Attributes = map[string]any{"order": orders[n]}
		}
	case bf.Item:
		node.NodeType = adf.ChildNodeListItem
	}
	var err error
	node.Content, err = adfBlocks(n, orders, depth)
	if err != nil {
		return nil, err
	}
	if n.Type == bf.Item && (len(node.Content) == 0 || node.Content[0].NodeType != adf.NodeParagraph) {
		node.Content = append([]*adf.Node{{NodeType: adf.NodeParagraph}}, node.Content...)
	}
	if n.Type == bf.BlockQuote && len(node.Content) == 0 {
		node.Content = []*adf.Node{{NodeType: adf.NodeParagraph}}
	}
	return node, nil
}

func adfInlines(parent *bf.Node, marks []adf.MarkNode, depth int) ([]*adf.Node, error) {
	if depth >= maxADFDepth {
		return nil, fmt.Errorf("markdown to ADF: %s exceeds supported nesting depth", parent.Type)
	}
	var content []*adf.Node
	for n := parent.FirstChild; n != nil; n = n.Next {
		var nodes []*adf.Node
		switch n.Type {
		case bf.Text:
			text := string(n.Literal)
			if adfUnparsedFence.MatchString(text) {
				return nil, errors.New("markdown to ADF: unsupported CodeBlock fence in inline content")
			}
			if strings.Contains(text, "\x00ref\x00") {
				return nil, errors.New("markdown to ADF: unsupported reference/footnote definition")
			}
			if strings.Contains(text, "[^") || strings.Contains(text, "^[") {
				return nil, errors.New("markdown to ADF: unsupported footnote format")
			}
			nodes = adfText(adfUnescape(strings.ReplaceAll(text, "\x00end\x00", "")), marks)
		case bf.Softbreak, bf.Hardbreak:
			nodes = []*adf.Node{{NodeType: adf.InlineNodeHardBreak}}
		case bf.Code:
			if strings.ContainsRune(string(n.Literal), '\n') {
				return nil, errors.New("markdown to ADF: unsupported multiline inline Code")
			}
			for _, mark := range marks {
				if mark.MarkType != adf.MarkLink {
					return nil, errors.New("markdown to ADF: unsupported Code combined with formatting marks")
				}
			}
			nodes = adfCodeText(adfRestoreCode(string(n.Literal)), append(slices.Clone(marks), adf.MarkNode{MarkType: adf.MarkCode}))
		case bf.Emph, bf.Strong, bf.Del, bf.Link:
			mark, err := adfMark(n)
			if err != nil {
				return nil, err
			}
			childMarks := slices.Clone(marks)
			if !slices.ContainsFunc(marks, func(existing adf.MarkNode) bool { return existing.MarkType == mark.MarkType }) {
				childMarks = append(childMarks, mark)
			}
			nodes, err = adfInlines(n, childMarks, depth+1)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("markdown to ADF: unsupported %s", n.Type)
		}
		content = append(content, nodes...)
	}
	return content, nil
}

func adfMark(n *bf.Node) (adf.MarkNode, error) {
	mark := adf.MarkNode{}
	switch n.Type {
	case bf.Emph:
		mark.MarkType = adf.MarkEm
	case bf.Strong:
		mark.MarkType = adf.MarkStrong
	case bf.Del:
		mark.MarkType = adf.MarkStrike
	case bf.Link:
		if n.NoteID != 0 {
			return mark, errors.New("markdown to ADF: unsupported footnote Link")
		}
		if strings.ContainsRune(string(n.Destination), 0) || strings.ContainsRune(string(n.Title), 0) {
			return mark, errors.New("markdown to ADF: unsupported reference definition inside Link")
		}
		href := adfUnescape(string(n.Destination))
		if err := validateADFLink(href); err != nil {
			return mark, err
		}
		mark.MarkType = adf.MarkLink
		attrs := map[string]any{"href": href}
		if len(n.Title) != 0 {
			attrs["title"] = adfTitleEscape.ReplaceAllStringFunc(string(n.Title), func(escape string) string {
				if escape[0] == '\\' {
					return escape[1:]
				}
				return html.UnescapeString(escape)
			})
		}
		mark.Attributes = attrs
	}
	return mark, nil
}

func validateADFLink(href string) error {
	// Inspect decoded bytes without rewriting the destination sent to Jira.
	decoded, err := url.PathUnescape(href)
	if err != nil || !utf8.ValidString(decoded) {
		return errors.New("markdown to ADF: malformed Link URI encoding")
	}
	if strings.ContainsFunc(decoded, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) {
		return errors.New("markdown to ADF: unsupported control or format character in Link URI")
	}
	if strings.ContainsFunc(href, unicode.IsSpace) || strings.ContainsAny(href, "\\<>\"{}|^`") {
		return errors.New("markdown to ADF: malformed Link URI characters")
	}
	uri, err := url.Parse(href)
	if err != nil {
		return errors.New("markdown to ADF: malformed Link URI")
	}
	if uri.User != nil {
		return errors.New("markdown to ADF: unsupported userinfo in Link URI")
	}
	switch strings.ToLower(uri.Scheme) {
	case "http", "https":
		if uri.Opaque != "" || uri.Hostname() == "" {
			return errors.New("markdown to ADF: HTTP Link URI requires a host")
		}
	case "mailto":
		if uri.Opaque == "" || uri.Host != "" || uri.Fragment != "" {
			return errors.New("markdown to ADF: malformed mailto Link URI")
		}
		recipients, err := url.PathUnescape(uri.Opaque)
		if err != nil {
			return errors.New("markdown to ADF: malformed mailto Link URI encoding")
		}
		if _, err := mail.ParseAddressList(recipients); err != nil {
			return errors.New("markdown to ADF: malformed mailto Link recipient")
		}
	default:
		return errors.New("markdown to ADF: Link URI requires an explicit http, https, or mailto scheme")
	}
	return nil
}

func adfText(text string, marks []adf.MarkNode) []*adf.Node {
	var content []*adf.Node
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			content = append(content, &adf.Node{NodeType: adf.InlineNodeHardBreak})
		}
		if line != "" {
			content = append(content, &adf.Node{NodeType: adf.ChildNodeText, NodeValue: adf.NodeValue{Text: line, Marks: slices.Clone(marks)}})
		}
	}
	return content
}

func adfCodeText(text string, marks []adf.MarkNode) []*adf.Node {
	if text == "" {
		return nil
	}
	return []*adf.Node{{NodeType: adf.ChildNodeText, NodeValue: adf.NodeValue{Text: text, Marks: slices.Clone(marks)}}}
}

func adfRestoreCode(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\x00ref\x00", ""), "\x00end\x00", "")
}

func adfUnescape(text string) string {
	return adfEntity.ReplaceAllStringFunc(text, html.UnescapeString)
}
