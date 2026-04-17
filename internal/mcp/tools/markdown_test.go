package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

func TestBodyToMarkdown_String(t *testing.T) {
	assert.Equal(t, "hello", bodyToMarkdown("hello"))
}

func TestBodyToMarkdown_Nil(t *testing.T) {
	assert.Equal(t, "", bodyToMarkdown(nil))
}

func TestBodyToMarkdown_ADF(t *testing.T) {
	doc := &adf.ADF{
		Version: 1,
		DocType: "doc",
		Content: []*adf.Node{
			{
				NodeType: "paragraph",
				Content: []*adf.Node{
					{NodeType: "text", NodeValue: adf.NodeValue{Text: "Hello world"}},
				},
			},
		},
	}
	got := bodyToMarkdown(doc)
	assert.Contains(t, got, "Hello world")
}

func TestBodyToMarkdown_UnknownType(t *testing.T) {
	assert.Equal(t, "", bodyToMarkdown(123))
}
