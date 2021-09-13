package man

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const (
	examples = `# Generate man pages in default location /tmp/man-jira-cli
$ jira man --generate

# Generate man pages in specified location
$ jira man --generate --output /path/to/man-pages`
)

// NewCmdMan is a man command.
func NewCmdMan() *cobra.Command {
	cmd := cobra.Command{
		Use:     "man",
		Short:   "Help generate man(7) pages for Jira CLI.",
		Long:    "Help generate man pages for Jira CLI compatible with UNIX style man pages.",
		Example: examples,
		RunE:    man,
	}

	cmd.Flags().BoolP("generate", "g", false, "Generate the man pages for Jira CLI")
	cmd.Flags().StringP("output", "o", "/tmp/man-jira-cli", "Name of the directory where the man pages would be generated")

	return &cmd
}

func man(cmd *cobra.Command, _ []string) error {
	gen, err := cmd.Flags().GetBool("generate")
	if err != nil {
		return err
	}

	if !gen {
		return cmd.Help()
	}

	out, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist.
	_, err = os.Stat(out)
	if err != nil {
		if err := os.MkdirAll(out, os.ModePerm); err != nil {
			return err
		}
	}

	header := &doc.GenManHeader{
		Title:   "Jira CLI",
		Section: "7",
	}

	// Generate man pages from the root.
	return doc.GenManTree(cmd.Parent(), header, out)
}
