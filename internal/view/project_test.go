package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestProjectRender(t *testing.T) {
	var b bytes.Buffer

	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}},
		{Key: "SCND", Name: "Second", Lead: lead{Name: "Person B"}},
		{Key: "THIRD", Name: "Third", Lead: lead{Name: "Person C"}},
	}

	board := NewProject(data, WithProjectWriter(&b))

	assert.NoError(t, board.Render())

	expected := `KEY	NAME	LEAD	
FRST	First	Person A
SCND	Second	Person B
THIRD	Third	Person C
`

	assert.Equal(t, expected, b.String())
}
