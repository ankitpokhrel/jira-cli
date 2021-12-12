package issue

import (
	"github.com/ankitpokhrel/jira-cli/pkg/jira/filter"
)

// KeyIssueNumComments is a filter key for issue comments.
const KeyIssueNumComments = filter.Key("issue-num-comments")

// NumCommentsFilter is a filter for issue comments.
type NumCommentsFilter struct {
	key   filter.Key
	value uint
}

// NewNumCommentsFilter constructs a filter to limit number of comments.
func NewNumCommentsFilter(value uint) NumCommentsFilter {
	return NumCommentsFilter{
		key:   KeyIssueNumComments,
		value: value,
	}
}

// Key returns key of this filter.
func (ncf NumCommentsFilter) Key() filter.Key {
	return ncf.key
}

// Val returns value of this filter.
func (ncf NumCommentsFilter) Val() interface{} {
	return ncf.value
}
