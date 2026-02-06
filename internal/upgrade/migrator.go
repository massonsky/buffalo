package upgrade

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Migration represents a configuration migration between versions.
type Migration struct {
	// FromVersion is the minimum version this migration applies to.
	FromVersion string
	// ToVersion is the target version after migration.
	ToVersion string
	// Description describes what this migration does.
	Description string
	// Component is the component being migrated.
	Component string
	// Breaking indicates if this is a breaking change.
	Breaking bool
	// Apply is the function that performs the migration.
	Apply func(data map[string]interface{}) error
}

// Migrator handles configuration migrations between Buffalo versions.
type Migrator struct {
	migrations []Migration
	backupDir  string
}

// NewMigrator creates a new configuration migrator.
func NewMigrator(backupDir string) *Migrator {
	m := &Migrator{
		backupDir: backupDir,
	}
	m.registerMigrations()
	return m
}

// registerMigrations registers all known migrations.
func (m *Migrator) registerMigrations() {
	m.migrations = []Migration{
		// 2.x -> 3.x migrations
		{
			FromVersion: "2.0.0",
			ToVersion:   "3.0.0",
			Description: "Rename 'versioning.strategy' to 'versioning.mode'",
			Component:   "buffalo.yaml",
			Breaking:    false,
			Apply: func(data map[string]interface{}) error {
				if versioning, ok := data["versioning"].(map[string]interface{}); ok {
					if strategy, exists := versioning["strategy"]; exists {
						versioning["mode"] = strategy
						delete(versioning, "strategy")
					}
				}
				return nil
			},
		},
		{
			FromVersion: "2.0.0",
			ToVersion:   "3.0.0",
			Description: "Add 'build.cache' section with default settings",
			Component:   "buffalo.yaml",
			Breaking:    false,
			Apply: func(data map[string]interface{}) error {
				build, ok := data["build"].(map[string]interface{})
				if !ok {
					build = make(map[string]interface{})
					data["build"] = build
				}
				if _, exists := build["cache"]; !exists {
					build["cache"] = map[string]interface{}{
						"enabled": true,
						"dir":     ".buffalo/cache",
					}
				}
				return nil
			},
		},

		// 3.x -> 4.x migrations
		{
			FromVersion: "3.0.0",
			ToVersion:   "4.0.0",
			Description: "Add 'workspace' section for multi-project support",
			Component:   "buffalo.yaml",
			Breaking:    false,
			Apply: func(data map[string]interface{}) error {
				if _, exists := data["workspace"]; !exists {
					data["workspace"] = map[string]interface{}{
						"enabled":  false,
						"projects": []interface{}{},
					}
				}
				return nil
			},
		},
		{
			FromVersion: "3.0.0",
			ToVersion:   "4.0.0",
			Description: "Migrate 'codegen' to 'languages' format",
			Component:   "buffalo.yaml",
			Breaking:    true,
			Apply: func(data map[string]interface{}) error {
				if codegen, ok := data["codegen"].(map[string]interface{}); ok {
					languages := make(map[string]interface{})

					// Migrate each language
					for lang, opts := range codegen {
						if langOpts, ok := opts.(map[string]interface{}); ok {
							languages[lang] = map[string]interface{}{
								"enabled": true,
								"output":  langOpts["output"],
								"options": langOpts,
							}
						}
					}

					data["languages"] = languages
					delete(data, "codegen")
				}
				return nil
			},
		},

		// 4.x -> 5.x migrations
		{
			FromVersion: "4.0.0",
			ToVersion:   "5.0.0",
			Description: "Add 'permissions' section for plugin sandboxing",
			Component:   "buffalo.yaml",
			Breaking:    false,
			Apply: func(data map[string]interface{}) error {
				if _, exists := data["permissions"]; !exists {
					data["permissions"] = map[string]interface{}{
						"plugins": map[string]interface{}{
							"allow_network": false,
							"allow_fs":      []string{"./protos", "./generated"},
							"allow_exec":    false,
						},
					}
				}
				return nil
			},
		},
		{
			FromVersion: "4.0.0",
			ToVersion:   "5.0.0",
			Description: "Add 'lsp' configuration section",
			Component:   "buffalo.yaml",
			Breaking:    false,
			Apply: func(data map[string]interface{}) error {
				if _, exists := data["lsp"]; !exists {
					data["lsp"] = map[string]interface{}{
						"enabled":           true,
						"port":              0, // 0 means auto-select
						"diagnostics":       true,
						"hover":             true,
						"completion":        true,
						"format_on_save":    true,
						"semantic_tokens":   true,
						"inlay_hints":       false,
						"code_lens":         true,
						"code_actions":      true,
						"document_symbols":  true,
						"workspace_symbols": true,
						"references":        true,
						"rename":            true,
					}
				}
				return nil
			},
		},
	}
}

