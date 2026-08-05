package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestComponentRender(t *testing.T) {
	var b bytes.Buffer

	data := []*jira.ProjectComponent{
		{ID: "10000", Name: "Backend", Description: "Core backend APIs"},
		{ID: "10001", Name: "[UI] Frontend", Description: "Web app"},
		{ID: "10002", Name: "Mobile", Description: nil},
	}
	component := NewComponent(data, WithComponentWriter(&b))
	assert.NoError(t, component.Render())

	expected := `ID	NAME	DESCRIPTION
10000	Backend	Core backend APIs
10001	[UI[] Frontend	Web app
10002	Mobile	
`
	assert.Equal(t, expected, b.String())
}
