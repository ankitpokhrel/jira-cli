package completion

import (
	"os"

	"github.com/spf13/cobra"
)

const helpText = `Output shell completion code for the specified shell (bash or zsh). 
To load completions:

Bash:

$ source <(jira completion bash)

# To load completions for each session, execute once:
Linux:
  $ jira completion bash > /etc/bash_completion.d/jira
MacOS:
  $ jira completion bash > /usr/local/etc/bash_completion.d/jira

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ jira completion zsh > "${fpath[1]}/_jira"

# You will need to start a new shell for this setup to take effect.
`

// NewCmdCompletion is a completion command.
func NewCmdCompletion() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion",
		Short:                 "Output shell completion code for the specified shell (bash or zsh)",
		Long:                  helpText,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(os.Stdout)
			}
		},
	}
}
