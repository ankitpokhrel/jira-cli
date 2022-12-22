package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestProjectRender(t *testing.T) {
	var b bytes.Buffer

	//nolint:unused
	type lead struct {
		Name string `json:"displayName"`
	}

	data := []*jira.Project{
		{Key: "FRST", Name: "First", Lead: lead{Name: "Person A"}, Type: jira.ProjectTypeClassic},
		{Key: "SCND", Name: "[2] Second", Lead: lead{Name: "Person B"}, Type: jira.ProjectTypeNextGen},
		{Key: "THIRD", Name: "Third", Lead: lead{Name: "Person C"}, Type: jira.ProjectTypeClassic},
	}
	project := NewProject(data, WithProjectWriter(&b))
	assert.NoError(t, project.Render())

	expected := `KEY	NAME	TYPE	LEAD
FRST	First	classic	Person A
SCND	⦗2⦘ Second	next-gen	Person B
THIRD	Third	classic	Person C
`
	assert.Equal(t, expected, b.String())
}
