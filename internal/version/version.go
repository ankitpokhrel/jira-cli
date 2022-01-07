package version

import (
	"fmt"
	"runtime"
	"strconv"
	"time"
)

// Build information is populated at build-time.
var (
	Version         = "v0.0.0-dev"
	GitCommit       = ""
	SourceDateEpoch = "-1"
	GoVersion       = runtime.Version()
	Compiler        = runtime.Compiler
	Platform        = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info returns version and build information.
func Info() string {
	i, err := strconv.ParseInt(SourceDateEpoch, 10, 64) //nolint:gomnd
	if err != nil {
		panic(err)
	}
	commitDate := ""
	if i >= 0 {
		// https://pkg.go.dev/time#Time.Format
		//
		//     $ TZ=MST date -Iseconds -d"Jan 2 15:04:05 2006 MST"
		//     2006-01-02T15:04:05-07:00
		commitDate = time.Unix(i, 0).UTC().Format("2006-01-02T15:04:05-07:00")
	}
	return fmt.Sprintf(
		"(Version=\"%s\", GitCommit=\"%s\", CommitDate=\"%s\", GoVersion=\"%s\", Compiler=\"%s\", Platform=\"%s\")",
		Version, GitCommit, commitDate, GoVersion, Compiler, Platform,
	)
}
