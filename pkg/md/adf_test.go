package md

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

func TestToADF(t *testing.T) {
	tests := []struct {
		name, markdown, content string
	}{
		{
			name: "empty", markdown: "", content: `[]`,
		},
		{
			name: "whitespace", markdown: " \t\n\n", content: `[]`,
		},
		{
			name:     "line breaks paragraphs tabs Unicode",
			markdown: "first\tcolumn\nsecond  \nthird\\\nfourth\n\n世界 café 🦊\n\nlast\n",
			content: `[
				{"type":"paragraph","content":[{"type":"text","text":"first\tcolumn"},{"type":"hardBreak"},{"type":"text","text":"second"},{"type":"hardBreak"},{"type":"text","text":"third"},{"type":"hardBreak"},{"type":"text","text":"fourth"}]},
				{"type":"paragraph","content":[{"type":"text","text":"世界 café 🦊"}]},
				{"type":"paragraph","content":[{"type":"text","text":"last"}]}
			]`,
		},
		{
			name: "headings and rule", markdown: "# one\n\n## two\n\n### three\n\n#### four\n\n##### five\n\n###### six\n\n---\n\nsetext\n======\n",
			content: `[
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"one"}]},
				{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"two"}]},
				{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"three"}]},
				{"type":"heading","attrs":{"level":4},"content":[{"type":"text","text":"four"}]},
				{"type":"heading","attrs":{"level":5},"content":[{"type":"text","text":"five"}]},
				{"type":"heading","attrs":{"level":6},"content":[{"type":"text","text":"six"}]},
				{"type":"rule"},
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"setext"}]}
			]`,
		},
		{
			name: "marks", markdown: "**bold _both_ bold** ~~gone~~ `a\t<&amp;>`",
			content: `[{"type":"paragraph","content":[
				{"type":"text","text":"bold ","marks":[{"type":"strong"}]},
				{"type":"text","text":"both","marks":[{"type":"strong"},{"type":"em"}]},
				{"type":"text","text":" bold","marks":[{"type":"strong"}]},
				{"type":"text","text":" "},
				{"type":"text","text":"gone","marks":[{"type":"strike"}]},
				{"type":"text","text":" "},
				{"type":"text","text":"a\t<&amp;>","marks":[{"type":"code"}]}
			]}]`,
		},
		{
			name: "repeated emphasis and marked line break", markdown: "*outer _inner_ outer*\n\n**one\ntwo**",
			content: `[
				{"type":"paragraph","content":[{"type":"text","text":"outer ","marks":[{"type":"em"}]},{"type":"text","text":"inner","marks":[{"type":"em"}]},{"type":"text","text":" outer","marks":[{"type":"em"}]}]},
				{"type":"paragraph","content":[{"type":"text","text":"one","marks":[{"type":"strong"}]},{"type":"hardBreak"},{"type":"text","text":"two","marks":[{"type":"strong"}]}]}
			]`,
		},
		{
			name: "links and JSON escaping", markdown: "[**label** `code`](https://example.invalid/a?x=1&y=2 \"a &quot;title&quot;\")",
			content: `[{"type":"paragraph","content":[
				{"type":"text","text":"label","marks":[{"type":"link","attrs":{"href":"https://example.invalid/a?x=1&y=2","title":"a \"title\""}},{"type":"strong"}]},
				{"type":"text","text":" ","marks":[{"type":"link","attrs":{"href":"https://example.invalid/a?x=1&y=2","title":"a \"title\""}}]},
				{"type":"text","text":"code","marks":[{"type":"link","attrs":{"href":"https://example.invalid/a?x=1&y=2","title":"a \"title\""}},{"type":"code"}]}
			]}]`,
		},
		{
			name: "escaped Markdown and entities", markdown: `\*literal\* &lt; &amp; &#x1F98A; &not-an-entity \\`,
			content: `[{"type":"paragraph","content":[
				{"type":"text","text":"*"},{"type":"text","text":"literal"},{"type":"text","text":"*"},
				{"type":"text","text":" "},{"type":"text","text":"<"},
				{"type":"text","text":" "},{"type":"text","text":"&"},
				{"type":"text","text":" "},{"type":"text","text":"🦊"},
				{"type":"text","text":" &not-an-entity "},{"type":"text","text":"\\"}
			]}]`,
		},
		{
			name: "nested lists and ordered starts", markdown: "7. first\n    - nested\n        3. inner\n        4. next\n8. second\n\nparagraph\n\n2. restart\n",
			content: `[
				{"type":"orderedList","attrs":{"order":7},"content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]},
						{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"nested"}]},
							{"type":"orderedList","attrs":{"order":3},"content":[
								{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"inner"}]}]},
								{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"next"}]}]}
							]}
						]}]}
					]},
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"second"}]}]}
				]},
				{"type":"paragraph","content":[{"type":"text","text":"paragraph"}]},
				{"type":"orderedList","attrs":{"order":2},"content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"restart"}]}]}]}
			]`,
		},
		{
			name: "blockquote allowed children", markdown: "> quote\n> next\n>\n> - bullet\n>\n> 9. item\n>\n> ```go\n> value\n> ```\n",
			content: `[{"type":"blockquote","content":[
				{"type":"paragraph","content":[{"type":"text","text":"quote"},{"type":"hardBreak"},{"type":"text","text":"next"}]},
				{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"bullet"}]}]}]},
				{"type":"orderedList","attrs":{"order":9},"content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item"}]}]}]},
				{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"value\n"}]}
			]}]`,
		},
		{
			name: "fenced and indented code", markdown: "```go\n\tvalue := \"<&amp;>\"\n\n7. not a list\n```\n\n    first\n    \tsecond\n\n3. actual list\n",
			content: `[
				{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"\tvalue := \"<&amp;>\"\n\n7. not a list\n"}]},
				{"type":"codeBlock","content":[{"type":"text","text":"first\n\tsecond\n"}]},
				{"type":"orderedList","attrs":{"order":3},"content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"actual list"}]}]}]}
			]`,
		},
		{
			name: "tilde fence", markdown: "~~~python\n\tvalue = 1\n~~~\n",
			content: `[{"type":"codeBlock","attrs":{"language":"python"},"content":[{"type":"text","text":"\tvalue = 1\n"}]}]`,
		},
		{
			name: "CRLF and CR line endings", markdown: "a\r\nb\rc\r\n\r\nd",
			content: `[{"type":"paragraph","content":[{"type":"text","text":"a"},{"type":"hardBreak"},{"type":"text","text":"b"},{"type":"hardBreak"},{"type":"text","text":"c"}]},{"type":"paragraph","content":[{"type":"text","text":"d"}]}]`,
		},
		{
			name: "terminal backslash", markdown: "tail\\",
			content: `[{"type":"paragraph","content":[{"type":"text","text":"tail\\"}]}]`,
		},
		{
			name: "auto links", markdown: "https://example.invalid/?a=1&b=2 <user@example.invalid>",
			content: `[{"type":"paragraph","content":[{"type":"text","text":"https://example.invalid/?a=1&b=2","marks":[{"type":"link","attrs":{"href":"https://example.invalid/?a=1&b=2"}}]},{"type":"text","text":" "},{"type":"text","text":"user@example.invalid","marks":[{"type":"link","attrs":{"href":"mailto:user@example.invalid"}}]}]}]`,
		},
		{
			name: "empty code and empty quote", markdown: "```\n```\n\n>\n",
			content: `[{"type":"codeBlock"},{"type":"blockquote","content":[{"type":"paragraph"}]}]`,
		},
		{
			name: "GFM table", markdown: "| a | **b** |\n| --- | --- |\n| `c` | |\n",
			content: `[{"type":"table","content":[
				{"type":"tableRow","content":[
					{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]},
					{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"b","marks":[{"type":"strong"}]}]}]}
				]},
				{"type":"tableRow","content":[
					{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"c","marks":[{"type":"code"}]}]}]},
					{"type":"tableCell","content":[{"type":"paragraph"}]}
				]}
			]}]`,
		},
		{
			name: "code with unsupported syntax remains literal", markdown: "```\n[^unused]: note\n[id]: /target\n<div>literal</div>\n![image](media)\n```",
			content: `[{"type":"codeBlock","content":[{"type":"text","text":"[^unused]: note\n[id]: /target\n<div>literal</div>\n![image](media)\n"}]}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ToADF(tt.markdown)
			if err != nil {
				t.Fatal(err)
			}
			if doc == nil || doc.Version != 1 || doc.DocType != "doc" || doc.Content == nil {
				t.Fatalf("invalid document: %#v", doc)
			}
			data, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(`{"version":1,"type":"doc","content":`+tt.content+`}`), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("ADF = %s; want content %s", data, tt.content)
			}
		})
	}
}

func TestToADFUnsupported(t *testing.T) {
	tests := []struct{ name, markdown, format string }{
		{"invalid UTF8", string([]byte{0xff}), "UTF-8"},
		{"NUL", "before\x00after", "NUL"},
		{"inline HTML", "before <b>private</b>", "HTMLSpan"},
		{"block HTML", "before\n\n<div>private</div>\n", "HTMLBlock"},
		{"HTML comment", "<!-- private -->", "HTMLSpan"},
		{"image", "before ![private](https://example.invalid/image.png)", "Image"},
		{"definition list", "term\n: private definition\n", "definition List"},
		{"unused footnote", "[^unused]: private note\n", "footnote definition"},
		{"footnote", "before[^note]\n\n[^note]: private note", "footnote"},
		{"inline footnote", "before^[private note]", "footnote"},
		{"reference definition", "[id]: /private", "reference"},
		{"nested reference definition", "- first\n\n    [id]: /private\n", "reference"},
		{"heading ID", "# private {#anchor}", "Heading ID"},
		{"fence metadata", "```go linenos\nprivate\n```", "CodeBlock"},
		{"unclosed fence", "```go\nprivate", "CodeBlock fence"},
		{"language fence misparsed in list", "6. first\n\n    ```go\n    91. private\n    ```\n\n7. last\n", "CodeBlock fence"},
		{"multiline inline code", "before `private\nvalue`", "multiline inline Code"},
		{"aligned table", "a | b\n:--- | ---:\nprivate | value\n", "TableCell alignment"},
		{"excess table cells", "a | b\n--- | ---\nvalue | value | private\n", "Table row"},
		{"excess empty table cells", "| a | b |\n| --- | --- |\n| value | value | |\n", "Table row"},
		{"excess separator-like cells", "a | b\n--- | ---\n- | - | -\n", "Table row"},
		{"code with strong", "**`private`**", "Code combined"},
		{"heading in list", "- first\n\n    # private\n", "Heading inside list Item"},
		{"quote in list", "- first\n\n    > private\n", "BlockQuote inside list Item"},
		{"heading in quote", "before\n\n> # private\n", "Heading inside BlockQuote"},
		{"rule in quote", "before\n\n> ---\n", "HorizontalRule inside BlockQuote"},
		{"table in quote", "> a | b\n> --- | ---\n> private | value\n", "Table inside BlockQuote"},
		{"nested quote", "before\n\n> quote\n> next\n>\n> > **private**\n>\n> 9. item\n", "BlockQuote inside BlockQuote"},
		{"ordered start overflow", "999999999999999999999. private", "List start"},
		{"deep quote", strings.Repeat("> ", 100) + "private", "nesting depth"},
		{"deep list", deepADFList(), "nesting depth"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ToADF(tt.markdown)
			if doc != nil || err == nil || !strings.Contains(err.Error(), tt.format) {
				t.Fatalf("ToADF = (%#v, %v); want nil document and %q error", doc, err, tt.format)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "example.invalid") {
				t.Fatalf("error leaked content: %v", err)
			}
		})
	}
}

