package filter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter"
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter/issue"
)

func TestCollectionGet(t *testing.T) {
	cltn := filter.Collection{issue.NewNumCommentsFilter(5)}
	assert.Equal(t, uint(5), cltn.Get(cltn[0].Key()))
	assert.Nil(t, cltn.Get("unknown"))
}

func TestCollectionGetInt(t *testing.T) {
	cltn := filter.Collection{issue.NewNumCommentsFilter(5)}
	assert.Equal(t, 5, cltn.GetInt(cltn[0].Key()))
	assert.Equal(t, 0, cltn.GetInt("unknown"))
}
