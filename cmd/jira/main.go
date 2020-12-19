package main

import (
	"fmt"
	"os"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/root"
)

const jiraAPITokenLink = "https://id.atlassian.com/manage-profile/security/api-tokens"

func main() {
	checkForJiraToken()

	rootCmd := root.NewCmdRoot()
	if _, err := rootCmd.ExecuteC(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func checkForJiraToken() {
	if os.Getenv("JIRA_API_TOKEN") != "" {
		return
	}

	msg := fmt.Sprintf(`You need to define JIRA_API_TOKEN env for the tool to work. 

You can generate a token using this link: %s

After generating the token, export it to your shell and run 'jira init' if you haven't already.`, jiraAPITokenLink)

	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
}
