package view

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

func TestIssueListRenderPTY(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("PTY harness requires Darwin or Linux controlling-terminal ioctls")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable: PTY driver uses only Python's standard library")
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the PTY driver")
	}
	driver := filepath.Join(filepath.Dir(sourceFile), "issues_pty_driver.py")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, driver, "--", binary,
		"-test.run=^TestIssueListRenderPTYHelper$", "-test.count=1", "-test.timeout=30s")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PTY regression: %v\n%s", err, out)
	}
	t.Log(strings.TrimSpace(string(out)))
}

// TestIssueListRenderPTYHelper runs only as the driver child. No CLI initialization,
// user configuration, netrc, keychain, proxy or external Jira server is involved.
func TestIssueListRenderPTYHelper(t *testing.T) {
	if os.Getenv("JIRA_CLI_RENDER_PTY_HELPER") != "1" {
		t.Skip("child fixture; run TestIssueListRenderPTY or issues_pty_driver.py -- <test binary> -test.run=^TestIssueListRenderPTYHelper$")
	}
	if _, present := os.LookupEnv("GLAMOUR_STYLE"); present {
		t.Fatal("fixture requires unset GLAMOUR_STYLE, not a forced dark style")
	}
	if tui.IsNotTTY() || tui.IsDumbTerminal() {
		t.Fatal("fixture requires raw stdout attached to a real terminal")
	}
	viper.Reset()
	defer viper.Reset()
	var mu sync.Mutex
	counts := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("fixture forbids %s requests", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, "/rest/api/3/issue/")
		if r.URL.Path != "/rest/api/3/issue/"+key || (key != "FIRST-1" && key != "SECOND-2") {
			t.Error("request outside synthetic issue endpoints")
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		counts[key]++
		count := counts[key]
		mu.Unlock()
		summary, body := "SELECTEDTITLE", "SELECTEDBODY"
		if key == "SECOND-2" {
			summary, body = "NAVIGATEDTITLE", "NAVIGATEDBODY"
			if count > 1 {
				summary, body = "REFRESHEDTITLE", "REFRESHEDBODY"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"key": key,
			"fields": map[string]any{
				"summary": summary,
				"description": map[string]any{
					"version": 1, "type": "doc",
					"content": []any{map[string]any{
						"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": body}},
					}},
				},
				"comment": map[string]any{"total": 0, "comments": []any{}},
			},
		}); err != nil {
			t.Errorf("encode synthetic issue: %v", err)
		}
	}))
	defer server.Close()
	viper.Set("installation", jira.InstallationTypeCloud)
	viper.Set("server", server.URL)
	viper.Set("login", "fixture@example.invalid")
	viper.Set("api_token", "synthetic-not-a-credential")
	viper.Set("auth_type", string(jira.AuthTypeBasic))
	api.Client(jira.Config{Server: server.URL, Login: "fixture@example.invalid", APIToken: "synthetic-not-a-credential"})
	list := IssueList{
		Project: "FIRST", Server: server.URL,
		Data:    []*jira.Issue{{Key: "FIRST-1", Fields: jira.IssueFields{Summary: "ROOTLISTREADY"}}},
		Display: DisplayFormat{Columns: []string{"key", "summary"}},
	}
	if err := list.Render(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if counts["FIRST-1"] != 1 || counts["SECOND-2"] != 2 {
		t.Errorf("GET counts = %v, want FIRST-1:1 SECOND-2:2 (navigation plus refresh)", counts)
	}
}
