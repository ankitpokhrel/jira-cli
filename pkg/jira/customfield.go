package jira

const (
	customFieldFormatOption = "option"
	customFieldFormatArray  = "array"
	customFieldFormatNumber = "number"
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
