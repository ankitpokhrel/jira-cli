package edit

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
)

const (
	helpText = `Edit a version (release) in a project.

You can provide either version ID or version name as the argument.`
	examples = `$ jira release edit "Version 1.0"

# Edit version by ID
$ jira release edit 10000 --name "Version 1.0 Updated"

# Edit version by name
$ jira release edit "v1.0" --name "v1.1" --description "Updated version"

# Mark version as released
$ jira release edit 10000 --released

# Update release date
$ jira release edit "v1.0" --release-date "2024-12-31"

# Use --no-input to skip prompts
$ jira release edit 10000 --name "v2.0" --no-input`
)

// NewCmdEdit is an edit command.
func NewCmdEdit() *cobra.Command {
	cmd := cobra.Command{
		Use:     "edit VERSION-ID-OR-NAME",
		Short:   "Edit a version in a project",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"update", "modify", "rename"},
		Annotations: map[string]string{
			"help:args": `VERSION-ID-OR-NAME	Version ID or name, eg: 10000 or "Version 1.0"`,
		},
		Args: cobra.ExactArgs(1),
		Run:  edit,
	}

	setFlags(&cmd)

	return &cmd
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "New version name")
	cmd.Flags().String("description", "", "Version description")
	cmd.Flags().Bool("released", false, "Mark version as released")
	cmd.Flags().Bool("unreleased", false, "Mark version as unreleased")
	cmd.Flags().Bool("archived", false, "Mark version as archived")
	cmd.Flags().Bool("unarchived", false, "Mark version as unarchived")
	cmd.Flags().String("release-date", "", "Release date (YYYY-MM-DD)")
	cmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().Bool("no-input", false, "Disable interactive prompts")
	cmd.Flags().Bool("debug", false, "Enable debug mode")
}

type editParams struct {
	versionIDOrName string
	name            string
	description     string
	releasedFlag    bool
	unreleasedFlag  bool
	archivedFlag    bool
	unarchivedFlag  bool
	releaseDate     string
	startDate       string
	noInput         bool
	debug           bool
}

func edit(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")

	params := parseArgsAndFlags(cmd.Flags(), args)
	client := api.DefaultClient(params.debug)

	// Resolve version ID from name or ID
	versionID, currentVersion, err := resolveVersion(client, params.versionIDOrName, project)
	cmdutil.ExitIfError(err)

	ec := editCmd{
		client:         client,
		params:         params,
		currentVersion: currentVersion,
	}

	if !params.noInput {
		cmdutil.ExitIfError(ec.askQuestions())
	}

	// Build update request with only changed fields
	updateReq := &jira.VersionUpdateRequest{}
	hasChanges := false

	if params.name != "" && params.name != currentVersion.Name {
		updateReq.Name = params.name
		hasChanges = true
	}

	if params.description != "" {
		var currentDesc string
		if currentVersion.Description != nil {
			currentDesc = fmt.Sprintf("%v", currentVersion.Description)
		}
		if params.description != currentDesc {
			updateReq.Description = params.description
			hasChanges = true
		}
	}

	if params.releasedFlag {
		released := true
		updateReq.Released = &released
		hasChanges = true
	} else if params.unreleasedFlag {
		released := false
		updateReq.Released = &released
		hasChanges = true
	}

	if params.archivedFlag {
		archived := true
		updateReq.Archived = &archived
		hasChanges = true
	} else if params.unarchivedFlag {
		archived := false
		updateReq.Archived = &archived
		hasChanges = true
	}

	if params.releaseDate != "" {
		updateReq.ReleaseDate = params.releaseDate
		hasChanges = true
	}

	if params.startDate != "" {
		updateReq.StartDate = params.startDate
		hasChanges = true
	}

	if !hasChanges {
		cmdutil.Failed("No changes to apply")
	}

	err = func() error {
		s := cmdutil.Info("Updating version...")
		defer s.Stop()

		return client.UpdateVersion(versionID, updateReq)
	}()

	cmdutil.ExitIfError(err)

	displayName := params.name
	if displayName == "" {
		displayName = currentVersion.Name
	}

	cmdutil.Success("Version updated: %s (ID: %s)", displayName, versionID)
}

