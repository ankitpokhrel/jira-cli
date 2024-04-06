package md

import (
	cf "github.com/kentaro-m/blackfriday-confluence"
	bf "github.com/russross/blackfriday/v2"

	"github.com/ankitpokhrel/jira-cli/pkg/md/jirawiki"
)

// ToJiraMD translates CommonMark to Jira flavored markdown.
func ToJiraMD(md string) string {
	if md == "" {
		return md
	}

	renderer := &cf.Renderer{Flags: cf.IgnoreMacroEscaping}
	r := bf.New(bf.WithRenderer(renderer), bf.WithExtensions(bf.CommonExtensions))

	return string(renderer.Render(r.Parse([]byte(md))))
}

// FromJiraMD translates Jira flavored markdown to CommonMark.
func FromJiraMD(jfm string) string {
	return jirawiki.Parse(jfm)
}
