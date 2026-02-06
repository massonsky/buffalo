package upgrade

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
)

// Upgrader handles the upgrade process.
type Upgrader struct {
	checker  *Checker
	migrator *Migrator
	logger   *logger.Logger
	client   *http.Client
}

// NewUpgrader creates a new upgrader.
func NewUpgrader(log *logger.Logger, backupDir string, opts ...CheckerOption) *Upgrader {
	return &Upgrader{
		checker:  NewChecker(opts...),
		migrator: NewMigrator(backupDir),
		logger:   log,
		client: &http.Client{
			Timeout: 5 * time.Minute, // Longer timeout for downloads
		},
	}
}

// Check checks for available updates.
func (u *Upgrader) Check(ctx context.Context) (*UpgradeCheck, error) {
	check, err := u.checker.CheckForUpdates(ctx)
	if err != nil {
		return nil, err
	}

	// Add migration steps if update is available
	if check.UpdateAvailable {
		check.MigrationSteps = u.migrator.GetMigrationSteps(check.CurrentVersion, check.LatestVersion)
	}

	return check, nil
}

// CheckVersion checks for a specific version.
func (u *Upgrader) CheckVersion(ctx context.Context, targetVersion string) (*UpgradeCheck, error) {
	release, err := u.checker.GetRelease(ctx, targetVersion)
	if err != nil {
		return nil, err
	}

	currentVersion := version.Version
	targetVer := release.Version

	check := &UpgradeCheck{
		CurrentVersion:  currentVersion,
		LatestVersion:   targetVer,
		UpdateAvailable: compareVersions(targetVer, currentVersion) != 0,
		LatestRelease:   release,
		MigrationSteps:  u.migrator.GetMigrationSteps(currentVersion, targetVer),
	}

	return check, nil
}

// Upgrade performs the upgrade to the specified version.
func (u *Upgrader) Upgrade(ctx context.Context, opts UpgradeOptions) (*MigrationResult, error) {
	// Determine target version
	var release *ReleaseInfo
	var err error

	if opts.TargetVersion == "" {
		release, err = u.checker.GetLatestRelease(ctx)
	} else {
		release, err = u.checker.GetRelease(ctx, opts.TargetVersion)
	}

	if err != nil {
		return nil, err
	}

	targetVersion := release.Version
	currentVersion := version.Version

	// Check if upgrade is needed
	if compareVersions(targetVersion, currentVersion) == 0 {
		return &MigrationResult{
			Success: true,
			Steps:   []MigrationStepResult{},
		}, nil
	}

	u.logger.Info("🦬 Starting upgrade",
		logger.String("from", currentVersion),
		logger.String("to", targetVersion))

	result := &MigrationResult{
		Success: true,
		Steps:   []MigrationStepResult{},
	}

	// Create backup directory
	backupDir := opts.BackupDir
	if backupDir == "" {
		backupDir = ".buffalo/backup"
	}

	if opts.CreateBackup && !opts.DryRun {
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return nil, errors.Wrap(err, errors.ErrIO, "failed to create backup directory")
		}
	}

	// Save rollback info
	if !opts.DryRun && opts.CreateBackup {
		rollbackInfo := &RollbackInfo{
			Timestamp:   time.Now(),
			FromVersion: currentVersion,
			ToVersion:   targetVersion,
		}

		// Get current binary path
		execPath, err := os.Executable()
		if err == nil && !opts.SkipBinaryUpgrade {
			binaryBackup := filepath.Join(backupDir, "buffalo.bak")
			if err := copyFile(execPath, binaryBackup); err == nil {
				rollbackInfo.BinaryBackupPath = binaryBackup
			}
		}

		if opts.ConfigPath != "" {
			configBackup := filepath.Join(backupDir, "buffalo.yaml.bak")
			if err := copyFile(opts.ConfigPath, configBackup); err == nil {
				rollbackInfo.BackupPath = configBackup
			}
		}

		u.migrator.SaveRollbackInfo(rollbackInfo)
		result.BackupPath = backupDir
	}

	// Upgrade binary
	if !opts.SkipBinaryUpgrade {
		u.logger.Info("📦 Downloading new version...")

		if opts.DryRun {
			u.logger.Info("  [dry-run] Would download and install binary",
				logger.String("version", targetVersion))
		} else {
			if err := u.downloadAndInstall(ctx, release); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, err)
				return result, err
			}
			u.logger.Info("✅ Binary upgraded successfully")
		}
	}

	// Migrate configs
	if !opts.SkipConfigMigration && opts.ConfigPath != "" {
		u.logger.Info("📝 Migrating configuration...")

		// Update migrator backup dir
		u.migrator.backupDir = backupDir

		migResult, err := u.migrator.Migrate(opts.ConfigPath, currentVersion, targetVersion, opts.DryRun)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err)
			return result, err
		}

		result.Steps = migResult.Steps

		if opts.DryRun {
			u.logger.Info("  [dry-run] Would apply the following migrations:")
			for _, step := range result.Steps {
				marker := "  "
				if step.Step.Breaking {
					marker = "⚠️"
				}
				u.logger.Info(fmt.Sprintf("    %s %s", marker, step.Step.Description))
			}
		} else {
			u.logger.Info("✅ Configuration migrated successfully")
		}
	}

	if result.Success && !opts.DryRun {
		u.logger.Info("🎉 Upgrade completed successfully!")
		u.logger.Info("   Run 'buffalo version' to verify the new version")
	}

	return result, nil
}

