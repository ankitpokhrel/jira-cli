package jira

const (
	customFieldFormatOption = "option"
	customFieldFormatArray  = "array"
	customFieldFormatNumber = "number"
)

type customField map[string]interface{}

type customFieldTypeNumber float64

type customFieldTypeOption struct {
	Value string `json:"value"`
}
