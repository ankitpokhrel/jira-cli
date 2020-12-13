package version

import (
	"fmt"
	"runtime"
)

// Build information is populated at build-time.
var (
	Version   string
	GitCommit string
	BuildDate string
	GoVersion = runtime.Version()
	Compiler  = runtime.Compiler
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info returns version and build information.
func Info() string {
	return fmt.Sprintf(
		"(Version=\"%s\", GitCommit=\"%s\", GoVersion=\"%s\", BuildDate=\"%s\", Compiler=\"%s\", Platform=\"%s\")",
		Version, GitCommit, GoVersion, BuildDate, Compiler, Platform,
	)
}
