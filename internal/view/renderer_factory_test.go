package view

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"golang.org/x/term"

	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const factoryTestMarkdown = "# Heading\n\nA **bold** paragraph and a [link](https://example.invalid).\n\n- First\n- Second\n\n```go\nprintln(\"code\")\n```\n\n"

// TestMDRendererFactoryBuiltinStyles requires captured styles and wrapping to survive later environment changes.
func TestMDRendererFactoryBuiltinStyles(t *testing.T) {
	for _, style := range []string{styles.DarkStyle, styles.LightStyle, styles.NoTTYStyle, styles.AsciiStyle, styles.DraculaStyle, styles.TokyoNightStyle, styles.PinkStyle} {
		t.Run(style, func(t *testing.T) {
			t.Setenv("GLAMOUR_STYLE", style)
			factory := newMDRendererFactory()
			expected, err := MDRenderer()
			if err != nil {
				t.Fatal(err)
			}
			markdown := factoryTestMarkdown + strings.Repeat("wrapped words ", wordWrap)
			want, err := expected.Render(markdown)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("GLAMOUR_STYLE", "missing-style-after-capture")
			if got := renderFactoryMarkdown(t, factory, markdown); got != want {
				t.Errorf("factory changed %s styling or word wrap", style)
			}
		})
	}
}

// TestMDRendererFactoryAutoWithoutTTY preserves non-TTY auto detection when the environment changes after capture.
func TestMDRendererFactoryAutoWithoutTTY(t *testing.T) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		t.Skip("automatic TTY background selection is covered by the PTY tests")
	}
	for _, style := range []string{"", styles.AutoStyle} {
		t.Run("style="+style, func(t *testing.T) {
			t.Setenv("GLAMOUR_STYLE", style)
			if style == "" {
				if err := os.Unsetenv("GLAMOUR_STYLE"); err != nil {
					t.Fatal(err)
				}
			}
			factory := newMDRendererFactory()
			t.Setenv("GLAMOUR_STYLE", styles.DarkStyle)
			expected, err := glamour.NewTermRenderer(glamour.WithStandardStyle(styles.NoTTYStyle), glamour.WithWordWrap(wordWrap))
			if err != nil {
				t.Fatal(err)
			}
			want, err := expected.Render(factoryTestMarkdown)
			if err != nil {
				t.Fatal(err)
			}
			if got := renderFactoryMarkdown(t, factory, factoryTestMarkdown); got != want {
				t.Error("auto style did not capture stdout's non-TTY state")
			}
		})
	}
}

// TestMDRendererFactoryCustomStyle ensures configured JSON styling is retained rather than reselected on each load.
func TestMDRendererFactoryCustomStyle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "style.json")
	const style = `{"document":{"block_prefix":"CAPTURED-STYLE\n","block_suffix":"\nEND-CAPTURED"}}`
	if err := os.WriteFile(path, []byte(style), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLAMOUR_STYLE", path)
	factory := newMDRendererFactory()
	expected, err := MDRenderer()
	if err != nil {
		t.Fatal(err)
	}
	want, err := expected.Render(factoryTestMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLAMOUR_STYLE", styles.AutoStyle)
	got := renderFactoryMarkdown(t, factory, factoryTestMarkdown)
	if got != want || !strings.Contains(got, "CAPTURED-STYLE") {
		t.Errorf("factory lost the configured custom style: %q", got)
	}
}

// TestMDRendererFactoryStyleErrors prevents later environment changes from masking captured style initialization failures.
func TestMDRendererFactoryStyleErrors(t *testing.T) {
	invalidJSON := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalidJSON, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, style := range []string{filepath.Join(t.TempDir(), "missing.json"), invalidJSON} {
		t.Run(filepath.Base(style), func(t *testing.T) {
			t.Setenv("GLAMOUR_STYLE", style)
			factory := newMDRendererFactory()
			_, wantErr := MDRenderer()
			if wantErr == nil {
				t.Fatal("expected an invalid style error")
			}
			t.Setenv("GLAMOUR_STYLE", styles.NoTTYStyle)
			renderer, err := factory()
			if renderer != nil || err == nil || err.Error() != wantErr.Error() {
				t.Errorf("factory error = %v, want %v", err, wantErr)
			}
		})
	}
}

// TestMDRendererFactoryFreshConcurrentRenderers requires isolated mutable renderers with identical captured styling under concurrent loads.
func TestMDRendererFactoryFreshConcurrentRenderers(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", styles.DarkStyle)
	factory := newMDRendererFactory()
	want := renderFactoryMarkdown(t, factory, factoryTestMarkdown)
	t.Setenv("GLAMOUR_STYLE", "missing-style-after-capture")
	const workers = 12
	renderers := make(chan *glamour.TermRenderer, workers)
	for range workers {
		go func() {
			renderer, err := factory()
			if err != nil {
				t.Errorf("create renderer: %v", err)
				renderers <- nil
				return
			}
			got, err := renderer.Render(factoryTestMarkdown)
			if err != nil || got != want {
				t.Errorf("concurrent rendering changed output: %v", err)
			}
			renderers <- renderer
		}()
	}
	seen := make(map[*glamour.TermRenderer]bool, workers)
	for range workers {
		renderer := <-renderers
		if renderer == nil {
			continue
		}
		if seen[renderer] {
			t.Error("factory shared a mutable renderer across loads")
		}
		seen[renderer] = true
	}
}

// TestIssueDetailRendererFactorySurvivesNavigationAndReload keeps renderer creation lazy while reusing captured options across ticket changes.
func TestIssueDetailRendererFactorySurvivesNavigationAndReload(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", styles.DarkStyle)
	factory := newMDRendererFactory()
	var calls int
	trackedFactory := func() (*glamour.TermRenderer, error) {
		calls++
		return factory()
	}
	setNavigationCloud(t)
	requests := make(chan navigationRequest, 8)
	server := httptest.NewTLSServer(navigationFlatHandler(t, requests))
	defer server.Close()
	list := IssueList{Server: server.URL}
	detail := list.detailView(navigationTestClient(server.URL), "A-1", trackedFactory)
	t.Setenv("GLAMOUR_STYLE", "missing-style-after-capture")
	loadFactoryDetail(t, detail.Load)
	if err := detail.SubmitComment("Synthetic reload comment"); err != nil {
		t.Fatal(err)
	}
	loadFactoryDetail(t, detail.Load)
	next := openNavigation(t, detail, "B-2")
	if calls != 2 {
		t.Fatalf("navigation unexpectedly constructed a renderer: %d calls", calls)
	}
	loadFactoryDetail(t, next.Load)
	loadFactoryDetail(t, next.Load)
	if calls != 4 {
		t.Errorf("reload/navigation lost the captured renderer factory: %d calls", calls)
	}
}

// renderFactoryMarkdown makes renderer construction and rendering failures fatal before output comparisons.
func renderFactoryMarkdown(t *testing.T, factory mdRendererFactory, markdown string) string {
	t.Helper()
	renderer, err := factory()
	if err != nil {
		t.Fatal(err)
	}
	out, err := renderer.Render(markdown)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// loadFactoryDetail requires a nonempty rendered snapshot so a silently skipped load cannot pass factory checks.
func loadFactoryDetail(t *testing.T, load func() (tui.DetailContent, error)) {
	t.Helper()
	content, err := load()
	if err != nil || content.Text == "" {
		t.Fatalf("captured renderer failed during detail load: %+v, %v", content, err)
	}
}