// Rollback rolls back to the previous version.
func (u *Upgrader) Rollback(configPath string) error {
	u.logger.Info("⏪ Rolling back upgrade...")

	info, err := u.migrator.LoadRollbackInfo()
	if err != nil {
		return err
	}

	// Restore binary
	if info.BinaryBackupPath != "" {
		execPath, err := os.Executable()
		if err == nil {
			if err := copyFile(info.BinaryBackupPath, execPath); err != nil {
				u.logger.Warn("Failed to restore binary", logger.Any("error", err))
			} else {
				u.logger.Info("✅ Binary restored")
			}
		}
	}

	// Restore config
	if info.BackupPath != "" {
		if err := u.migrator.Rollback(configPath); err != nil {
			return err
		}
		u.logger.Info("✅ Configuration restored")
	}

	u.logger.Info("🎉 Rollback completed successfully!")
	return nil
}

// GetChangelog returns the changelog between versions.
func (u *Upgrader) GetChangelog(ctx context.Context, fromVersion, toVersion string) (string, error) {
	if fromVersion == "" {
		fromVersion = version.Version
	}

	if toVersion == "" {
		latest, err := u.checker.GetLatestRelease(ctx)
		if err != nil {
			return "", err
		}
		toVersion = latest.Version
	}

	return u.checker.GetChangelogBetweenVersions(ctx, fromVersion, toVersion)
}

// downloadAndInstall downloads and installs the new binary.
func (u *Upgrader) downloadAndInstall(ctx context.Context, release *ReleaseInfo) error {
	// Find appropriate asset
	asset, err := u.checker.GetAssetForPlatform(release)
	if err != nil {
		return err
	}

	u.logger.Info("  Downloading", logger.String("file", asset.Name), logger.Int64("size", asset.Size))

	// Download asset
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to create download request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to download binary")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(errors.ErrIO, fmt.Sprintf("download failed: %s", resp.Status))
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "buffalo-upgrade-*")
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to create temp file")
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return errors.Wrap(err, errors.ErrIO, "failed to write downloaded file")
	}
	tmpFile.Close()

	// Extract binary from archive
	binaryPath, err := u.extractBinary(tmpPath, asset.Name)
	if err != nil {
		return err
	}
	defer os.Remove(binaryPath)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to get executable path")
	}

	// Replace current binary
	if err := replaceBinary(execPath, binaryPath); err != nil {
		return err
	}

	return nil
}

// extractBinary extracts the buffalo binary from an archive.
func (u *Upgrader) extractBinary(archivePath, archiveName string) (string, error) {
	// Determine archive type
	switch {
	case hasAnySuffix(archiveName, ".tar.gz", ".tgz"):
		return u.extractFromTarGz(archivePath)
	case hasAnySuffix(archiveName, ".zip"):
		return u.extractFromZip(archivePath)
	default:
		// Assume it's a raw binary
		return archivePath, nil
	}
}

// extractFromTarGz extracts buffalo binary from a tar.gz archive.
func (u *Upgrader) extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to open archive")
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to create gzip reader")
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	binaryName := "buffalo"
	if runtime.GOOS == "windows" {
		binaryName = "buffalo.exe"
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, errors.ErrIO, "failed to read archive")
		}

		if filepath.Base(hdr.Name) == binaryName {
			tmpFile, err := os.CreateTemp("", "buffalo-binary-*")
			if err != nil {
				return "", errors.Wrap(err, errors.ErrIO, "failed to create temp file")
			}

			if _, err := io.Copy(tmpFile, tr); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				return "", errors.Wrap(err, errors.ErrIO, "failed to extract binary")
			}

			tmpFile.Close()
			return tmpFile.Name(), nil
		}
	}

	return "", errors.New(errors.ErrNotFound, "buffalo binary not found in archive")
}

// extractFromZip extracts buffalo binary from a zip archive.
func (u *Upgrader) extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to open zip archive")
	}
	defer r.Close()

	binaryName := "buffalo"
	if runtime.GOOS == "windows" {
		binaryName = "buffalo.exe"
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return "", errors.Wrap(err, errors.ErrIO, "failed to open file in archive")
			}

			tmpFile, err := os.CreateTemp("", "buffalo-binary-*")
			if err != nil {
				rc.Close()
				return "", errors.Wrap(err, errors.ErrIO, "failed to create temp file")
			}

			if _, err := io.Copy(tmpFile, rc); err != nil {
				tmpFile.Close()
				rc.Close()
				os.Remove(tmpFile.Name())
				return "", errors.Wrap(err, errors.ErrIO, "failed to extract binary")
			}

			tmpFile.Close()
			rc.Close()
			return tmpFile.Name(), nil
		}
	}

	return "", errors.New(errors.ErrNotFound, "buffalo binary not found in archive")
}

// replaceBinary replaces the current binary with a new one.
func replaceBinary(currentPath, newPath string) error {
	// Make new binary executable
	if err := os.Chmod(newPath, 0755); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to set permissions")
	}

	// On Windows, we need to rename the current binary first
	if runtime.GOOS == "windows" {
		oldPath := currentPath + ".old"
		os.Remove(oldPath) // Remove any existing .old file
		if err := os.Rename(currentPath, oldPath); err != nil {
			return errors.Wrap(err, errors.ErrIO, "failed to rename current binary")
		}
	}

	// Copy new binary to current location
	if err := copyFile(newPath, currentPath); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to replace binary")
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Preserve permissions
	sourceInfo, err := os.Stat(src)
	if err == nil {
		destFile.Chmod(sourceInfo.Mode())
	}

	return nil
}

// hasAnySuffix checks if s has any of the given suffixes.
func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}
