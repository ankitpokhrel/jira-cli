// Package tools holds the individual MCP tool handlers. Each handler is
// a small adapter from a typed input/output struct onto the existing
// pkg/jira client. Handlers must not depend on cobra, viper, survey, or
// tui; their dependencies are injected via the Deps struct.
package tools
