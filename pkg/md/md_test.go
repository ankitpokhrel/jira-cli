package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertJiraNestedLists(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single level list unchanged",
			input:    "* Item 1\n* Item 2",
			expected: "* Item 1\n* Item 2",
		},
		{
			name:     "nested list converted",
			input:    "* Item 1\n** Subitem 1\n** Subitem 2",
			expected: "* Item 1\n\t- Subitem 1\n\t- Subitem 2",
		},
		{
			name:     "deeply nested list",
			input:    "* Item\n** Level 2\n*** Level 3\n**** Level 4",
			expected: "* Item\n\t- Level 2\n\t\t- Level 3\n\t\t\t- Level 4",
		},
		{
			name:     "bold text not affected",
			input:    "**bold text** and more",
			expected: "**bold text** and more",
		},
		{
			name:     "mixed content",
			input:    "* Item with **bold**\n** Subitem\n*** Deep item",
			expected: "* Item with **bold**\n\t- Subitem\n\t\t- Deep item",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertJiraNestedLists(tc.input))
		})
	}
}

func TestToJiraMDWithNestedLists(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "jira wiki nested list",
			input:    "* Item 1\n** Subitem 1\n** Subitem 2\n* Item 2",
			expected: "* Item 1\n** Subitem 1\n** Subitem 2\n* Item 2\n\n",
		},
		{
			name:     "three level nesting",
			input:    "* Level 1\n** Level 2\n*** Level 3",
			expected: "* Level 1\n** Level 2\n*** Level 3\n\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ToJiraMD(tc.input))
		})
	}
}

func TestToJiraMD(t *testing.T) {
	jfm := `# H1
Some _Markdown_ text.

## H2
Foobar.

### H3
Fuga

> quote

- - - -

**strong text**
~~strikethrough text~~
[Example Domain](http://www.example.com/)
![](https://path.to/image.jpg)

* list1
* list2
* list3

Paragraph

1. number1
2. number2
3. number3

|a  |b  |c  |
|---|---|---|
|1  |2  |3  |
|4  |5  |6  |

{panel:title=My Title}
**Subtitle**

Some text with a title
{panel}

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("hello world")
}` + "```"

	expected := `h1. H1
Some _Markdown_ text.

h2. H2
Foobar.

h3. H3
Fuga

{quote}
quote

{quote}


----
*strong text*
-strikethrough text-
[Example Domain|http://www.example.com/]
!https://path.to/image.jpg!

* list1
* list2
* list3

Paragraph

# number1
# number2
# number3

||a||b||c||
|1|2|3|
|4|5|6|

{panel:title=My Title}
*Subtitle*

Some text with a title
{panel}

` + "```go" + `
package main

import "fmt"

func main\(\) {
    fmt.Println\("hello world"\)
}` + "```\n\n"

	assert.Equal(t, expected, ToJiraMD(jfm))
}
