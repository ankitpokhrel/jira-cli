package main

import (
	"fmt"
	"os"

	"github.com/ankitpokhrel/jira-cli/cmd/jira"
)

func main() {
	if err := jira.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
