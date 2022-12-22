// Package jql is a very simple JQL query builder that cannot do a lot at the moment.
//
// There is no JQL syntax check and relies on the package user to construct a valid query.
//
// It cannot combine AND and OR query currently. That means you cannot construct a query like the one below:
// project="JQL" AND issue in openSprints() AND (type="Story" OR resolution="Done")
package jql
