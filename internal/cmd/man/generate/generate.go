package generate

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
)

const (
	examples = `$ jira man generate --output /path/to/man-pages

# Create the output directory if it does not exist
$ jira man generate --create --output /this/path/does-not-exist`
)

// NewCmdGenerate is a generate command.
func NewCmdGenerate() *cobra.Command {
	return &cobra.Command{
		Use:     "generate",
		Short:   "Generate man(7) pages for Jira CLI.",
		Long:    "Generate man(7) pages for the various commands used in Jira CLI compatible with UNIX man pages.",
		Example: examples,
		Aliases: []string{"gen"},
		Run:     generateManPages,
	}
}

func generateManPages(cmd *cobra.Command, _ []string) {
	createManLocation, err := cmd.Flags().GetBool("create")
	cmdutil.ExitIfError(err)

	manLocationName, err := cmd.Flags().GetString("output")
	cmdutil.ExitIfError(err)

	// Point to the current directory if no directory name is passed in.
	if manLocationName == "" {
		manLocationName = "."
	}

	// Check if the directory exists, based on the flag create or print an error.
	_, err = os.Stat(manLocationName)
	if os.IsNotExist(err) {
		if createManLocation {
			err = os.MkdirAll(manLocationName, os.ModePerm)
			if err != nil {
				cmdutil.ExitIfError(err)
			}
		} else {
			cmdutil.ExitIfError(err)
		}
	}

	header := &doc.GenManHeader{
		Title:   "Jira CLI",
		Section: "7",
	}

	// We are two command levels deep, so generate the pages from the root.
	err = doc.GenManTree(cmd.Parent().Parent(), header, manLocationName)
	if err != nil {
		cmdutil.ExitIfError(err)
	}
}

// SetFlags sets flags supported by a generate command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("output", "o", "", "Name of the directory where the man pages would be generated")
	cmd.Flags().BoolP("create", "r", false, "Create the directory if it does not exist")
}
