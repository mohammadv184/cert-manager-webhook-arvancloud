package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	// Version is the version of the CLI.
	Version = "dev"
	// BuildDate is the date the CLI was built.
	BuildDate = ""
	// Commit is the commit hash the CLI was built from.
	Commit = ""
	// GoVersion is the version of Go used to build the CLI.
	GoVersion = runtime.Version()
	// OS is the operating system the CLI was built for.
	OS = runtime.GOOS
	// Arch is the architecture the CLI was built for.
	Arch = runtime.GOARCH
	// Compiler is the compiler used to build the CLI.
	Compiler = runtime.Compiler
)

func init() {
	if bi, isAvailable := debug.ReadBuildInfo(); isAvailable {
		if bi.Main.Version != "" {
			Version = bi.Main.Version
		}
		if Commit == "" {
			Commit = fmt.Sprintf("(unknown, sum=%s)", bi.Main.Sum)
		}
		if BuildDate == "" {
			BuildDate = "(unknown)"
		}
	}
}
