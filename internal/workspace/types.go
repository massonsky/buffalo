// Package workspace provides multi-project monorepo management for Buffalo.
package workspace

import (
	"time"
)

// Config represents a workspace configuration (buffalo-workspace.yaml).
type Config struct {
	// Workspace contains workspace-level settings.
	Workspace WorkspaceSettings `yaml:"workspace" json:"workspace"`
	// Projects is the list of projects in the workspace.
	Projects []Project `yaml:"projects" json:"projects"`
	// Policies contains workspace-wide policies.
	Policies Policies `yaml:"policies" json:"policies"`
}

// WorkspaceSettings contains workspace-level configuration.
type WorkspaceSettings struct {
	// Name is the workspace name.
	Name string `yaml:"name" json:"name"`
	// Description of the workspace.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Version of the workspace configuration.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// Project represents a proto project within the workspace.
type Project struct {
	// Name is the unique project identifier.
	Name string `yaml:"name" json:"name"`
	// Path is the relative path to the project directory.
	Path string `yaml:"path" json:"path"`
	// Tags for categorizing projects.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	// DependsOn lists project dependencies.
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	// Config is an optional project-specific config file override.
	Config string `yaml:"config,omitempty" json:"config,omitempty"`
	// Enabled indicates if the project is active.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// IsEnabled returns true if the project is enabled (default: true).
func (p *Project) IsEnabled() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

// Policies contains workspace-wide build and validation policies.
type Policies struct {
	// ConsistentVersions requires all projects to use the same proto plugin versions.
	ConsistentVersions bool `yaml:"consistent_versions" json:"consistent_versions"`
	// SharedDependencies allows sharing dependencies across projects.
	SharedDependencies bool `yaml:"shared_dependencies" json:"shared_dependencies"`
	// NoCircularDeps prevents circular dependencies between projects.
	NoCircularDeps bool `yaml:"no_circular_deps" json:"no_circular_deps"`
	// RequireTags requires all projects to have at least one tag.
	RequireTags bool `yaml:"require_tags" json:"require_tags"`
	// AllowedTags limits which tags can be used.
	AllowedTags []string `yaml:"allowed_tags,omitempty" json:"allowed_tags,omitempty"`
}

// ProjectState represents the current state of a project.
type ProjectState struct {
	// Project is the project configuration.
	Project *Project
	// LastBuild is when the project was last built.
	LastBuild time.Time
	// LastModified is when the project was last modified.
	LastModified time.Time
	// Hash is the content hash of the project's proto files.
	Hash string
	// Status is the current build status.
	Status BuildStatus
	// Error contains any error from the last build.
	Error error
}

// BuildStatus represents the build state of a project.
type BuildStatus string

const (
	// StatusUnknown when status is not determined.
	StatusUnknown BuildStatus = "unknown"
	// StatusPending when build is pending.
	StatusPending BuildStatus = "pending"
	// StatusBuilding when build is in progress.
	StatusBuilding BuildStatus = "building"
	// StatusSuccess when build succeeded.
	StatusSuccess BuildStatus = "success"
	// StatusFailed when build failed.
	StatusFailed BuildStatus = "failed"
	// StatusSkipped when build was skipped.
	StatusSkipped BuildStatus = "skipped"
)

// BuildResult contains the result of building a project.
type BuildResult struct {
	// Project is the project that was built.
	Project *Project
	// Status is the build status.
	Status BuildStatus
	// Duration is how long the build took.
	Duration time.Duration
	// Error contains any error that occurred.
	Error error
	// OutputFiles lists the generated files.
	OutputFiles []string
	// CacheHit indicates if the build used cached results.
	CacheHit bool
}

// AffectedResult contains analysis of affected projects.
type AffectedResult struct {
	// ChangedFiles lists files that changed.
	ChangedFiles []string
	// DirectlyAffected lists projects directly affected by changes.
	DirectlyAffected []*Project
	// TransitivelyAffected lists projects indirectly affected through dependencies.
	TransitivelyAffected []*Project
	// RecommendedBuild suggests which projects to build.
	RecommendedBuild []string
}

// ValidationResult contains workspace validation results.
type ValidationResult struct {
	// Valid indicates if the workspace is valid.
	Valid bool
	// Errors contains validation errors.
	Errors []ValidationError
	// Warnings contains validation warnings.
	Warnings []ValidationWarning
}

// ValidationError represents a validation error.
type ValidationError struct {
	// Project is the affected project (empty for workspace-level).
	Project string
	// Field is the invalid field.
	Field string
	// Message describes the error.
	Message string
}

// ValidationWarning represents a validation warning.
type ValidationWarning struct {
	// Project is the affected project (empty for workspace-level).
	Project string
	// Message describes the warning.
	Message string
}

// DependencyGraph represents the project dependency graph.
type DependencyGraph struct {
	// Projects maps project names to their dependencies.
	Projects map[string][]string
	// Reverse maps project names to projects that depend on them.
	Reverse map[string][]string
}
