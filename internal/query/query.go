package query

// FlagParser wraps pflag.FlagSet struct.
type FlagParser interface {
	GetBool(string) (bool, error)
	GetString(string) (string, error)
	GetStringArray(string) ([]string, error)
}
