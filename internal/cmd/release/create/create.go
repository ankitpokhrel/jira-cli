package create

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	helpText = `Create a version (release) in a project.`
	examples = `$ jira release create

# Create a version with minimal info
$ jira release create --name "Version 1.0"

# Create a version with full details
$ jira release create --name "v2.0" --description "Major release" --released --release-date "2024-12-31"

# Create in a specific project
$ jira release create -p PROJECT --name "v1.5"`
)

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	return &cobra.Command{
		Use:     "create",
		Short:   "Create a version in a project",
		Long:    helpText,
		Example: examples,
		Run:     create,
	}
}

// SetFlags sets flags supported by create command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Version name (required)")
	cmd.Flags().String("description", "", "Version description")
	cmd.Flags().Bool("released", false, "Mark version as released")
	cmd.Flags().Bool("archived", false, "Mark version as archived")
	cmd.Flags().String("release-date", "", "Release date (YYYY-MM-DD)")
	cmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().Bool("no-input", false, "Disable interactive prompts")
	cmd.Flags().Bool("debug", false, "Enable debug mode")
}

type createParams struct {
	Name        string
	Description string
	Released    bool
	Archived    bool
	ReleaseDate string
	StartDate   string
	NoInput     bool
	Debug       bool
}

func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")

	params := parseFlags(cmd.Flags())
	client := api.DefaultClient(params.Debug)
	cc := createCmd{
		client: client,
		params: params,
	}

	if cc.isNonInteractive() || params.NoInput || tui.IsDumbTerminal() {
		params.NoInput = true

		if params.Name == "" {
			cmdutil.Failed("Param `--name` is mandatory when using a non-interactive mode")
		}
	}

	qs := cc.getQuestions()
	if len(qs) > 0 {
		ans := struct{ Name, Description string }{}
		err := survey.Ask(qs, &ans)
		cmdutil.ExitIfError(err)

		if params.Name == "" {
			params.Name = ans.Name
		}
		if params.Description == "" {
			params.Description = ans.Description
		}
	}

	version, err := func() (*jira.ProjectVersion, error) {
		s := cmdutil.Info("Creating version...")
		defer s.Stop()

		req := &jira.VersionCreateRequest{
			Name:        params.Name,
			Description: params.Description,
			Project:     project,
			Archived:    params.Archived,
			Released:    params.Released,
			ReleaseDate: params.ReleaseDate,
			StartDate:   params.StartDate,
		}

		return client.CreateVersion(req)
	}()

	cmdutil.ExitIfError(err)
	cmdutil.Success(
		"Version created: %s (ID: %s)\n%s",
		version.Name,
		version.ID,
		cmdutil.GenerateServerBrowseURL(server, project),
	)
}

func (cc *createCmd) getQuestions() []*survey.Question {
	var qs []*survey.Question

	if cc.params.Name == "" {
		qs = append(qs, &survey.Question{
			Name:     "name",
			Prompt:   &survey.Input{Message: "Version name"},
			Validate: survey.Required,
		})
	}

	if cc.params.Description == "" && !cc.params.NoInput {
		qs = append(qs, &survey.Question{
			Name: "description",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Description (optional)",
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: true,
			},
		})
	}

	return qs
}

type createCmd struct {
	client *jira.Client
	params *createParams
}

func (cc *createCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData()
}

func parseFlags(flags query.FlagParser) *createParams {
	name, err := flags.GetString("name")
	cmdutil.ExitIfError(err)

	description, err := flags.GetString("description")
	cmdutil.ExitIfError(err)

	released, err := flags.GetBool("released")
	cmdutil.ExitIfError(err)

	archived, err := flags.GetBool("archived")
	cmdutil.ExitIfError(err)

	releaseDate, err := flags.GetString("release-date")
	cmdutil.ExitIfError(err)

	startDate, err := flags.GetString("start-date")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &createParams{
		Name:        name,
		Description: description,
		Released:    released,
		Archived:    archived,
		ReleaseDate: releaseDate,
		StartDate:   startDate,
		NoInput:     noInput,
		Debug:       debug,
	}
}
