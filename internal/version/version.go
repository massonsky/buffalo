// Package version provides version information for Buffalo.
package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	// Version is the current version of Buffalo.
	// Injected at build time via ldflags:
	//   -X github.com/massonsky/buffalo/internal/version.Version=1.32.5
	// When built without ldflags (e.g. plain `go install`), defaults to "dev".
	Version = "dev"

	// GitCommit is the git commit hash (short).
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built.
	BuildDate = "unknown"

	// GoVersion is the Go version used to build the binary.
	GoVersion = runtime.Version()

	// Platform is the OS/Arch combination.
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// FullVersion returns version with commit hash suffix (e.g. "1.32.5+d58ff2d").
func FullVersion() string {
	if Version == "dev" {
		return "dev"
	}
	commit := safeShort(GitCommit, 7)
	return Version + "+" + commit
}

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf(
		"Buffalo v%s (%s)\nGit commit: %s\nBuild date: %s\nGo version: %s\nPlatform:   %s",
		FullVersion(), safeShort(GitCommit, 7), GitCommit, BuildDate, GoVersion, Platform,
	)
}

// Short returns a short version string.
func Short() string {
	return fmt.Sprintf("v%s", FullVersion())
}

// safeShort returns first n chars of s, or s if shorter.
func safeShort(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
