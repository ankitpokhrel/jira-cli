package view

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestBoardRender(t *testing.T) {
	var b bytes.Buffer

	data := []*jira.Board{
		{ID: 1, Name: "First", Type: "scrum"},
		{ID: 2, Name: "[2] Second", Type: "kanban"},
		{ID: 3, Name: "Third", Type: "nextgen"},
	}
	board := NewBoard(data, WithBoardWriter(&b))
	assert.NoError(t, board.Render())

	expected := `ID	NAME	TYPE
1	First	scrum
2	[2[] Second	kanban
3	Third	nextgen
`
	assert.Equal(t, expected, b.String())
}
