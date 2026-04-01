// Package version provides version information for Buffalo.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of Buffalo.
	Version = "1.22.2"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built.
	BuildDate = "unknown"

	// GoVersion is the Go version used to build the binary.
	GoVersion = runtime.Version()

	// Platform is the OS/Arch combination.
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf(
		"Buffalo %s\nGit commit: %s\nBuild date: %s\nGo version: %s\nPlatform: %s",
		Version, GitCommit, BuildDate, GoVersion, Platform,
	)
}

// Short returns a short version string.
func Short() string {
	return fmt.Sprintf("Buffalo %s (%s)", Version, GitCommit[:7])
}
