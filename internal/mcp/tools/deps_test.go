package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeps_IssueURL(t *testing.T) {
	d := &Deps{Server: "https://example.atlassian.net"}
	assert.Equal(t, "https://example.atlassian.net/browse/TEST-1", d.IssueURL("TEST-1"))
}

func TestDeps_IssueURL_TrimsTrailingSlash(t *testing.T) {
	d := &Deps{Server: "https://example.atlassian.net/"}
	assert.Equal(t, "https://example.atlassian.net/browse/TEST-1", d.IssueURL("TEST-1"))
}

func TestDeps_ResolveProject_UsesDefaultWhenEmpty(t *testing.T) {
	d := &Deps{DefaultProject: "ABC"}
	assert.Equal(t, "ABC", d.ResolveProject(""))
}

func TestDeps_ResolveProject_PrefersExplicit(t *testing.T) {
	d := &Deps{DefaultProject: "ABC"}
	assert.Equal(t, "XYZ", d.ResolveProject("XYZ"))
}
