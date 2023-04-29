package unlink

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Unlink disconnects two issues from each other, if already connected.`
	examples = `$ jira issue unlink ISSUE-1 ISSUE-2`
)

// NewCmdUnlink is an unlink command.
func NewCmdUnlink() *cobra.Command {
	cmd := cobra.Command{
		Use:     "unlink INWARD_ISSUE_KEY OUTWARD_ISSUE_KEY",
		Short:   "Unlink disconnects two issues from each other",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"uln"},
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
	client := api.DefaultClient(params.debug)
	uc := unlinkCmd{
		client: client,
		params: params,
	}

	cmdutil.ExitIfError(uc.setInwardIssueKey(project))
	cmdutil.ExitIfError(uc.setOutwardIssueKey(project))

	err := func() error {
		s := cmdutil.Info("Unlinking issues")
		defer s.Stop()

		linkID, err := client.GetLinkID(uc.params.inwardIssueKey, uc.params.outwardIssueKey)
		if err != nil {
			return err
		}

		return client.UnlinkIssue(linkID)
	}()
	cmdutil.ExitIfError(err)

	server := viper.GetString("server")

	cmdutil.Success("Issues unlinked")
	fmt.Printf("%s\n", cmdutil.GenerateServerURL(server, uc.params.inwardIssueKey))

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