func deepADFList() string {
	var input strings.Builder
	for i := 0; i < 30; i++ {
		input.WriteString(strings.Repeat("    ", i) + "- private\n")
	}
	return input.String()
}

func TestToADFLinkEscapes(t *testing.T) {
	doc, err := ToADF(`[a\]b](https://example.invalid/a\(b\)?x=1&amp;y=2 "a\"b \&quot; \\")`)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) != 1 || doc.Content[0].NodeType != adf.NodeParagraph {
		t.Fatalf("invalid link paragraph: %#v", doc)
	}
	var label strings.Builder
	for _, n := range doc.Content[0].Content {
		if n.NodeType != adf.ChildNodeText || len(n.Marks) != 1 || n.Marks[0].MarkType != adf.MarkLink {
			t.Fatalf("invalid link text: %#v", n)
		}
		label.WriteString(n.Text)
		want := map[string]any{"href": "https://example.invalid/a(b)?x=1&y=2", "title": "a\"b &quot; \\"}
		if !reflect.DeepEqual(n.Marks[0].Attributes, want) {
			t.Errorf("link attributes = %#v, want %#v", n.Marks[0].Attributes, want)
		}
	}
	if label.String() != "a]b" {
		t.Errorf("link label = %q", label.String())
	}
}

func TestToADFAllowedLinks(t *testing.T) {
	tests := []struct{ name, markdown, href string }{
		{"https", "[label](https://example.invalid/route)", "https://example.invalid/route"},
		{"http", "[label](http://example.invalid/route)", "http://example.invalid/route"},
		{"uppercase scheme", "[label](HTTPS://example.invalid/route)", "HTTPS://example.invalid/route"},
		{"mailto", "[label](mailto:user@example.invalid?subject=hello%20world)", "mailto:user@example.invalid?subject=hello%20world"},
		{"uppercase mailto", "[label](MAILTO:user@example.invalid)", "MAILTO:user@example.invalid"},
		{"mailto recipients", "[label](mailto:first@example.invalid,second@example.invalid)", "mailto:first@example.invalid,second@example.invalid"},
		{"percent encoding", "[label](https://example.invalid/a%20b/%2F/%25/%E4%B8%96?q=%23value#fragment)", "https://example.invalid/a%20b/%2F/%25/%E4%B8%96?q=%23value#fragment"},
		{"Unicode path", "[label](https://example.invalid/世界/café)", "https://example.invalid/世界/café"},
		{"IPv6 and port", "[label](http://[::1]:8080/route)", "http://[::1]:8080/route"},
		{"entity decoding", "[label](https&#58;//example.invalid/?a=1&amp;b=2)", "https://example.invalid/?a=1&b=2"},
		{"https autolink", "https://example.invalid/route", "https://example.invalid/route"},
		{"http autolink", "http://example.invalid/route", "http://example.invalid/route"},
		{"mailto autolink", "<user@example.invalid>", "mailto:user@example.invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ToADF(tt.markdown)
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.Content) != 1 || doc.Content[0].NodeType != adf.NodeParagraph || len(doc.Content[0].Content) == 0 {
				t.Fatalf("invalid link paragraph: %#v", doc)
			}
			for _, n := range doc.Content[0].Content {
				if n.NodeType != adf.ChildNodeText || len(n.Marks) != 1 || n.Marks[0].MarkType != adf.MarkLink {
					t.Fatalf("invalid link text: %#v", n)
				}
				if got := n.Marks[0].Attributes.(map[string]any)["href"]; got != tt.href {
					t.Errorf("href = %q, want %q", got, tt.href)
				}
			}
		})
	}
}

