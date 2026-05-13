package cmdcommon

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/pkg/md"
)

// Body format values for the --body-format flag.
const (
	// FormatMarkdown means the body is CommonMark and should be converted to JIRA wiki markup before sending.
	FormatMarkdown = "markdown"
	// FormatWiki means the body is already JIRA wiki markup and should be sent verbatim.
	FormatWiki = "wiki"

	// FlagBodyFormat is the long flag name registered on each affected command.
	FlagBodyFormat = "body-format"
	// ConfigKeyBodyFormat is the viper config key that mirrors the flag.
	ConfigKeyBodyFormat = "body.format"
)

// AddBodyFormatFlag registers --body-format on the given flag set.
func AddBodyFormatFlag(flags *pflag.FlagSet) {
	flags.String(FlagBodyFormat, "",
		`Body source format: "markdown" (default, converted to JIRA wiki markup) or "wiki" (sent as-is)`)
}

// ResolveBodyFormat returns the effective body format using the resolution order:
// flag value > viper "body.format" config key > FormatMarkdown default. The returned
// value is validated against the allowed values and an error is returned otherwise.
func ResolveBodyFormat(flags *pflag.FlagSet) (string, error) {
	v, _ := flags.GetString(FlagBodyFormat)
	if v == "" {
		v = viper.GetString(ConfigKeyBodyFormat)
	}
	if v == "" {
		v = FormatMarkdown
	}
	if v != FormatMarkdown && v != FormatWiki {
		return "", fmt.Errorf("invalid --%s %q (allowed values: %s, %s)", FlagBodyFormat, v, FormatMarkdown, FormatWiki)
	}
	return v, nil
}

// ConvertBody returns body converted from CommonMark to JIRA wiki markup when format
// is FormatMarkdown, or body verbatim when format is FormatWiki.
func ConvertBody(body, format string) string {
	if format == FormatWiki {
		return body
	}
	return md.ToJiraMD(body)
}
