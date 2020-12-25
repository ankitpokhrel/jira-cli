package adf

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestADF(t *testing.T) {
	data, err := ioutil.ReadFile("./testdata/md.json")
	assert.NoError(t, err)

	var adf ADF
	err = json.Unmarshal(data, &adf)
	assert.NoError(t, err)

	tr := NewTranslator(&adf, &MarkdownTranslator{})

	expected := "#H1\n##H2\n> Blockquote text\n\nImplement epic browser\n```\nPanel paragraph\n```\n```\n**Strong and underlined** Paragraph 1\nParagraph 2\n```\n**Bold Text**\n_Italic Text_\nUnderlined Text\n~Strikethrough text~\n[Link](https://ankit.pl)\n- Unordered list item 1\n\t- Next\n\t\t- Another\n\t\t\t- New level\n- Unordered list item 2\n- Unordered list item 3\n1. Ordered list item 1\n2. Ordered list item 2\n3. Ordered list item 3\n\t4. nested\n\t\t5. second level\n\t\t\t6. third level\n\t\t\t\t7. fourth level\n\n**Table Header 1** | **Table Header 2** | **Table Header 3**\n--- | --- | ---\nTable row 1 column 1 | Table row 1 column 2 | Table row 1 column 3\nTable row 2 column 1 | Table row 2 column 2 | Table row 2 column 3\n```go\npackage main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```\n\n**Table Header 1** | **Table Header 2** | **Table Header 3** | **Table Header 4** | **Table Header 5**\n--- | --- | --- | --- | ---\nTable row 1 column 1 | Table row 2 column 1 | Table row 3 column 1 | Table row 4 column 1 | Table row 5 column 1\nTable row 1 column 2 | Table row 2 column 2 | Table row 3 column 2 | Table row 4 column 2 | Table row 5 column 2\nTable row 1 column 2 | Table row 2 column 3 | Table row 3 column 3 | Table row 4 column 3 | Table row 5 column 3\n"
	assert.Equal(t, expected, tr.Translate())
}
