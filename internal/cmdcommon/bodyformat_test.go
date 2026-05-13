package cmdcommon

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestResolveBodyFormat(t *testing.T) {
	cases := []struct {
		name       string
		flagValue  string
		viperValue string
		want       string
		wantErr    bool
	}{
		{
			name: "defaults to markdown when nothing is set",
			want: FormatMarkdown,
		},
		{
			name:       "viper value used when flag is unset",
			viperValue: FormatWiki,
			want:       FormatWiki,
		},
		{
			name:       "flag overrides viper",
			flagValue:  FormatMarkdown,
			viperValue: FormatWiki,
			want:       FormatMarkdown,
		},
		{
			name:      "explicit wiki flag is honored",
			flagValue: FormatWiki,
			want:      FormatWiki,
		},
		{
			name:      "invalid flag value errors",
			flagValue: "html",
			wantErr:   true,
		},
		{
			name:       "invalid viper value errors",
			viperValue: "asciidoc",
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			if tc.viperValue != "" {
				viper.Set(ConfigKeyBodyFormat, tc.viperValue)
			}

			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			AddBodyFormatFlag(flags)
			if tc.flagValue != "" {
				assert.NoError(t, flags.Set(FlagBodyFormat, tc.flagValue))
			}

			got, err := ResolveBodyFormat(flags)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConvertBody(t *testing.T) {
	t.Run("markdown is converted to JIRA wiki", func(t *testing.T) {
		// `# foo` is a CommonMark H1; the wiki form is `h1. foo`.
		got := ConvertBody("# foo\n", FormatMarkdown)
		assert.Contains(t, got, "h1. foo")
	})

	t.Run("wiki bypasses conversion verbatim", func(t *testing.T) {
		// `# foo` in JIRA wiki is a numbered-list item; we must NOT convert it to `h1.`.
		body := "# first\n# second\n"
		got := ConvertBody(body, FormatWiki)
		assert.Equal(t, body, got)
	})

	t.Run("empty body returns empty", func(t *testing.T) {
		assert.Equal(t, "", ConvertBody("", FormatMarkdown))
		assert.Equal(t, "", ConvertBody("", FormatWiki))
	})
}
