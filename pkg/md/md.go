package md

import (
	cf "github.com/kentaro-m/blackfriday-confluence"
	bf "github.com/russross/blackfriday/v2"
)

// ToJiraMD translates CommonMark to Jira flavored markdown.
func ToJiraMD(jfm string) string {
	if jfm == "" {
		return jfm
	}

	renderer := &cf.Renderer{Flags: cf.IgnoreMacroEscaping}
	r := bf.New(bf.WithRenderer(renderer), bf.WithExtensions(bf.CommonExtensions))

	return string(renderer.Render(r.Parse([]byte(jfm))))
}
