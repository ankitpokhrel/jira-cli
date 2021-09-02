package adf

import (
	"fmt"
	"strings"
)

const (
	bgColorInfo    = "#deebff"
	bgColorNote    = "#eae6ff"
	bgColorError   = "#ffebe6"
	bgColorSuccess = "#e3fcef"
	bgColorWarning = "#fffae6"

	panelTypeInfo    = "info"
	panelTypeNote    = "note"
	panelTypeError   = "error"
	panelTypeSuccess = "success"
	panelTypeWarning = "warning"
)

// JiraMarkdownTranslator is a jira markdown translator.
type JiraMarkdownTranslator struct {
	*MarkdownTranslator
}

// NewJiraMarkdownTranslator constructs jira markdown translator.
func NewJiraMarkdownTranslator() *JiraMarkdownTranslator {
	openHooks := nodeTypeHook{
		NodePanel: nodePanelOpenHook,
	}

	closeHooks := nodeTypeHook{
		NodePanel: nodePanelCloseHook,
	}

	return &JiraMarkdownTranslator{
		MarkdownTranslator: NewMarkdownTranslator(
			WithMarkdownOpenHooks(openHooks),
			WithMarkdownCloseHooks(closeHooks),
		),
	}
}

// Open implements TagOpener interface.
func (tr *JiraMarkdownTranslator) Open(n Connector, d int) string {
	return tr.MarkdownTranslator.Open(n, d)
}

// Close implements TagCloser interface.
func (tr *JiraMarkdownTranslator) Close(n Connector) string {
	return tr.MarkdownTranslator.Close(n)
}

func nodePanelOpenHook(n Connector) string {
	attrs := n.GetAttributes()

	var tag strings.Builder

	tag.WriteString("\n{panel")
	if attrs != nil {
		a := attrs.(map[string]interface{})
		if len(a) > 0 {
			tag.WriteString(":")
		}
		for k, v := range a {
			if k == "panelType" {
				switch v {
				case panelTypeInfo:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorInfo))
				case panelTypeNote:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorNote))
				case panelTypeError:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorError))
				case panelTypeSuccess:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorSuccess))
				case panelTypeWarning:
					tag.WriteString(fmt.Sprintf("bgColor=%s", bgColorWarning))
				}
			} else {
				tag.WriteString(fmt.Sprintf("|%s=%s", k, v))
			}
		}
	}
	tag.WriteString("}\n")

	return tag.String()
}

func nodePanelCloseHook(Connector) string {
	return "{panel}\n"
}
