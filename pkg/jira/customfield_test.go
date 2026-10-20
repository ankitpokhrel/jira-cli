package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCustomFieldTypeUserCloud(t *testing.T) {
	accountID := "5f7e1b2c3d4e5f6a7b8c9d0e"
	user := newCustomFieldTypeUser(accountID, InstallationTypeCloud)

	assert.Nil(t, user.Name)
	assert.NotNil(t, user.AccountID)
	assert.Equal(t, accountID, *user.AccountID)
}

func TestNewCustomFieldTypeUserLocal(t *testing.T) {
	username := "john.doe"
	user := newCustomFieldTypeUser(username, InstallationTypeLocal)

	assert.NotNil(t, user.Name)
	assert.Nil(t, user.AccountID)
	assert.Equal(t, username, *user.Name)
}

func TestNewCustomFieldTypeUserDefaultIsCloud(t *testing.T) {
	accountID := "5f7e1b2c3d4e5f6a7b8c9d0e"
	user := newCustomFieldTypeUser(accountID, "")

	assert.Nil(t, user.Name)
	assert.NotNil(t, user.AccountID)
	assert.Equal(t, accountID, *user.AccountID)
}
