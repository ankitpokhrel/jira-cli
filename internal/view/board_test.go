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
		{ID: 1, Name: "first", Type: "scrum"},
		{ID: 2, Name: "second", Type: "kanban"},
		{ID: 3, Name: "third", Type: "nextgen"},
	}

	board := NewBoard(data, WithBoardWriter(&b))

	assert.NoError(t, board.Render())

	expected := `ID	NAME	TYPE	
1	first	scrum
2	second	kanban
3	third	nextgen
`

	assert.Equal(t, expected, b.String())
}
