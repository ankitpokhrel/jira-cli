package jira

const (
	customFieldFormatOption  = "option"
	customFieldFormatArray   = "array"
	customFieldFormatNumber  = "number"
	customFieldFormatProject = "project"
	customFieldFormatUser    = "user"
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

type customFieldTypeUser struct {
	Name      *string `json:"name,omitempty"`      // For local (Server/DC) installation.
	AccountID *string `json:"accountId,omitempty"` // For cloud installation.
}

type customFieldTypeUserSet struct {
	Set customFieldTypeUser `json:"set"`
}

// newCustomFieldTypeUser creates a user field value appropriate for the installation type.
func newCustomFieldTypeUser(val, installationType string) customFieldTypeUser {
	if installationType == InstallationTypeLocal {
		return customFieldTypeUser{Name: &val}
	}
	return customFieldTypeUser{AccountID: &val}
}
