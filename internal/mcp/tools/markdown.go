package tools

import (
	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

// bodyToMarkdown renders a Jira description-or-comment body field to markdown.
// The body is interface{} because v3 returns *adf.ADF and v2 returns string.
func bodyToMarkdown(body any) string {
	switch v := body.(type) {
	case nil:
		return ""
	case string:
		return v
	case *adf.ADF:
		if v == nil {
			return ""
		}
		return adf.NewTranslator(v, adf.NewMarkdownTranslator()).Translate()
	default:
		return ""
	}
}
