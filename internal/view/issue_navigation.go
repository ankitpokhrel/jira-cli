package view

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const navigationInputLimit = 256

var (
	navigationKeyPattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*-[1-9][0-9]*$`)
	navigationNumberPattern = regexp.MustCompile(`^[1-9][0-9]*$`)
	errNavigationInput      = errors.New("enter a full issue key like ABC-123 or a positive number without leading zeros")
	errNavigationProject    = errors.New("cannot infer the current project; enter a full issue key like ABC-123")
)

func normalizeNavigationKey(input, currentKey string) (string, error) {
	if len(input) > navigationInputLimit || strings.IndexFunc(input, unicode.IsControl) >= 0 {
		return "", errNavigationInput
	}
	key := strings.TrimSpace(input)
	if navigationNumberPattern.MatchString(key) {
		if !validNavigationKey(currentKey) {
			return "", errNavigationProject
		}
		project, _, _ := strings.Cut(currentKey, "-")
		key = project + "-" + key
	}
	if !validNavigationKey(key) {
		return "", errNavigationInput
	}
	return strings.ToUpper(key), nil
}

func validNavigationKey(key string) bool {
	if len(key) > navigationInputLimit || !navigationKeyPattern.MatchString(key) {
		return false
	}
	_, number, _ := strings.Cut(key, "-")
	_, err := strconv.ParseUint(number, 10, 64)
	return err == nil
}
