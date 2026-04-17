package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/massonsky/buffalo/internal/upgrade"
	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	upgradeTargetVersion string
	upgradeDryRun        bool
	upgradeForce         bool
	upgradeSkipBinary    bool
	upgradeSkipConfig    bool
	upgradeBackup        bool
	upgradeCheck         bool
	upgradeChangelog     bool
	upgradeRollback      bool
	upgradeFromSource    bool

	upgradeCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Buffalo to a newer version",
		Long: `Upgrade Buffalo and migrate configurations to a newer version.

This command checks for available updates, downloads the new binary,
and migrates your configuration files automatically.

Examples:
  # Check for available updates
  buffalo upgrade --check

  # Upgrade to the latest version
  buffalo upgrade

  # Upgrade to a specific version
  buffalo upgrade --to 5.0.0

  # Preview what would change (dry run)
  buffalo upgrade --dry-run

  # Show changelog between versions
  buffalo upgrade --changelog

  # Rollback to the previous version
  buffalo upgrade --rollback

  # Skip binary upgrade, only migrate configs
  buffalo upgrade --skip-binary

  # Skip config migration, only upgrade binary
  buffalo upgrade --skip-config

  # Build from source instead of downloading binary
  buffalo upgrade --source`,
		Run: runUpgrade,
	}
)

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.Flags().StringVar(&upgradeTargetVersion, "to", "", "target version to upgrade to (default: latest)")
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "show what would change without applying")
	upgradeCmd.Flags().BoolVarP(&upgradeForce, "force", "f", false, "skip confirmation prompts")
	upgradeCmd.Flags().BoolVar(&upgradeSkipBinary, "skip-binary", false, "skip binary upgrade")
	upgradeCmd.Flags().BoolVar(&upgradeSkipConfig, "skip-config", false, "skip configuration migration")
	upgradeCmd.Flags().BoolVar(&upgradeBackup, "backup", true, "create backup before upgrade")
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "check for available updates")
	upgradeCmd.Flags().BoolVar(&upgradeChangelog, "changelog", false, "show changelog between versions")
	upgradeCmd.Flags().BoolVar(&upgradeRollback, "rollback", false, "rollback to previous version")
	upgradeCmd.Flags().BoolVar(&upgradeFromSource, "source", false, "build from source using 'go install'")
}

func runUpgrade(cmd *cobra.Command, args []string) {
	log := logger.New(logger.WithLevel(logger.INFO))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	backupDir := ".buffalo/backup"
	upgrader := upgrade.NewUpgrader(log, backupDir)

	// Handle rollback
	if upgradeRollback {
		runRollback(log, upgrader)
		return
	}

	// Handle changelog
	if upgradeChangelog {
		runChangelog(ctx, log, upgrader)
		return
	}

	// Handle check
	if upgradeCheck {
		runUpgradeCheck(ctx, log, upgrader)
		return
	}

	// Perform upgrade
	runUpgradeProcess(ctx, log, upgrader)
}

func runUpgradeCheck(ctx context.Context, log *logger.Logger, upgrader *upgrade.Upgrader) {
	var check *upgrade.UpgradeCheck
	var err error

	if upgradeTargetVersion != "" {
		check, err = upgrader.CheckVersion(ctx, upgradeTargetVersion)
	} else {
		check, err = upgrader.Check(ctx)
	}

	if err != nil {
		log.Error("Failed to check for updates", logger.Any("error", err))
		os.Exit(1)
	}

	printUpgradeCheck(check)
}

func printUpgradeCheck(check *upgrade.UpgradeCheck) {
	fmt.Println()
	fmt.Println("🦬 Buffalo Upgrade Check")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	fmt.Printf("  Current version: %s\n", check.CurrentVersion)
	fmt.Printf("  Latest version:  %s\n", check.LatestVersion)
	fmt.Println()

	if !check.UpdateAvailable {
		fmt.Println("✅ You're already on the latest version!")
		fmt.Println()
		return
	}

	fmt.Println("📦 Update available!")
	fmt.Println()

	if check.LatestRelease != nil && check.LatestRelease.Name != "" {
		fmt.Printf("  Release: %s\n", check.LatestRelease.Name)
		if !check.LatestRelease.PublishedAt.IsZero() {
			fmt.Printf("  Released: %s\n", check.LatestRelease.PublishedAt.Format("2006-01-02"))
		}
		fmt.Println()
	}

	if len(check.MigrationSteps) > 0 {
		fmt.Println("📝 Migration steps:")
		for i, step := range check.MigrationSteps {
			marker := "  "
			if step.Breaking {
				marker = "⚠️"
			}
			fmt.Printf("  %d. %s %s\n", i+1, marker, step.Description)
			fmt.Printf("     Component: %s (%s → %s)\n", step.Component, step.FromVersion, step.ToVersion)
		}
		fmt.Println()
	}

	fmt.Printf("Run 'buffalo upgrade' to upgrade to %s\n", check.LatestVersion)
	fmt.Printf("Run 'buffalo upgrade --dry-run' to preview changes\n")
	fmt.Println()
}

