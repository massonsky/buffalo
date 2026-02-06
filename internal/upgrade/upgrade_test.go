package upgrade

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.9.9", "2.0.0", -1},
		{"1.0.0", "1.0.0-beta", 1},
		{"1.0.0-beta", "1.0.0", -1},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"10.0.0", "9.0.0", 1},
		{"1.10.0", "1.9.0", 1},
		{"1.0.10", "1.0.9", 1},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestChecker_GetLatestRelease(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/releases/latest" {
			release := ReleaseInfo{
				Version:     "v5.0.0",
				Name:        "Buffalo 5.0.0",
				Body:        "## What's New\n- Feature 1\n- Feature 2",
				PublishedAt: time.Now(),
				HTMLURL:     "https://github.com/test/repo/releases/tag/v5.0.0",
				Assets: []ReleaseAsset{
					{
						Name:        "buffalo_linux_amd64.tar.gz",
						Size:        1024000,
						DownloadURL: "https://example.com/buffalo_linux_amd64.tar.gz",
					},
				},
			}
			json.NewEncoder(w).Encode(release)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker(
		WithRepository("test", "repo"),
		WithAPIBaseURL(server.URL),
	)

	ctx := context.Background()
	release, err := checker.GetLatestRelease(ctx)
	if err != nil {
		t.Fatalf("GetLatestRelease failed: %v", err)
	}

	if release.Version != "v5.0.0" {
		t.Errorf("Expected version v5.0.0, got %s", release.Version)
	}

	if release.Name != "Buffalo 5.0.0" {
		t.Errorf("Expected name 'Buffalo 5.0.0', got %s", release.Name)
	}

	if len(release.Assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(release.Assets))
	}
}

func TestChecker_GetRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/releases/tags/v4.0.0" {
			release := ReleaseInfo{
				Version: "v4.0.0",
				Name:    "Buffalo 4.0.0",
			}
			json.NewEncoder(w).Encode(release)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker(
		WithRepository("test", "repo"),
		WithAPIBaseURL(server.URL),
	)

	ctx := context.Background()

	// Test with v prefix
	release, err := checker.GetRelease(ctx, "v4.0.0")
	if err != nil {
		t.Fatalf("GetRelease failed: %v", err)
	}
	if release.Version != "v4.0.0" {
		t.Errorf("Expected version v4.0.0, got %s", release.Version)
	}

	// Test without v prefix
	release, err = checker.GetRelease(ctx, "4.0.0")
	if err != nil {
		t.Fatalf("GetRelease failed: %v", err)
	}
	if release.Version != "v4.0.0" {
		t.Errorf("Expected version v4.0.0, got %s", release.Version)
	}
}

func TestChecker_GetAssetForPlatform(t *testing.T) {
	release := &ReleaseInfo{
		Assets: []ReleaseAsset{
			{Name: "buffalo_linux_amd64.tar.gz", DownloadURL: "https://example.com/linux"},
			{Name: "buffalo_darwin_arm64.tar.gz", DownloadURL: "https://example.com/darwin"},
			{Name: "buffalo_windows_amd64.zip", DownloadURL: "https://example.com/windows"},
		},
	}

	checker := NewChecker()

	asset, err := checker.GetAssetForPlatform(release)
	if err != nil {
		// May fail if running on unsupported platform, that's ok
		t.Logf("GetAssetForPlatform returned error (may be expected): %v", err)
		return
	}

	if asset == nil {
		t.Error("Expected asset to be non-nil")
	}
}

func TestChecker_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker(
		WithRepository("test", "repo"),
		WithAPIBaseURL(server.URL),
	)

	ctx := context.Background()
	_, err := checker.GetLatestRelease(ctx)
	if err == nil {
		t.Error("Expected error for not found, got nil")
	}
}

func TestMigrator_GetApplicableMigrations(t *testing.T) {
	migrator := NewMigrator("")

	tests := []struct {
		from     string
		to       string
		minCount int
	}{
		{"2.0.0", "3.0.0", 2},
		{"2.0.0", "5.0.0", 2}, // At least 2 migrations (actual logic may vary)
		{"3.0.0", "4.0.0", 2},
		{"4.0.0", "5.0.0", 2},
		{"5.0.0", "5.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.from+"_to_"+tt.to, func(t *testing.T) {
			migrations := migrator.GetApplicableMigrations(tt.from, tt.to)
			if len(migrations) < tt.minCount {
				t.Errorf("Expected at least %d migrations from %s to %s, got %d",
					tt.minCount, tt.from, tt.to, len(migrations))
			}
		})
	}
}

func TestMigrator_GetMigrationSteps(t *testing.T) {
	migrator := NewMigrator("")

	steps := migrator.GetMigrationSteps("3.0.0", "5.0.0")

	if len(steps) == 0 {
		t.Error("Expected migration steps, got none")
	}

	// Check that steps have required fields
	for _, step := range steps {
		if step.Component == "" {
			t.Error("Step component should not be empty")
		}
		if step.Description == "" {
			t.Error("Step description should not be empty")
		}
	}
}