func TestToADFRejectedLinks(t *testing.T) {
	tests := []struct{ name, href string }{
		{"javascript", "javascript:private"},
		{"mixed case javascript", "JaVaScRiPt:private"},
		{"entity scheme", "javascript&colon;private"},
		{"data", "DaTa:text/plain,private"},
		{"file", "FiLe:///private"},
		{"unknown scheme", "custom:private"},
		{"ftp", "ftp://example.invalid/private"},
		{"relative", "relative/private"},
		{"root relative", "/private"},
		{"protocol relative", "//example.invalid/private"},
		{"fragment only", "#private"},
		{"empty host", "https://"},
		{"missing host", "https:/private"},
		{"opaque HTTP", "http:private"},
		{"port without host", "http://:80/private"},
		{"malformed host", "https://[private"},
		{"malformed port", "https://example.invalid:private/"},
		{"userinfo", "https://user:private@example.invalid/"},
		{"encoded userinfo", "https://%75ser:private@example.invalid/"},
		{"invalid path escape", "https://example.invalid/private%GG"},
		{"invalid query escape", "https://example.invalid/?private=%GG"},
		{"invalid fragment escape", "https://example.invalid/#private%GG"},
		{"invalid opaque escape", "mailto:user%GG@example.invalid"},
		{"encoded scheme", "%68ttps://example.invalid/private"},
		{"encoded controls", "https://example.invalid/private%0D%0A"},
		{"encoded NUL", "https://example.invalid/private%00"},
		{"encoded query control", "https://example.invalid/?private=%09"},
		{"encoded DEL", "https://example.invalid/private%7F"},
		{"encoded C1 control", "https://example.invalid/private%C2%85"},
		{"raw control", "https://example.invalid/private\x01"},
		{"raw Unicode control", "https://example.invalid/private\u0085"},
		{"entity control", "https://example.invalid/private&Tab;"},
		{"raw format character", "https://example.invalid/private\u202e"},
		{"encoded format character", "https://example.invalid/private%E2%80%8B"},
		{"invalid encoded UTF8", "https://example.invalid/private%FF"},
		{"raw space", "https://example.invalid/private route"},
		{"mailto host", "mailto://user@example.invalid/private"},
		{"mailto recipient", "mailto:private"},
		{"mailto without recipient", "mailto:?subject=private"},
		{"mailto fragment", "mailto:user@example.invalid#private"},
		{"mailto header control", "mailto:user@example.invalid?subject=private%0a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ToADF("private-body\n\n[private-label](" + tt.href + ")")
			if doc != nil || err == nil || !strings.Contains(err.Error(), "Link") {
				t.Fatalf("ToADF = (%#v, %v); want nil document and Link error", doc, err)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "example.invalid") || strings.Contains(err.Error(), tt.href) {
				t.Fatalf("error leaked content: %v", err)
			}
		})
	}
}

