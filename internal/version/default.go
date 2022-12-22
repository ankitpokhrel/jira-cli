//go:build go1.12

package version

import "runtime/debug"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		// info.Main.Version describes the version of the module containing
		// package main, not the version of “the main module”.
		// See https://golang.org/issue/33975.
		if Version == "v0.0.0-dev" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
}
