package cmdcommon

import (
	"fmt"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// EnsureProject terminates the command with a helpful message when no project
// is configured or supplied via -p/--project.
func EnsureProject(project string) {
	if project == "" {
		cmdutil.Failed(
			"No project provided.\n" +
				"Set a default project by running 'jira init', or pass one with the -p/--project flag.",
		)
	}
}

// ResolveProjectType returns the Jira "style" (classic vs next-gen) for the
// effective project.
func ResolveProjectType(client *jira.Client, project, cachedType, installation string, overridden bool) (string, error) {
	if installation != jira.InstallationTypeCloud || project == "" {
		return cachedType, nil
	}
	if !overridden && cachedType != "" {
		return cachedType, nil
	}

	p, err := client.ProjectByKey(project)
	if err != nil {
		if overridden {
			return "", fmt.Errorf("unable to resolve type for project %q: %w", project, err)
		}
		return cachedType, nil
	}
	return p.Type, nil
}
