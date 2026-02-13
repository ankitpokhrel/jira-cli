package jira

import "strings"

const (
	customFieldFormatOption    = "option"
	customFieldFormatArray     = "array"
	customFieldFormatNumber    = "number"
	customFieldFormatProject   = "project"
	customFieldFormatCascading = "option-with-child"
)

type customField map[string]interface{}

type customFieldTypeNumber float64

type customFieldTypeNumberSet struct {
	Set customFieldTypeNumber `json:"set"`
}

type customFieldTypeStringSet struct {
	Set string `json:"set"`
}

type customFieldTypeOption struct {
	Value string `json:"value"`
}

type customFieldTypeOptionSet struct {
	Set customFieldTypeOption `json:"set"`
}

type customFieldTypeOptionAddRemove struct {
	Add    *customFieldTypeOption `json:"add,omitempty"`
	Remove *customFieldTypeOption `json:"remove,omitempty"`
}

type customFieldTypeProject struct {
	Value string `json:"key"`
}

type customFieldTypeProjectSet struct {
	Set customFieldTypeProject `json:"set"`
}

type customFieldTypeCascadingChild struct {
	Value string `json:"value"`
}

type customFieldTypeCascading struct {
	Value string                         `json:"value"`
	Child *customFieldTypeCascadingChild `json:"child,omitempty"`
}

type customFieldTypeCascadingSet struct {
	Set customFieldTypeCascading `json:"set"`
}

// parseCascadingValue parses a cascading select value in "Parent->Child" format.
func parseCascadingValue(val string) customFieldTypeCascading {
	parts := strings.SplitN(val, "->", 2)
	parent := strings.TrimSpace(parts[0])
	cf := customFieldTypeCascading{Value: parent}
	if len(parts) == 2 {
		child := strings.TrimSpace(parts[1])
		cf.Child = &customFieldTypeCascadingChild{Value: child}
	}
	return cf
}
