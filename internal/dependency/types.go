// Package dependency provides dependency management for proto files.
package dependency

import "time"

// Dependency represents a proto dependency.
type Dependency struct {
	// Name is the dependency name (e.g., "googleapis", "protoc-gen-validate")
	Name string `json:"name" yaml:"name"`

	// Source is where to get the dependency
	Source DependencySource `json:"source" yaml:"source"`

	// Version constraint (e.g., "v1.2.3", ">=1.0.0", "main", "master")
	Version string `json:"version" yaml:"version"`

	// SubPath is the path within the repo to proto files (optional)
	SubPath string `json:"sub_path,omitempty" yaml:"sub_path,omitempty"`

	// Includes are specific proto files to include (optional, default: all)
	Includes []string `json:"includes,omitempty" yaml:"includes,omitempty"`

	// Excludes are patterns to exclude
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}

// DependencySource represents where to get the dependency.
type DependencySource struct {
	// Type: "git", "url", "local"
	Type string `json:"type" yaml:"type"`

	// URL for git or http sources
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Path for local sources
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Ref for git (branch, tag, commit)
	Ref string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

// LockFile represents buffalo.lock file with pinned versions.
type LockFile struct {
	// Version of lock file format
	Version string `json:"version" yaml:"version"`

	// Generated timestamp
	Generated time.Time `json:"generated" yaml:"generated"`

	// Dependencies with resolved versions
	Dependencies []LockedDependency `json:"dependencies" yaml:"dependencies"`
}

// LockedDependency is a dependency with resolved version.
type LockedDependency struct {
	Name    string    `json:"name" yaml:"name"`
	Version string    `json:"version" yaml:"version"` // Exact version/commit
	Source  string    `json:"source" yaml:"source"`   // Full URL
	Hash    string    `json:"hash" yaml:"hash"`       // SHA256 of downloaded content
	Updated time.Time `json:"updated" yaml:"updated"`
}

// InstallOptions configures dependency installation.
type InstallOptions struct {
	// Force reinstall even if already exists
	Force bool

	// Update to latest version
	Update bool

	// DryRun only shows what would be installed
	DryRun bool

	// Verbose output
	Verbose bool

	// WorkspaceDir is the buffalo workspace directory
	WorkspaceDir string
}

// DownloadResult contains information about downloaded dependency.
type DownloadResult struct {
	Name      string
	Version   string
	LocalPath string
	ProtoPath string // Path to add to proto_path
	Hash      string
	Error     error
}
