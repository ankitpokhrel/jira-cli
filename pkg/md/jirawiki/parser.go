package jirawiki

import (
	"fmt"
	"strings"
)

// Supported Jira wiki tags.
const (
	TagHeading1      = "h1."
	TagHeading2      = "h2."
	TagHeading3      = "h3."
	TagHeading4      = "h4."
	TagHeading5      = "h5."
	TagHeading6      = "h6."
	TagBlockQuote    = "bq."
	TagQuote         = "{quote}"
	TagPanel         = "{panel}"
	TagCodeBlock     = "{code}"
	TagNoFormat      = "{noformat}"
	TagLink          = "["
	TagOrderedList   = "#"
	TagUnorderedList = "*" // '*' can be either be bold or an unordered list 🤦.
	TagBold          = "*"
	TagTable         = "||"

	// Let's group tags based on their behavior.
	typeTagTextEffect    = "text-effect"
	typeTagList          = "list"
	typeTagHeading       = "heading"
	typeTagInlineQuote   = "inline-quote"
	typeTagReferenceLink = "link"
	typeTagFencedCode    = "code"
	typeTagTable         = "table"
	typeTagOther         = "other"

	// Supported attributes.
	attrTitle = "title"

	// Line feeds.
	newLine        = '\n'
	carriageReturn = '\r'
)

var validTags = []string{
	TagHeading1,
	TagHeading2,
	TagHeading3,
	TagHeading4,
	TagHeading5,
	TagHeading6,
	TagBlockQuote,
	TagQuote,
	TagPanel,
	TagCodeBlock,
	TagNoFormat,
	TagLink,
	TagOrderedList,
	TagUnorderedList,
	TagBold,
	TagTable,
}

var replacements = map[string]string{
	TagHeading1:    "#",  // '#' can be either be a h1 tag or an ordered list 🤷.
	TagHeading2:    "##", // '##' could mean a h2 tag or indentation for an ordered list 😑.
	TagHeading3:    "###",
	TagHeading4:    "####",
	TagHeading5:    "#####",
	TagHeading6:    "######",
	TagQuote:       "> ",
	TagPanel:       "---",
	TagBlockQuote:  ">",
	TagCodeBlock:   "```",
	TagNoFormat:    "```",
	TagOrderedList: "-",
	TagBold:        "**",
	TagTable:       "|",
}

// Parse converts input string to Jira markdown.
func Parse(input string) string {
	return secondPass(firstPass(input))
}

/*
First pass:
  - Fetch all lines from the input while skipping unnecessary line feeds.
*/
func firstPass(input string) []string {
	var (
		beg   = 0
		size  = len(input)
		lines = make([]string, 0)
	)

	for beg < size {
		end := beg
		for end < size && input[end] != carriageReturn && input[end] != newLine {
			end++
		}
		lines = append(lines, input[beg:end])

		for end < size && input[end] == carriageReturn {
			end++
		}

		beg = end + 1
	}

	return lines
}

/*
Second pass: actual rendering.
  - Process each line to search and mark tags.
  - Use replacements to prepare markdown.
*/
func secondPass(lines []string) string {
	var (
		out     strings.Builder
		lineNum int
	)

	for lineNum < len(lines) {
		line := lines[lineNum]
		tokens := tokenize(line)

		if len(tokens) == 0 {
			out.WriteString(line)

			lineNum++
			if lineNum < len(lines)-1 {
				out.WriteRune(newLine)
			}
			continue
		}

		var beg int

	out:
		for beg < len(line) {
			end := beg

			if token, ok := tokenStarts(beg, tokens); ok {
				switch token.family {
				case typeTagTextEffect:
					end = token.handleTextEffects(line, &out)
				case typeTagHeading:
					end = token.handleHeadings(line, &out)
				case typeTagInlineQuote:
					end = token.handleInlineBlockQuote(line, &out)
				case typeTagList:
					end = token.handleList(line, &out)
				case typeTagFencedCode:
					lineNum = token.handleFencedCodeBlock(lineNum, lines, &out)
					break out
				case typeTagReferenceLink:
					end = token.handleReferenceLink(line, &out)
				case typeTagTable:
					end = token.handleTable(line, &out)
				case typeTagOther:
					if token.tag == TagQuote {
						// If end is same as size of the input, it implies that
						// we've found a closing token, and we will ignore it.
						if token.endIdx != len(line)-1 {
							out.WriteString(fmt.Sprintf("\n%s", replacements[token.tag]))
						}
					} else {
						out.WriteString(fmt.Sprintf("\n%s", replacements[token.tag]))
					}

					if token.tag == TagPanel {
						if t, ok := token.attrs[attrTitle]; ok {
							out.WriteString(fmt.Sprintf("\n**%s**\n", t))
						}

						if token.endIdx != len(line)-1 {
							out.WriteRune(newLine)
						}
					}

					end = token.endIdx
				}
			} else {
				out.WriteRune(rune(line[beg]))
			}

			end++
			beg = end
		}

		lineNum++
		out.WriteRune(newLine)
	}

	return out.String()
}