func (ec *editCmd) askQuestions() error {
	var qs []*survey.Question

	// Current version display
	var currentDesc string
	if ec.currentVersion.Description != nil {
		currentDesc = fmt.Sprintf("%v", ec.currentVersion.Description)
	}

	currentInfo := fmt.Sprintf("Current: %s (Released: %v, Archived: %v)",
		ec.currentVersion.Name,
		ec.currentVersion.Released,
		ec.currentVersion.Archived,
	)

	if ec.params.name == "" {
		qs = append(qs, &survey.Question{
			Name: "name",
			Prompt: &survey.Input{
				Message: fmt.Sprintf("Version name [%s]", currentInfo),
				Default: ec.currentVersion.Name,
			},
		})
	}

	if ec.params.description == "" {
		qs = append(qs, &survey.Question{
			Name: "description",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Description (optional)",
					Default:       currentDesc,
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: true,
			},
		})
	}

	if len(qs) == 0 {
		return nil
	}

	ans := struct {
		Name        string
		Description string
	}{}

	if err := survey.Ask(qs, &ans); err != nil {
		return err
	}

	if ec.params.name == "" {
		ec.params.name = ans.Name
	}
	if ec.params.description == "" {
		ec.params.description = ans.Description
	}

	return nil
}

type editCmd struct {
	client         *jira.Client
	params         *editParams
	currentVersion *jira.ProjectVersion
}

// resolveVersion resolves a version identifier (ID or name) to its ID and full version object.
// It first attempts to fetch by ID, and if that fails, searches by name in the project's versions.
func resolveVersion(client *jira.Client, versionIDOrName, project string) (string, *jira.ProjectVersion, error) {
	s := cmdutil.Info(fmt.Sprintf("Fetching version %s...", versionIDOrName))
	defer s.Stop()

	// Try to get by ID first
	if version, err := client.GetVersion(versionIDOrName); err == nil {
		return versionIDOrName, version, nil
	}

	// If that fails, search by name in project versions
	versions, err := client.Release(project)
	if err != nil {
		return "", nil, err
	}

	for _, v := range versions {
		if v.Name == versionIDOrName {
			return v.ID, v, nil
		}
	}

	return "", nil, fmt.Errorf("version not found: %s", versionIDOrName)
}

func parseArgsAndFlags(flags query.FlagParser, args []string) *editParams {
	name, err := flags.GetString("name")
	cmdutil.ExitIfError(err)

	description, err := flags.GetString("description")
	cmdutil.ExitIfError(err)

	releasedFlag, err := flags.GetBool("released")
	cmdutil.ExitIfError(err)

	unreleasedFlag, err := flags.GetBool("unreleased")
	cmdutil.ExitIfError(err)

	archivedFlag, err := flags.GetBool("archived")
	cmdutil.ExitIfError(err)

	unarchivedFlag, err := flags.GetBool("unarchived")
	cmdutil.ExitIfError(err)

	releaseDate, err := flags.GetString("release-date")
	cmdutil.ExitIfError(err)

	startDate, err := flags.GetString("start-date")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	// Check for conflicting flags
	if releasedFlag && unreleasedFlag {
		cmdutil.Failed("Cannot use both --released and --unreleased flags")
	}
	if archivedFlag && unarchivedFlag {
		cmdutil.Failed("Cannot use both --archived and --unarchived flags")
	}

	// Auto-enable no-input if any flags are provided beyond the version arg
	if name != "" || description != "" || releasedFlag || unreleasedFlag ||
		archivedFlag || unarchivedFlag || releaseDate != "" || startDate != "" {
		noInput = true
	}

	return &editParams{
		versionIDOrName: args[0],
		name:            name,
		description:     description,
		releasedFlag:    releasedFlag,
		unreleasedFlag:  unreleasedFlag,
		archivedFlag:    archivedFlag,
		unarchivedFlag:  unarchivedFlag,
		releaseDate:     releaseDate,
		startDate:       startDate,
		noInput:         noInput,
		debug:           debug,
	}
}
