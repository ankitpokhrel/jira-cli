package completion

import (
	"os"

	"github.com/spf13/cobra"
)

const helpText = `Output shell completion code for the specified shell (bash, zsh, fish or PowerShell). 

To load completions:

Bash:

  $ source <(jira completion bash)

  # To load completions for each session, execute once:
  Linux:
    $ jira completion bash > /etc/bash_completion.d/jira
  macOS:
    $ jira completion bash > /usr/local/etc/bash_completion.d/jira

Zsh:

  # If shell completion is not already enabled in your environment you will need
  # to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ jira completion zsh > "${fpath[1]}/_jira"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ jira completion fish | source

  # To load completions for each session, execute once:
  $ jira completion fish > ~/.config/fish/completions/_jira.fish

PowerShell:

  PS> jira completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> jira completion powershell > _jira.ps1
  # and source this file from your PowerShell profile.
`

// NewCmdCompletion is a completion command.
func NewCmdCompletion() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion",
		Short:                 "Output shell completion code for the specified shell (bash or zsh)",
		Long:                  helpText,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				_ = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
		},
	}
}