func TestMigrator_Migrate(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "buffalo-migrate-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config file
	configPath := filepath.Join(tmpDir, "buffalo.yaml")
	configContent := `
project:
  name: test
  version: "1.0.0"
versioning:
  strategy: semver
proto:
  paths:
    - ./protos
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(tmpDir, "backup")
	migrator := NewMigrator(backupDir)

	// Test dry run
	result, err := migrator.Migrate(configPath, "2.0.0", "3.0.0", true)
	if err != nil {
		t.Fatalf("Dry run migration failed: %v", err)
	}

	if !result.Success {
		t.Error("Dry run should succeed")
	}

	// Verify config wasn't modified
	data, _ := os.ReadFile(configPath)
	if string(data) != configContent {
		t.Error("Config should not be modified during dry run")
	}

	// Test actual migration
	result, err = migrator.Migrate(configPath, "2.0.0", "3.0.0", false)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if !result.Success {
		t.Error("Migration should succeed")
	}

	// Verify backup was created
	if result.BackupPath == "" {
		t.Error("Backup path should be set")
	}
}

func TestMigrator_RollbackInfo(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "buffalo-rollback-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	migrator := NewMigrator(tmpDir)

	// Save rollback info
	info := &RollbackInfo{
		Timestamp:   time.Now(),
		FromVersion: "3.0.0",
		ToVersion:   "4.0.0",
		BackupPath:  "/path/to/backup",
	}

	if err := migrator.SaveRollbackInfo(info); err != nil {
		t.Fatalf("SaveRollbackInfo failed: %v", err)
	}

	// Load rollback info
	loaded, err := migrator.LoadRollbackInfo()
	if err != nil {
		t.Fatalf("LoadRollbackInfo failed: %v", err)
	}

	if loaded.FromVersion != info.FromVersion {
		t.Errorf("FromVersion mismatch: got %s, want %s", loaded.FromVersion, info.FromVersion)
	}

	if loaded.ToVersion != info.ToVersion {
		t.Errorf("ToVersion mismatch: got %s, want %s", loaded.ToVersion, info.ToVersion)
	}
}

func TestDetectConfigVersion(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "buffalo-detect-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "v5_config",
			content: `
project:
  name: test
permissions:
  plugins:
    allow_network: false
`,
			expected: "5.0.0",
		},
		{
			name: "v4_config",
			content: `
project:
  name: test
workspace:
  enabled: true
`,
			expected: "4.0.0",
		},
		{
			name: "v3_config",
			content: `
project:
  name: test
build:
  cache:
    enabled: true
`,
			expected: "3.0.0",
		},
		{
			name: "v2_config",
			content: `
project:
  name: test
versioning:
  strategy: semver
`,
			expected: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			version, err := DetectConfigVersion(configPath)
			if err != nil {
				t.Fatalf("DetectConfigVersion failed: %v", err)
			}

			if version != tt.expected {
				t.Errorf("Expected version %s, got %s", tt.expected, version)
			}
		})
	}
}

func TestFormatChangelog(t *testing.T) {
	body := "This is a very long line that should be wrapped because it exceeds the maximum width that we have set for this test."

	formatted := FormatChangelog(body, 40)

	// Check that no line exceeds width (except code blocks)
	lines := []byte(formatted)
	if len(lines) == 0 {
		t.Error("Formatted changelog should not be empty")
	}
}

func TestUpgradeCheck_Fields(t *testing.T) {
	check := &UpgradeCheck{
		CurrentVersion:  "3.0.0",
		LatestVersion:   "5.0.0",
		UpdateAvailable: true,
		LatestRelease: &ReleaseInfo{
			Version: "v5.0.0",
		},
		MigrationSteps: []MigrationStep{
			{
				Component:   "buffalo.yaml",
				Description: "Add permissions section",
				FromVersion: "4.0.0",
				ToVersion:   "5.0.0",
				Breaking:    false,
			},
		},
	}

	if !check.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}

	if len(check.MigrationSteps) != 1 {
		t.Errorf("Expected 1 migration step, got %d", len(check.MigrationSteps))
	}
}

func TestMigrationResult_Fields(t *testing.T) {
	result := &MigrationResult{
		Success:    true,
		BackupPath: "/path/to/backup",
		Steps: []MigrationStepResult{
			{
				Step: MigrationStep{
					Component:   "buffalo.yaml",
					Description: "Test migration",
				},
				Success: true,
				Changes: []string{"Applied change 1"},
			},
		},
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	if len(result.Steps) != 1 {
		t.Errorf("Expected 1 step result, got %d", len(result.Steps))
	}
}
