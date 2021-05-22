package cmdcommon

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

const (
	// ActionSubmit is a submit action.
	ActionSubmit = "Submit"
	// ActionCancel is a cancel action.
	ActionCancel = "Cancel"
	// ActionMetadata is an add metadata action.
	ActionMetadata = "Add metadata"
)

// SetCreateFlags sets flags supported by create command.
func SetCreateFlags(cmd *cobra.Command, prefix string) {
	cmd.Flags().SortFlags = false

	if prefix == "Epic" {
		cmd.Flags().StringP("name", "n", "", "Epic name")
	} else {
		cmd.Flags().StringP("type", "t", "", "Issue type")
		cmd.Flags().StringP("parent", "P", "", "Parent issue key (Valid only if issue type is sub-task)")
	}
	cmd.Flags().StringP("summary", "s", "", prefix+" summary or title")
	cmd.Flags().StringP("body", "b", "", prefix+" description")
	cmd.Flags().StringP("priority", "y", "", prefix+" priority")
	cmd.Flags().StringP("assignee", "a", "", prefix+" assignee (email or display name)")
	cmd.Flags().StringArrayP("label", "l", []string{}, prefix+" labels")
	cmd.Flags().StringArrayP("component", "C", []string{}, prefix+" components")
	cmd.Flags().StringP("template", "T", "", "Path to a file to read body/description from")
	cmd.Flags().Bool("web", false, "Open in web browser after successful creation")
	cmd.Flags().Bool("no-input", false, "Disable prompt for non-required fields")
}

// GetNextAction provide user an option to select next action.
func GetNextAction() *survey.Question {
	return &survey.Question{
		Name: "action",
		Prompt: &survey.Select{
			Message: "What's next?",
			Options: []string{
				ActionSubmit,
				ActionMetadata,
				ActionCancel,
			},
		},
		Validate: survey.Required,
	}
}

// GetMetadata gathers a list of metadata users wants to add.
func GetMetadata() []*survey.Question {
	return []*survey.Question{
		{
			Name: "metadata",
			Prompt: &survey.MultiSelect{
				Message: "What would you like to add?",
				Options: []string{"Priority", "Components", "Labels"},
			},
		},
	}
}

// GetMetadataQuestions prepares metadata question to input from user.
func GetMetadataQuestions(cat []string) []*survey.Question {
	var qs []*survey.Question

	for _, c := range cat {
		switch c {
		case "Priority":
			qs = append(qs, &survey.Question{
				Name:   "priority",
				Prompt: &survey.Input{Message: "Priority"},
			})
		case "Components":
			qs = append(qs, &survey.Question{
				Name: "components",
				Prompt: &survey.Input{
					Message: "Components",
					Help:    "Comma separated list of valid components. For eg: BE,FE",
				},
			})
		case "Labels":
			qs = append(qs, &survey.Question{
				Name: "labels",
				Prompt: &survey.Input{
					Message: "Labels",
					Help:    "Comma separated list of labels. For eg: backend,urgent",
				},
			})
		}
	}

	return qs
}
