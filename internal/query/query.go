package query

// FlagParser wraps pflag.FlagSet struct.
type FlagParser interface {
	GetBool(string) (bool, error)
	GetString(string) (string, error)
	GetStringArray(string) ([]string, error)
	GetStringToString(string) (map[string]string, error)
	GetUint(name string) (uint, error)
	Set(name, value string) error
}