// GetApplicableMigrations returns migrations needed to go from fromVersion to toVersion.
func (m *Migrator) GetApplicableMigrations(fromVersion, toVersion string) []Migration {
	var applicable []Migration

	for _, mig := range m.migrations {
		// Check if this migration applies
		if compareVersions(mig.FromVersion, fromVersion) <= 0 &&
			compareVersions(mig.ToVersion, toVersion) <= 0 &&
			compareVersions(mig.ToVersion, fromVersion) > 0 {
			applicable = append(applicable, mig)
		}
	}

	return applicable
}

// GetMigrationSteps returns migration steps without applying them.
func (m *Migrator) GetMigrationSteps(fromVersion, toVersion string) []MigrationStep {
	migrations := m.GetApplicableMigrations(fromVersion, toVersion)
	steps := make([]MigrationStep, len(migrations))

	for i, mig := range migrations {
		steps[i] = MigrationStep{
			Component:   mig.Component,
			Description: mig.Description,
			FromVersion: mig.FromVersion,
			ToVersion:   mig.ToVersion,
			Breaking:    mig.Breaking,
		}
	}

	return steps
}

// Migrate applies all applicable migrations to the config file.
func (m *Migrator) Migrate(configPath string, fromVersion, toVersion string, dryRun bool) (*MigrationResult, error) {
	result := &MigrationResult{
		Success: true,
		Steps:   []MigrationStepResult{},
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to read config file")
	}

	// Parse YAML into map for manipulation
	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "failed to parse config file")
	}

	// Create backup if not dry run
	if !dryRun && m.backupDir != "" {
		backupPath, err := m.createBackup(configPath, configData)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrIO, "failed to create backup")
		}
		result.BackupPath = backupPath
	}

	// Get applicable migrations
	migrations := m.GetApplicableMigrations(fromVersion, toVersion)

	// Apply each migration
	for _, mig := range migrations {
		stepResult := MigrationStepResult{
			Step: MigrationStep{
				Component:   mig.Component,
				Description: mig.Description,
				FromVersion: mig.FromVersion,
				ToVersion:   mig.ToVersion,
				Breaking:    mig.Breaking,
			},
			Success: true,
			Changes: []string{},
		}

		if !dryRun {
			if err := mig.Apply(configData); err != nil {
				stepResult.Success = false
				stepResult.Error = err
				result.Success = false
				result.Errors = append(result.Errors, err)
			}
		}

		stepResult.Changes = append(stepResult.Changes, mig.Description)
		result.Steps = append(result.Steps, stepResult)
	}

	// Write migrated config if not dry run
	if !dryRun && result.Success {
		if err := m.writeConfig(configPath, configData); err != nil {
			return nil, errors.Wrap(err, errors.ErrIO, "failed to write migrated config")
		}
	}

	return result, nil
}

// createBackup creates a backup of the config file.
func (m *Migrator) createBackup(configPath string, data map[string]interface{}) (string, error) {
	if m.backupDir == "" {
		return "", nil
	}

	// Create backup directory
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return "", err
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(configPath)
	backupName := fmt.Sprintf("%s.%s.bak", baseName, timestamp)
	backupPath := filepath.Join(m.backupDir, backupName)

	// Write backup
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(backupPath, yamlData, 0644); err != nil {
		return "", err
	}

	return backupPath, nil
}

