package jira

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	customFieldFormatAny     = "any"
	customFieldFormatOption  = "option"
	customFieldFormatArray   = "array"
	customFieldFormatNumber  = "number"
	customFieldFormatString  = "string"
	customFieldFormatProject = "project"
	customFieldFormatJson    = "json"
)

type customField map[string]interface{}

type customFieldTypeNumber float64

type customFieldTypeString string

type customFieldTypeOption struct {
	Value string `json:"value"`
}

type customFieldTypeProject struct {
	Value string `json:"key"`
}

type customFieldTypeJson struct {
	Json string
}

func (field customFieldTypeJson) MarshalJSON() ([]byte, error) {
	return []byte(field.Json), nil
}

type customFieldEditTypeSet struct {
	Set any `json:"set"`
}

type customFieldEditTypeAddRemove struct {
	Add    *any `json:"add,omitempty"`
	Remove *any `json:"remove,omitempty"`
}

func constructCustomField(dataType string, itemType string, value string) any {
	switch dataType {
	case customFieldFormatOption:
		return customFieldTypeOption{Value: value}
	case customFieldFormatProject:
		return customFieldTypeProject{Value: value}
	case customFieldFormatArray:
		pieces := strings.Split(strings.TrimSpace(value), ",")
		items := make([]any, len(pieces))
		for idx, piece := range pieces {
			items[idx] = constructCustomField(itemType, "", piece)
		}
		return items
	case customFieldFormatNumber:
		num, err := strconv.ParseFloat(value, 64) //nolint:gomnd
		if err != nil {
			// Let Jira API handle data type error for now.
			return value
		} else {
			return customFieldTypeNumber(num)
		}
	case customFieldFormatAny:
		fallthrough
	case customFieldFormatString:
		return customFieldTypeString(value)
	case customFieldFormatJson:
		return customFieldTypeJson{Json: value}
	default:
		// An unknown type like "version" or "user". Try parsing as JSON,
		// and if that doesn't work, just send as a string
		var unused any
		err := json.Unmarshal([]byte(value), &unused)
		if err == nil {
			return customFieldTypeJson{Json: value}
		} else {
			return value
		}
	}
}
