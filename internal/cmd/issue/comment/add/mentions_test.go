package add

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestFindMentionUser(t *testing.T) {
	user := &jira.User{
		AccountID:   "a123b",
		DisplayName: "Person A",
		Email:       "person.a@example.com",
	}

	tests := []struct {
		name    string
		query   string
		users   []*jira.User
		want    *jira.User
		wantErr string
	}{
		{name: "display name", query: "person a", users: []*jira.User{user}, want: user},
		{name: "email", query: "person.a@example.com", users: []*jira.User{user}, want: user},
		{name: "no exact match", query: "person", users: []*jira.User{user}, wantErr: `mention query "person" did not match an exact user`},
		{name: "ambiguous match", query: "person a", users: []*jira.User{user, user}, wantErr: `mention query "person a" matches multiple users`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := findMentionUser(test.query, test.users)
			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Same(t, test.want, got)
		})
	}
}
