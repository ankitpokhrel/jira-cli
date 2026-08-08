package edit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBodyForEdit(t *testing.T) {
	t.Parallel()

	markdown := "## Objective\n\n- item\n- [ ] criterion\n"
	jiraWiki := "h2. Objective\n\n* item\n* [ ] criterion\n"
	jiraWikiInline := "*important* [documentation|https://example.com]"
	converted := "h2. Objective\n* item\n* \\[ \\] criterion\n\n"

	tests := []struct {
		name         string
		body         string
		originalBody string
		isADF        bool
		expected     string
	}{
		{
			name:     "converts Markdown for Jira Server",
			body:     markdown,
			expected: converted,
		},
		{
			name:     "continues converting Markdown for Jira Cloud",
			body:     markdown,
			isADF:    true,
			expected: converted,
		},
		{
			name:     "preserves Jira Wiki Markup for Jira Server",
			body:     jiraWiki,
			expected: jiraWiki,
		},
		{
			name:     "preserves inline Jira Wiki Markup for Jira Server",
			body:     jiraWikiInline,
			expected: jiraWikiInline,
		},
		{
			name:         "omits an unchanged Jira Server description",
			body:         jiraWiki,
			originalBody: jiraWiki,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, bodyForEdit(tt.body, tt.originalBody, tt.isADF))
		})
	}
}