func runChangelog(ctx context.Context, log *logger.Logger, upgrader *upgrade.Upgrader) {
	fromVersion := version.Version
	toVersion := upgradeTargetVersion

	changelog, err := upgrader.GetChangelog(ctx, fromVersion, toVersion)
	if err != nil {
		log.Error("Failed to get changelog", logger.Any("error", err))
		os.Exit(1)
	}

	if changelog == "" {
		fmt.Println("No changelog available between versions")
		return
	}

	fmt.Println()
	fmt.Println("📋 Changelog")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
	fmt.Println(changelog)
}

func runRollback(log *logger.Logger, upgrader *upgrade.Upgrader) {
	log.Info("⏪ Rolling back to previous version...")

	// Find config path
	configPath := findConfigPath()

	if err := upgrader.Rollback(configPath); err != nil {
		log.Error("Rollback failed", logger.Any("error", err))
		os.Exit(1)
	}
}

func runUpgradeProcess(ctx context.Context, log *logger.Logger, upgrader *upgrade.Upgrader) {
	// Check for updates first
	var check *upgrade.UpgradeCheck
	var err error

	if upgradeTargetVersion != "" {
		check, err = upgrader.CheckVersion(ctx, upgradeTargetVersion)
	} else {
		check, err = upgrader.Check(ctx)
	}

	if err != nil {
		log.Error("Failed to check for updates", logger.Any("error", err))
		os.Exit(1)
	}

	if !check.UpdateAvailable && upgradeTargetVersion == "" {
		fmt.Println("✅ You're already on the latest version!")
		return
	}

	// Show what will happen
	printUpgradeCheck(check)

	// Confirm unless forced or dry-run
	if !upgradeForce && !upgradeDryRun {
		if !confirmUpgrade(check) {
			fmt.Println("Upgrade canceled")
			return
		}
	}

	// Find config path
	configPath := findConfigPath()

	// Perform upgrade
	opts := upgrade.UpgradeOptions{
		TargetVersion:       upgradeTargetVersion,
		DryRun:              upgradeDryRun,
		Force:               upgradeForce,
		SkipBinaryUpgrade:   upgradeSkipBinary,
		SkipConfigMigration: upgradeSkipConfig || configPath == "",
		CreateBackup:        upgradeBackup,
		BackupDir:           ".buffalo/backup",
		ConfigPath:          configPath,
		FromSource:          upgradeFromSource,
	}

	result, err := upgrader.Upgrade(ctx, opts)
	if err != nil {
		log.Error("Upgrade failed", logger.Any("error", err))
		os.Exit(1)
	}

	// Print result
	if upgradeDryRun {
		fmt.Println()
		fmt.Println("🔍 Dry run completed - no changes were made")
		fmt.Println()
	} else if result.Success {
		fmt.Println()
		fmt.Println("🎉 Upgrade completed successfully!")
		fmt.Println()

		if result.BackupPath != "" {
			fmt.Printf("  Backup saved to: %s\n", result.BackupPath)
		}

		fmt.Println("  Run 'buffalo version' to verify the new version")
		fmt.Println("  Run 'buffalo upgrade --rollback' to undo if needed")
		fmt.Println()
	} else {
		fmt.Println()
		fmt.Println("❌ Upgrade completed with errors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
		fmt.Println()

		if result.BackupPath != "" {
			fmt.Printf("  Backup available at: %s\n", result.BackupPath)
			fmt.Println("  Run 'buffalo upgrade --rollback' to restore")
		}
		os.Exit(1)
	}
}

func confirmUpgrade(check *upgrade.UpgradeCheck) bool {
	// Check for breaking changes
	hasBreaking := false
	for _, step := range check.MigrationSteps {
		if step.Breaking {
			hasBreaking = true
			break
		}
	}

	if hasBreaking {
		fmt.Println("⚠️  This upgrade includes BREAKING CHANGES!")
		fmt.Println()
	}

	fmt.Printf("Upgrade from %s to %s? [y/N]: ", check.CurrentVersion, check.LatestVersion)

	var response string
	_, _ = fmt.Scanln(&response)

	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

func findConfigPath() string {
	// Try common config file names
	paths := []string{
		"buffalo.yaml",
		"buffalo.yml",
		".buffalo.yaml",
		".buffalo.yml",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}
