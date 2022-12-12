package jirawiki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:dupl
func TestParseHeadingTags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1",
			input:    "h1. Heading 1",
			expected: "# Heading 1\n",
		},
		{
			name:     "h2",
			input:    "h2. Heading 2",
			expected: "## Heading 2\n",
		},
		{
			name:     "h3",
			input:    "h3. Heading 3",
			expected: "### Heading 3\n",
		},
		{
			name:     "h4",
			input:    "h4. Heading 4",
			expected: "#### Heading 4\n",
		},
		{
			name:     "h5",
			input:    "h5. Heading 5",
			expected: "##### Heading 5\n",
		},
		{
			name:     "h6",
			input:    "h6. Heading 6",
			expected: "###### Heading 6\n",
		},
		{
			name: "all headings",
			input: `h1. Heading 1
h2. Heading 2
h3. Heading 3
h4. Heading 4
h5. Heading 5
h6. Heading 6`,
			expected: `# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6
`,
		},
		{
			name:     "empty heading",
			input:    "h3.",
			expected: "###\n",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

func TestParseTextEffectTags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold",
			input:    "*bold*",
			expected: "**bold**\n\n",
		},
		{
			name:     "bold, italic and strikethrough",
			input:    "Line with *bold*, _italic_ and -strikethrough- text. And _italics with *bold* text in it_.",
			expected: "Line with **bold**, _italic_ and -strikethrough- text. And _italics with **bold** text in it_.\n",
		},
		{
			name:     "partially closed bold tag",
			input:    "*bold",
			expected: "**bold**\n",
		},
		{
			name:     "partially closed bold tag in a sentence",
			input:    "Line with *bold and _italic_ text.",
			expected: "Line with **bold and _italic_ text.**\n",
		},
		{
			name:     "partially closed bold and italic in a sentence",
			input:    "Line with *bold and _italic text.",
			expected: "Line with **bold and _italic text.**\n",
		},
		{
			name:     "normal sentence with braces and semicolon should be parsed as is",
			input:    "Line with semicolon inside curly braces {{MySQL::Conn()}}.",
			expected: "Line with semicolon inside curly braces {{MySQL::Conn()}}.",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

func TestParseListTags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "unordered list",
			input: `* Item 1
 * Item 2
 ** Subitem 1
 ** Subitem 2
 *** Subitem 2 item 1
 * Item 3
 * Item 4`,
			expected: `- Item 1
- Item 2
	- Subitem 1
	- Subitem 2
		- Subitem 2 item 1
- Item 3
- Item 4
`,
		},
		{
			name: "ordered list",
			input: `# Ordered list item 1
 ## Ordered list subitem 1
 ## Ordered list subitem 2
 ### Ordered list subitem 2 item 1`,
			expected: `- Ordered list item 1
	- Ordered list subitem 1
	- Ordered list subitem 2
		- Ordered list subitem 2 item 1
`,
		},
		{
			name: "empty sublist",
			input: `* Item 1
** Subitem 1
**`,
			expected: `- Item 1
	- Subitem 1
	-
`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

//nolint:dupl
func TestParseReferenceLinks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid link with title",
			input:    "[title|https://ankit.pl]",
			expected: "[title](https://ankit.pl)\n",
		},
		{
			name:     "valid link without title",
			input:    "[https://ankit.pl]",
			expected: "[](https://ankit.pl)\n",
		},
		{
			name:     "mailto link",
			input:    "[mailto:hi@ankit.pl]",
			expected: "[](mailto:hi@ankit.pl)\n",
		},
		{
			name:     "anchor link",
			input:    "[#somewhere]",
			expected: "[](#somewhere)\n",
		},
		{
			name:     "valid link wrapped around texts",
			input:    "A text with [a link|https://ankit.pl] in between.",
			expected: "A text with [a link](https://ankit.pl) in between.\n",
		},
		{
			name:     "valid link mixed with bold, italic and strikethrough text",
			input:    "A *bold*, _italic_ and -strikethrough- text with [a link|https://ankit.pl] in between.",
			expected: "A **bold**, _italic_ and -strikethrough- text with [a link](https://ankit.pl) in between.\n",
		},
		{
			name:     "invalid link",
			input:    "This is a [Link|https://ankit.pl, and some texts.",
			expected: "This is a [Link|https://ankit.pl, and some texts.",
		},
		{
			name:     "empty link",
			input:    "This link is empty []",
			expected: "This link is empty []()\n",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

func TestParseBlockQuote(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "blockquote",
			input: `{quote}Blockquote
{quote}`,
			expected: "\n> Blockquote\n\n",
		},
		{
			name:     "one line blockquote",
			input:    "{quote}Blockquote {without} ending new line{quote}",
			expected: "\n> Blockquote {without} ending new line\n",
		},
		{
			name:     "unclosed blockquote",
			input:    "{quote}Blockquote {without} closing and a *bold* text",
			expected: "\n> Blockquote {without} closing and a **bold** text\n",
		},
		{
			name:     "inline blockquote",
			input:    "bq. Inline blockquote",
			expected: "\n> Inline blockquote\n",
		},
		{
			name:     "empty inline blockquote",
			input:    "bq.",
			expected: "\n>\n",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

func TestParsePanels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "panel",
			input: `{panel}
This is a panel description.
And, a new line.
{panel}
`,
			expected: `
---
This is a panel description.
And, a new line.
---
`,
		},
		{
			name: "panel with attributes",
			input: `{panel:title=Panel Title|bgColor=#fff}
Panel description.
{panel}
`,
			expected: `
---
**Panel Title**

Panel description.
---
`,
		},
		{
			name: "panel with alternate syntax",
			input: `{panel:Panel Title}
Panel description.
{panel}
`,
			expected: `
---
**Panel Title**

Panel description.
---
`,
		},
		{
			name: "panel with inline syntax",
			input: `{panel}Panel description.{panel}
`,
			expected: `
---
Panel description.
---
`,
		},
		{
			name: "panel with inline syntax and title",
			input: `{panel:title=Panel Title}Panel description.{panel}
`,
			expected: `
---
**Panel Title**

Panel description.
---
`,
		},
		{
			name: "panel with invalid syntax",
			input: `{panel
Panel description.
{panel}
`,
			expected: `{panel
Panel description.
---
`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

func TestParseFencedCodeBlocks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "preformatted block",
			input: `{noformat}
This text *should* be displayed as is.
{noformat}`,
			expected: "\n```\nThis text *should* be displayed as is.\n```\n",
		},
		{
			name: "code block",
			input: `{code}
<html>HTML</html>
{code}`,
			expected: "\n```\n<html>HTML</html>\n```\n",
		},
		{
			name: "code block with language",
			input: `{code:go}
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
{code}`,
			expected: "\n```go" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```\n",
		},
		{
			name: "code block with language in title",
			input: `{code:title=hello.go}
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
{code}`,
			expected: "\n```go" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```\n",
		},
		{
			name: "out of memory bug in preformatted block #221",
			input: `{noformat}
1
2
3
4
5
6
{noformat}
{noformat}
7
{noformat}`,
			expected: "\n```\n1\n2\n3\n4\n5\n6\n```\n\n```\n7\n```\n",
		},
		{
			name: "Back to back fenced code block should not result in infinite loop",
			input: `{code:go}
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
{code}
{code}
fn main() {
	println!("Hello, world!");
}
{code}`,
			expected: "\n```go" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```\n" + "\n```" + `
fn main() {
	println!("Hello, world!");
}
` + "```\n",
		},
		{
			name: "Back to back fenced code block in the list should render properly",
			input: `1. Ordered list item A
{code:go}
package main

func main() {
	println("Hello, World!")
}{code}

2. Ordered list item B
{code:go}
// no code
{code}`,
			expected: "1. Ordered list item A\n\n```go" + `
package main

func main() {
	println("Hello, World!")
}` + "\n```\n\n2. Ordered list item B\n\n```go\n// no code" + "\n```\n",
		},
		{
			name: "Back to back preformatted block in the list should render properly",
			input: `1. Ordered list item A
{noformat}
package main

func main() {
	println("Hello, World!")
}{noformat}

2. Ordered list item B
{noformat}
// no code
{noformat}`,
			expected: "1. Ordered list item A\n\n```" + `
package main

func main() {
	println("Hello, World!")
}` + "\n```\n\n2. Ordered list item B\n\n```\n// no code" + "\n```\n",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}

func TestTables(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "valid table",
			input: `||heading 1||heading 2||heading 3||
|col A1|col A2|col A3|
|col B1|col B2|col B3|`,
			expected: `|heading 1|heading 2|heading 3|
|---|---|---|
|col A1|col A2|col A3|
|col B1|col B2|col B3|
`,
		},
		{
			name:  "valid table with no rows",
			input: `||heading 1||heading 2||heading 3||`,
			expected: `|heading 1|heading 2|heading 3|
|---|---|---|
`,
		},
		{
			name: "valid table with no headers",
			input: `||||
|col A1|
|col B1|`,
			expected: `||
|---|
|col A1|
|col B1|
`,
		},
		{
			name: "invalid table",
			input: `||heading 1||heading 2||heading 3
|col A1|col A2|col A3|`,
			expected: "||heading 1||heading 2||heading 3|col A1|col A2|col A3|\n",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, Parse(tc.input))
		})
	}
}
