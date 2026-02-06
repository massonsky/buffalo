// Package upgrade provides functionality for upgrading Buffalo and migrating configurations.
package upgrade

import (
	"time"
)

// ReleaseInfo contains information about a Buffalo release.
type ReleaseInfo struct {
	// Version is the release version (e.g., "5.0.0").
	Version string `json:"tag_name"`
	// Name is the release name.
	Name string `json:"name"`
	// Body is the release notes/changelog.
	Body string `json:"body"`
	// PublishedAt is when the release was published.
	PublishedAt time.Time `json:"published_at"`
	// HTMLURL is the URL to the release page.
	HTMLURL string `json:"html_url"`
	// Prerelease indicates if this is a pre-release.
	Prerelease bool `json:"prerelease"`
	// Draft indicates if this is a draft release.
	Draft bool `json:"draft"`
	// Assets contains the release assets (binaries).
	Assets []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents a downloadable asset in a release.
type ReleaseAsset struct {
	// Name is the asset filename.
	Name string `json:"name"`
	// Size is the asset size in bytes.
	Size int64 `json:"size"`
	// DownloadURL is the URL to download the asset.
	DownloadURL string `json:"browser_download_url"`
	// ContentType is the MIME type of the asset.
	ContentType string `json:"content_type"`
}

// UpgradeCheck contains the result of checking for updates.
type UpgradeCheck struct {
	// CurrentVersion is the currently installed version.
	CurrentVersion string
	// LatestVersion is the latest available version.
	LatestVersion string
	// UpdateAvailable indicates if an update is available.
	UpdateAvailable bool
	// LatestRelease contains full release information.
	LatestRelease *ReleaseInfo
	// MigrationSteps lists what will be migrated.
	MigrationSteps []MigrationStep
}

// MigrationStep describes a single migration action.
type MigrationStep struct {
	// Component being migrated (e.g., "buffalo.yaml", "plugins").
	Component string
	// Description of what will change.
	Description string
	// FromVersion is the source version.
	FromVersion string
	// ToVersion is the target version.
	ToVersion string
	// Breaking indicates if this is a breaking change.
	Breaking bool
}

// MigrationResult contains the result of a migration.
type MigrationResult struct {
	// Success indicates if migration was successful.
	Success bool
	// Steps contains the executed migration steps.
	Steps []MigrationStepResult
	// BackupPath is the path to the backup (if created).
	BackupPath string
	// Errors contains any errors that occurred.
	Errors []error
}

// MigrationStepResult contains the result of a single migration step.
type MigrationStepResult struct {
	// Step is the migration step that was executed.
	Step MigrationStep
	// Success indicates if this step was successful.
	Success bool
	// Error contains any error that occurred.
	Error error
	// Changes describes what was changed.
	Changes []string
}

// UpgradeOptions contains options for the upgrade process.
type UpgradeOptions struct {
	// TargetVersion is the version to upgrade to (empty for latest).
	TargetVersion string
	// DryRun if true, only shows what would change without applying.
	DryRun bool
	// Force if true, skips confirmation prompts.
	Force bool
	// SkipBinaryUpgrade if true, only migrates configs without updating binary.
	SkipBinaryUpgrade bool
	// SkipConfigMigration if true, only updates binary without migrating configs.
	SkipConfigMigration bool
	// CreateBackup if true, creates backup before migration.
	CreateBackup bool
	// BackupDir is the directory for backups.
	BackupDir string
	// ConfigPath is the path to buffalo.yaml.
	ConfigPath string
}

// RollbackInfo contains information about a previous upgrade for rollback.
type RollbackInfo struct {
	// Timestamp when the upgrade was performed.
	Timestamp time.Time `json:"timestamp"`
	// FromVersion is the version before upgrade.
	FromVersion string `json:"from_version"`
	// ToVersion is the version after upgrade.
	ToVersion string `json:"to_version"`
	// BackupPath is the path to the backup.
	BackupPath string `json:"backup_path"`
	// BinaryBackupPath is the path to the binary backup.
	BinaryBackupPath string `json:"binary_backup_path"`
}
