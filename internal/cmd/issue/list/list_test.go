package list_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ankitpokhrel/jira-cli/internal/cmd/epic"
	issuecmd "github.com/ankitpokhrel/jira-cli/internal/cmd/issue"
	"github.com/ankitpokhrel/jira-cli/internal/cmd/sprint"
	"github.com/ankitpokhrel/jira-cli/internal/query"
)

// issueCommand uses the real issue command tree so flag checks include shared command wiring.
func issueCommand(t *testing.T, name string) *cobra.Command {
	t.Helper()
	cmd, _, err := issuecmd.NewCmdIssue().Find([]string{name})
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// TestCommentThreadFlags keeps native threads opt-in with explicit authentication, ordering, and whole-thread limit guidance.
func TestCommentThreadFlags(t *testing.T) {
	for _, name := range []string{"list", "view"} {
		t.Run(name, func(t *testing.T) {
			cmd := issueCommand(t, name)
			threads := cmd.Flags().Lookup("comment-threads")
			if threads == nil || threads.DefValue != "false" || threads.Value.Type() != "bool" {
				t.Fatalf("expected an opt-in boolean comment-threads flag, got %+v", threads)
			}
			for _, term := range []string{"experimental", "Jira Cloud", "API-token Basic auth"} {
				if !strings.Contains(threads.Usage, term) {
					t.Errorf("thread flag help is missing %q: %s", term, threads.Usage)
				}
			}
			comments := cmd.Flags().Lookup("comments")
			if comments == nil || comments.DefValue != "1" || !strings.Contains(comments.Usage, "N whole root threads with all replies") || !strings.Contains(comments.Usage, "0 hides comments") {
				t.Fatalf("comments default/help changed unexpectedly: %+v", comments)
			}
			for _, term := range []string{"newest-first", "oldest-first", "HAR", "top-level", "Read-only", "persisted", "unsupported"} {
				if !strings.Contains(cmd.Long, term) {
					t.Errorf("command help is missing %q", term)
				}
			}
			example := "jira issue " + name + " "
			if name == "view" {
				example += "ISSUE-1 "
			}
			if !strings.Contains(cmd.Example, example+"--comment-threads --comments 5") {
				t.Error("missing opt-in example")
			}
		})
	}
}

// TestCommentThreadFlagGroups rejects raw threaded output while preserving supported flat and hidden-comment combinations.
func TestCommentThreadFlagGroups(t *testing.T) {
	for _, name := range []string{"list", "view"} {
		t.Run(name, func(t *testing.T) {
			for _, tt := range []struct {
				name    string
				args    []string
				wantErr bool
			}{
				{name: "default"},
				{name: "flat", args: []string{"--comments", "5"}},
				{name: "threads", args: []string{"--comment-threads", "--comments", "5"}},
				{name: "hidden", args: []string{"--comment-threads", "--comments", "0"}},
				{name: "plain", args: []string{"--comment-threads", "--plain"}},
				{name: "raw", args: []string{"--raw"}},
				{name: "raw incompatible", args: []string{"--raw", "--comment-threads"}, wantErr: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					cmd := issueCommand(t, name)
					if err := cmd.ParseFlags(tt.args); err != nil {
						t.Fatal(err)
					}
					err := cmd.ValidateFlagGroups()
					if (err != nil) != tt.wantErr {
						t.Fatalf("ValidateFlagGroups() = %v, want error %v", err, tt.wantErr)
					}
					if tt.wantErr && (!strings.Contains(err.Error(), "raw") || !strings.Contains(err.Error(), "comment-threads")) {
						t.Errorf("incompatibility error does not name both flags: %v", err)
					}
				})
			}
		})
	}
}

// TestCommentThreadsUnsupportedCommands prevents epic and sprint lists from inheriting unsupported thread flags or exclusivity rules.
func TestCommentThreadsUnsupportedCommands(t *testing.T) {
	for _, root := range []*cobra.Command{epic.NewCmdEpic(), sprint.NewCmdSprint()} {
		t.Run(root.Name(), func(t *testing.T) {
			cmd, _, err := root.Find([]string{"list"})
			if err != nil {
				t.Fatal(err)
			}
			if cmd.Flags().Lookup("comment-threads") != nil || strings.Contains(cmd.UsageString(), "comment-threads") {
				t.Error("unsupported threaded comments are advertised")
			}
			comments := cmd.Flags().Lookup("comments")
			if comments == nil || comments.Usage != "Show N comments when viewing the issue" {
				t.Fatalf("shared comments help is not generic: %+v", comments)
			}
			if err := cmd.ParseFlags([]string{"--raw", "--comments", "0"}); err != nil {
				t.Fatal(err)
			}
			if err := cmd.ValidateFlagGroups(); err != nil {
				t.Fatalf("shared flags inherited thread exclusivity: %v", err)
			}
			if err := cmd.ParseFlags([]string{"--comment-threads"}); err == nil || !strings.Contains(err.Error(), "unknown flag: --comment-threads") {
				t.Errorf("expected unsupported flag rejection, got %v", err)
			}
		})
	}
}

// TestCommentThreadsDoNotChangeListQuery ensures detail presentation options cannot alter issue search criteria.
func TestCommentThreadsDoNotChangeListQuery(t *testing.T) {
	var flatQuery string
	for _, threads := range []bool{false, true} {
		cmd := issueCommand(t, "list")
		cmd.Flags().Bool("debug", false, "Turn on debug output")
		args := []string{"--status", "In Progress", "--priority", "High", "--jql", "summary ~ cli", "--comments", "5"}
		if threads {
			args = append(args, "--comment-threads", "--csv")
		}
		if err := cmd.ParseFlags(args); err != nil {
			t.Fatal(err)
		}
		if err := cmd.ValidateFlagGroups(); err != nil {
			t.Fatal(err)
		}
		q, err := query.NewIssue("ISSUE", cmd.Flags())
		if err != nil {
			t.Fatal(err)
		}
		if !threads {
			flatQuery = q.Get()
		} else if got := q.Get(); got != flatQuery {
			t.Errorf("threaded details changed list query: got %q, want %q", got, flatQuery)
		}
	}
}
