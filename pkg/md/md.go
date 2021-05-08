package md

import (
	cf "github.com/kentaro-m/blackfriday-confluence"
	bf "github.com/russross/blackfriday/v2"
)

// JiraToGithubFlavored translates Jira flavored markdown to Github flavored markdown.
func JiraToGithubFlavored(jfm string) string {
	renderer := &cf.Renderer{Flags: cf.IgnoreMacroEscaping}
	r := bf.New(bf.WithRenderer(renderer), bf.WithExtensions(bf.CommonExtensions))

	return string(renderer.Render(r.Parse([]byte(jfm))))
}