// writeConfig writes the config data back to the file.
func (m *Migrator) writeConfig(configPath string, data map[string]interface{}) error {
	// Marshal with proper formatting
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	// Add header comment
	header := "# Buffalo Configuration\n# Migrated automatically by buffalo upgrade\n\n"

	return os.WriteFile(configPath, []byte(header+string(yamlData)), 0644)
}

// SaveRollbackInfo saves rollback information for later use.
func (m *Migrator) SaveRollbackInfo(info *RollbackInfo) error {
	if m.backupDir == "" {
		return errors.New(errors.ErrInvalidInput, "backup directory not set")
	}

	rollbackPath := filepath.Join(m.backupDir, "rollback.json")

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to marshal rollback info")
	}

	return os.WriteFile(rollbackPath, data, 0644)
}

// LoadRollbackInfo loads the most recent rollback information.
func (m *Migrator) LoadRollbackInfo() (*RollbackInfo, error) {
	if m.backupDir == "" {
		return nil, errors.New(errors.ErrInvalidInput, "backup directory not set")
	}

	rollbackPath := filepath.Join(m.backupDir, "rollback.json")

	data, err := os.ReadFile(rollbackPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New(errors.ErrNotFound, "no rollback information found")
		}
		return nil, errors.Wrap(err, errors.ErrIO, "failed to read rollback info")
	}

	var info RollbackInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "failed to parse rollback info")
	}

	return &info, nil
}

// Rollback restores the previous configuration from backup.
func (m *Migrator) Rollback(configPath string) error {
	info, err := m.LoadRollbackInfo()
	if err != nil {
		return err
	}

	// Restore config backup
	if info.BackupPath != "" {
		data, err := os.ReadFile(info.BackupPath)
		if err != nil {
			return errors.Wrap(err, errors.ErrIO, "failed to read backup file")
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return errors.Wrap(err, errors.ErrIO, "failed to restore config")
		}
	}

	return nil
}

// ValidateConfig validates a config file after migration.
func (m *Migrator) ValidateConfig(configPath string) error {
	_, err := config.LoadFromFile(configPath)
	if err != nil {
		return errors.Wrap(err, errors.ErrValidation, "migrated config is invalid")
	}
	return nil
}

// DetectConfigVersion attempts to detect the Buffalo version a config was created for.
func DetectConfigVersion(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to read config")
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return "", errors.Wrap(err, errors.ErrInvalidInput, "failed to parse config")
	}

	// Check for version markers
	// v5.x has 'permissions' and 'lsp' sections
	if _, hasPermissions := configData["permissions"]; hasPermissions {
		return "5.0.0", nil
	}

	// v4.x has 'workspace' section
	if _, hasWorkspace := configData["workspace"]; hasWorkspace {
		return "4.0.0", nil
	}

	// v3.x has 'build.cache' section
	if build, ok := configData["build"].(map[string]interface{}); ok {
		if _, hasCache := build["cache"]; hasCache {
			return "3.0.0", nil
		}
	}

	// v2.x has 'versioning.strategy'
	if versioning, ok := configData["versioning"].(map[string]interface{}); ok {
		if _, hasStrategy := versioning["strategy"]; hasStrategy {
			return "2.0.0", nil
		}
	}

	// Default to oldest supported version
	return "1.0.0", nil
}

// FormatChangelog formats release notes for display.
func FormatChangelog(body string, width int) string {
	if width <= 0 {
		width = 80
	}

	lines := strings.Split(body, "\n")
	var formatted []string

	for _, line := range lines {
		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			formatted = append(formatted, line)
			continue
		}

		// Wrap long lines
		if len(line) > width {
			words := strings.Fields(line)
			var currentLine string

			for _, word := range words {
				if len(currentLine)+len(word)+1 > width {
					formatted = append(formatted, currentLine)
					currentLine = word
				} else {
					if currentLine == "" {
						currentLine = word
					} else {
						currentLine += " " + word
					}
				}
			}

			if currentLine != "" {
				formatted = append(formatted, currentLine)
			}
		} else {
			formatted = append(formatted, line)
		}
	}

	return strings.Join(formatted, "\n")
}
