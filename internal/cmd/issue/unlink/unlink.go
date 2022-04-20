package unlinkParams

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	helpText     = `Unlink disconnects two issues from each other, if already connected.`
	examples     = `$ jira issue unlink ISSUE-1 ISSUE-2`
	optionCancel = "Cancel"
)

// NewCmdUnlink is an unlink command.
func NewCmdUnlink() *cobra.Command {
	cmd := cobra.Command{
		Use:     "unlink INWARD_ISSUE_KEY OUTWARD_ISSUE_KEY",
		Short:   "Unlink disconnects two issues from each other",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"ln"},
		Annotations: map[string]string{
			"help:args": "INWARD_ISSUE_KEY\tIssue key of the source issue, eg: ISSUE-1\n" +
				"OUTWARD_ISSUE_KEY\tIssue key of the target issue, eg: ISSUE-2.",
		},
		Run: unlink,
	}

	cmd.Flags().Bool("web", false, "Open inward issue in web browser after successful unlinking")

	return &cmd
}

func unlink(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	params := parseArgsAndFlags(cmd.Flags(), args, project)
	client := api.Client(jira.Config{Debug: params.debug})
	uc := unlinkCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(uc.setInwardIssueKey(project))
	cmdutil.ExitIfError(uc.setOutwardIssueKey(project))

	// TODO: Why is this being compared with optionCancel?!
	if uc.params.outwardIssueKey == optionCancel {
		cmdutil.Fail("Action aborted")
		os.Exit(0)
	}

	err := func() error {
		s := cmdutil.Info("Unlinking issues")
		defer s.Stop()

		return client.UnlinkIssue(uc.params.inwardIssueKey, uc.params.outwardIssueKey)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Issues unlinked")
	fmt.Printf("%s/browse/%s\n", server, uc.params.inwardIssueKey)

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, uc.params.inwardIssueKey)
		cmdutil.ExitIfError(err)
	}
}

type unlinkParams struct {
	inwardIssueKey  string
	outwardIssueKey string
	debug           bool
}

func parseArgsAndFlags(flags query.FlagParser, args []string, project string) *unlinkParams {
	var inwardIssueKey, outwardIssueKey string

	nargs := len(args)
	if nargs >= 1 {
		inwardIssueKey = cmdutil.GetJiraIssueKey(project, args[0])
	}
	if nargs >= 2 {
		outwardIssueKey = cmdutil.GetJiraIssueKey(project, args[1])
	}

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	return &unlinkParams{
		inwardIssueKey:  inwardIssueKey,
		outwardIssueKey: outwardIssueKey,
		debug:           debug,
	}
}

type unlinkCmd struct {
	client *jira.Client
	params *unlinkParams
}

func (uc *unlinkCmd) setInwardIssueKey(project string) error {
	if uc.params.inwardIssueKey != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "inwardIssueKey",
		Prompt:   &survey.Input{Message: "Inward issue key"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	uc.params.inwardIssueKey = cmdutil.GetJiraIssueKey(project, ans)

	return nil
}

func (uc *unlinkCmd) setOutwardIssueKey(project string) error {
	if uc.params.outwardIssueKey != "" {
		return nil
	}

	var ans string

	qs := &survey.Question{
		Name:     "outwardIssueKey",
		Prompt:   &survey.Input{Message: "Outward issue key"},
		Validate: survey.Required,
	}
	if err := survey.Ask([]*survey.Question{qs}, &ans); err != nil {
		return err
	}
	uc.params.outwardIssueKey = cmdutil.GetJiraIssueKey(project, ans)

	return nil
}