// Mark tokens in a given string.
func tokenize(line string) []*Token { //nolint:gocyclo
	line = strings.TrimSpace(line)

	var (
		tokens []*Token
		beg    = 0
		size   = len(line)
	)

out:
	for beg < size-1 {
		var (
			end     int
			tagType = getTagType(line, beg)
		)

		switch tagType {
		case typeTagTextEffect:
			end = beg + 1
			for end < len(line) && line[end] != line[beg] {
				end++
			}

			var word string
			if end < size-1 {
				word = line[beg : end+1]
			} else {
				word = line[beg:end]
			}

			tokens = append(tokens, &Token{
				tag:      word,
				family:   typeTagTextEffect,
				startIdx: beg,
				endIdx:   end,
			})
			end++
		case typeTagHeading:
			fallthrough
		case typeTagInlineQuote:
			end = beg + 1
			for end < len(line) && line[end] != '.' {
				end++
			}
			word := line[beg : end+1]

			tokens = append(tokens, &Token{
				tag:      word,
				family:   tagType,
				startIdx: beg,
				endIdx:   end,
			})
			break out
		case typeTagList:
			end = beg + 1
			for end < len(line) && line[end] == line[beg] {
				end++
			}
			word := line[beg:end]

			tokens = append(tokens, &Token{
				tag:      word,
				family:   typeTagList,
				startIdx: beg,
				endIdx:   end,
			})
			end++
		case typeTagReferenceLink:
			end = beg + 1
			for end < len(line) && line[end] != ']' {
				end++
			}
			word := line[beg : end+1]

			tokens = append(tokens, &Token{
				tag:      word,
				family:   typeTagReferenceLink,
				startIdx: beg,
				endIdx:   end,
			})
		case typeTagTable:
			end = len(line) - 1

			tokens = append(tokens, &Token{
				tag:      line,
				family:   typeTagTable,
				startIdx: beg,
				endIdx:   end,
			})
		default:
			end = beg + 1
			for end < size && line[end] != '*' && line[end] != '{' && line[end] != '}' && line[end] != '[' && line[end] != ']' {
				end++
			}

			if end != size && line[end] != '*' && line[end] != '{' && line[end] != '[' {
				end++
			}

			word := line[beg:end]
			word, attrs := extractAttributes(word)

			if isToken(word) {
				fam := typeTagOther
				if word == TagCodeBlock || word == TagNoFormat {
					fam = typeTagFencedCode
				}

				tokens = append(tokens, &Token{
					tag:      word,
					family:   fam,
					attrs:    attrs,
					startIdx: beg,
					endIdx:   end - 1,
				})
			}
		}

		beg = end
	}

	return tokens
}

func extractAttributes(token string) (string, map[string]string) {
	attrs := make(map[string]string)

	if token[0] != '{' || !strings.Contains(token, ":") {
		return token, attrs
	}

	pieces := strings.Split(token, ":")
	if len(pieces) != 2 || pieces[1] == "" {
		return token, attrs
	}

	tag := pieces[0] + "}"
	meta := pieces[1][0 : len(pieces[1])-1]

	// We only support title attribute at the moment.
	validAttr := attrTitle

	pieces = strings.Split(meta, "|")
	for _, m := range pieces {
		props := strings.Split(m, "=")
		if len(props) == 1 {
			attrs[validAttr] = props[0]
		} else {
			key, val := props[0], props[1]
			if key == validAttr {
				attrs[key] = val
			}
		}
	}

	return tag, attrs
}

// Token represents jira tags in a given string.
type Token struct {
	tag      string
	family   string
	attrs    map[string]string
	startIdx int
	endIdx   int
}

func (t *Token) handleTextEffects(line string, out *strings.Builder) int {
	word := line[t.startIdx+1 : t.endIdx]

	out.WriteString(replacements[string(line[t.startIdx])])
	out.WriteString(word)
	out.WriteString(replacements[string(line[t.startIdx])])

	if t.endIdx == len(line)-1 {
		out.WriteRune(newLine)
	}

	return t.endIdx
}

func (t *Token) handleHeadings(line string, out *strings.Builder) int {
	word := line[t.endIdx+1:]

	out.WriteString(replacements[t.tag])
	out.WriteString(word)

	return t.endIdx + len(word)
}

func (t *Token) handleInlineBlockQuote(line string, out *strings.Builder) int {
	word := line[t.endIdx+1:]

	out.WriteString(fmt.Sprintf("\n%s", replacements[t.tag]))
	out.WriteString(word)

	return t.endIdx + len(word)
}

