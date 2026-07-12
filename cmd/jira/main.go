package main

import (
	"fmt"
	"os"

	"github.com/rethab/jira-cli/internal/cmd/root"
)

func main() {
	rootCmd := root.NewCmdRoot()
	if _, err := rootCmd.ExecuteC(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