func TestToADFListMetadata(t *testing.T) {
	tests := []struct {
		name, markdown string
		orders         []int
	}{
		{"tab indentation", "5. first\n\t- child\n\t\t2. grandchild\n6. last", []int{5, 2}},
		{"zero start and empty item", "0. \n1. last\n", []int{0}},
		{"pipe ending", "4. value |\n5. other\\|\n", []int{4}},
		{"fenced code in list", "6. first\n\n    ```\n    91. not a list\n    ```\n\n7. last\n", []int{6}},
		{"indented code in list", "6. first\n\n        91. not a list\n\n7. last\n", []int{6}},
		{"quoted list with inline code", "> 8. `value`\n> 9. next\n", []int{8}},
		{"table header resembling ordered item", "7. header | b |\n--- | --- |\na | b |\n\n2. item\n", []int{2}},
		{"code and table", "```\n| a | b | c |\n| - | - | - |\n```\n\na | b\n--- | ---\nx | y\n\n3. item\n", []int{3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ToADF(tt.markdown)
			if err != nil {
				t.Fatal(err)
			}
			var orders []int
			var visit func(*adf.Node)
			visit = func(n *adf.Node) {
				if n.NodeType == adf.NodeOrderedList {
					orders = append(orders, n.Attributes.(map[string]any)["order"].(int))
				}
				if n.NodeType == adf.ChildNodeListItem && (len(n.Content) == 0 || n.Content[0].NodeType != adf.NodeParagraph) {
					t.Fatalf("list item does not start with paragraph: %#v", n)
				}
				for _, child := range n.Content {
					if n.NodeType == adf.NodeCodeBlock && (child.NodeType != adf.ChildNodeText || len(child.Marks) != 0) {
						t.Fatalf("invalid code block child: %#v", child)
					}
					visit(child)
				}
			}
			for _, n := range doc.Content {
				visit(n)
			}
			data, err := json.Marshal(doc)
			if !reflect.DeepEqual(orders, tt.orders) {
				t.Fatalf("orders = %v, want %v; ADF = %s", orders, tt.orders, data)
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `\u0000`) {
				t.Fatalf("metadata leaked: %s", data)
			}
		})
	}
}

func TestToADFMarkdownTranslator(t *testing.T) {
	doc, err := ToADF("plain\n\n- item\n")
	if err != nil {
		t.Fatal(err)
	}
	got := adf.NewTranslator(doc, adf.NewMarkdownTranslator()).Translate()
	if got != "plain\n\n- item\n" {
		t.Fatalf("translated Markdown = %q", got)
	}
}