func (t *Token) handleList(line string, out *strings.Builder) int {
	end := t.endIdx + 1

	for i := t.startIdx; i < t.endIdx-1; i++ {
		out.WriteRune('\t')
	}

	if end >= len(line) {
		out.WriteString("-")
		return t.endIdx
	}

	rem := strings.TrimSpace(line[end:])
	out.WriteString(fmt.Sprintf("- %s", rem))

	end += len(rem) + 1

	return end
}

func (t *Token) handleFencedCodeBlock(idx int, lines []string, out *strings.Builder) int {
	if idx == len(lines)-1 {
		return t.endIdx
	}

	out.WriteString(fmt.Sprintf("\n%s", replacements[t.tag]))

	if t, ok := t.attrs[attrTitle]; ok {
		pieces := strings.Split(t, ".")
		if len(pieces) == 2 {
			out.WriteString(pieces[1])
		} else {
			out.WriteString(t)
		}
	}

	out.WriteRune(newLine)

	i := idx + 1
	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == TagCodeBlock || line == TagNoFormat {
			break
		}

		if x := checkForInlineClose(line); x > 0 {
			out.WriteString(line[:x])
			out.WriteRune(newLine)
			break
		} else {
			// Write everything as is.
			out.WriteString(lines[i])
		}
		out.WriteRune(newLine)
	}
	out.WriteString(replacements[t.tag])

	return i
}

func (t *Token) handleReferenceLink(line string, out *strings.Builder) int {
	if len(line) < 2 {
		return t.endIdx
	}

	body := line[t.startIdx+1 : t.endIdx]
	pieces := strings.Split(body, "|")

	var link string

	if len(pieces) == 2 {
		link = fmt.Sprintf("[%s](%s)", pieces[0], pieces[1])
	} else {
		link = fmt.Sprintf("[](%s)", pieces[0])
	}

	out.WriteString(link)

	return t.endIdx
}

func (t *Token) handleTable(line string, out *strings.Builder) int {
	if line[1] != '|' {
		out.WriteString(line)
		return t.endIdx
	}

	headers := strings.ReplaceAll(line, TagTable, replacements[TagTable])
	cols := strings.Split(headers, "|")

	var sep strings.Builder
	for i := 0; i < len(cols)-2; i++ {
		sep.WriteString("|---")
	}

	row := fmt.Sprintf("%s\n%s|", headers, sep.String())

	out.WriteString(row)

	return t.endIdx
}

func isToken(inp string) bool {
	for _, tag := range validTags {
		if inp == tag {
			return true
		}
	}
	return false
}

func tokenStarts(idx int, tokens []*Token) (*Token, bool) {
	for _, token := range tokens {
		if idx == token.startIdx {
			return token, true
		}
	}
	return nil, false
}

func getTagType(line string, beg int) string {
	if isTextEffect(line[beg], line[beg+1]) {
		return typeTagTextEffect
	}
	if isListTag(line[beg], line[beg+1]) {
		return typeTagList
	}
	if isHeadingsTag(beg, line) {
		return typeTagHeading
	}
	if isInlineBlockQuote(beg, line) {
		return typeTagInlineQuote
	}
	if isReferenceLink(beg, line) {
		return typeTagReferenceLink
	}
	if isTable(beg, line) {
		return typeTagTable
	}
	return typeTagOther
}

func isTextEffect(beg, next uint8) bool {
	s := string(beg)
	return s == TagBold && (next != ' ' && next != beg)
}

func isListTag(beg, next uint8) bool {
	s := string(beg)
	return (s == TagOrderedList || s == TagUnorderedList) && (next == ' ' || next == beg)
}

func isHeadingsTag(beg int, line string) bool {
	size := len(line)
	if size < 3 {
		return false
	}
	return line[beg] == 'h' && line[2] == '.'
}

func isInlineBlockQuote(beg int, line string) bool {
	size := len(line)
	if size < 3 {
		return false
	}
	return line[beg] == 'b' && line[2] == '.'
}

func isReferenceLink(beg int, line string) bool {
	if line[beg] != '[' {
		return false
	}

	var end int

	for beg < len(line) {
		end = beg + 1
		for end < len(line) && line[end] != ']' {
			end++
		}
		break
	}

	return end < len(line) && line[end] == ']'
}

func isTable(beg int, line string) bool {
	end := len(line) - 1
	return end != beg && line[beg] == '|' && line[end] == '|'
}

func checkForInlineClose(line string) int {
	n := len(line)

	if n > len(TagCodeBlock) && line[n-len(TagCodeBlock):] == TagCodeBlock {
		return n - len(TagCodeBlock)
	}
	if n > len(TagNoFormat) && line[n-len(TagNoFormat):] == TagNoFormat {
		return n - len(TagNoFormat)
	}

	return 0
}
