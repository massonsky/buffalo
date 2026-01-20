package versioning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/massonsky/buffalo/pkg/errors"
)

// Strategy represents versioning strategy
type Strategy string

const (
	StrategyHash      Strategy = "hash"      // Use content hash as version
	StrategyTimestamp Strategy = "timestamp" // Use timestamp as version
	StrategySemantic  Strategy = "semantic"  // Use semantic versioning (v1, v2, v3)
	StrategyGit       Strategy = "git"       // Use git commit hash
)

// OutputFormat represents how versioned files are organized
type OutputFormat string

const (
	FormatDirectory OutputFormat = "directory" // Create version directories (v1/, v2/)
	FormatSuffix    OutputFormat = "suffix"    // Add version suffix (_v1, _v2)
)

// Manager manages file versioning
type Manager struct {
	enabled      bool
	strategy     Strategy
	outputFormat OutputFormat
	keepVersions int
	stateDir     string // Directory to store version state
}

// Options contains versioning options
type Options struct {
	Enabled      bool
	Strategy     Strategy
	OutputFormat OutputFormat
	KeepVersions int
	StateDir     string
}

// New creates a new versioning manager
func New(opts Options) *Manager {
	return &Manager{
		enabled:      opts.Enabled,
		strategy:     opts.Strategy,
		outputFormat: opts.OutputFormat,
		keepVersions: opts.KeepVersions,
		stateDir:     opts.StateDir,
	}
}

// FileVersion represents a versioned file
type FileVersion struct {
	ProtoPath   string    // Path to proto file
	Version     string    // Version identifier
	Hash        string    // Content hash
	Timestamp   time.Time // Creation time
	OutputPath  string    // Path to generated output
	GeneratedAt time.Time // When it was generated
}

// IsEnabled returns true if versioning is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// ComputeHash computes SHA256 hash of a file
func (m *Manager) ComputeHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to open file: %s", filePath)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to compute hash: %s", filePath)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetCurrentVersion gets the current version for a proto file
func (m *Manager) GetCurrentVersion(protoPath string) (*FileVersion, error) {
	if !m.enabled {
		return nil, nil
	}

	stateFile := m.getStateFile(protoPath)
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil, nil // No previous version
	}

	// Read state file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to read state file: %s", stateFile)
	}

	// Parse state (format: hash|version|timestamp|outputPath)
	parts := strings.Split(string(data), "|")
	if len(parts) < 3 {
		return nil, nil // Invalid state
	}

	timestamp, _ := time.Parse(time.RFC3339, parts[2])
	outputPath := ""
	if len(parts) >= 4 {
		outputPath = parts[3]
	}
	return &FileVersion{
		ProtoPath:  protoPath,
		Hash:       parts[0],
		Version:    parts[1],
		Timestamp:  timestamp,
		OutputPath: outputPath,
	}, nil
}

// ShouldGenerateNewVersion checks if a new version should be generated
func (m *Manager) ShouldGenerateNewVersion(protoPath string) (bool, error) {
	if !m.enabled {
		return true, nil // Always generate if versioning is disabled
	}

	currentHash, err := m.ComputeHash(protoPath)
	if err != nil {
		return false, err
	}

	prevVersion, err := m.GetCurrentVersion(protoPath)
	if err != nil {
		return false, err
	}

	if prevVersion == nil {
		return true, nil // First version
	}

	// Check if content changed
	if currentHash != prevVersion.Hash {
		return true, nil
	}

	// Even if hash matches, check if the output directory still exists
	// This handles the case where generated files were deleted but cache/state remains
	if prevVersion.OutputPath != "" {
		if _, err := os.Stat(prevVersion.OutputPath); os.IsNotExist(err) {
			return true, nil // Output was deleted, need to regenerate
		}
	}

	return false, nil
}

// GenerateVersion generates a new version identifier
func (m *Manager) GenerateVersion(protoPath string) (string, error) {
	switch m.strategy {
	case StrategyHash:
		hash, err := m.ComputeHash(protoPath)
		if err != nil {
			return "", err
		}
		return hash[:8], nil // Short hash

	case StrategyTimestamp:
		return time.Now().Format("20060102150405"), nil

	case StrategySemantic:
		prevVersion, err := m.GetCurrentVersion(protoPath)
		if err != nil {
			return "", err
		}
		if prevVersion == nil {
			return "v1", nil
		}
		// Parse version number and increment
		versionNum := 1
		if strings.HasPrefix(prevVersion.Version, "v") {
			fmt.Sscanf(prevVersion.Version, "v%d", &versionNum)
			versionNum++
		}
		return fmt.Sprintf("v%d", versionNum), nil

	case StrategyGit:
		// TODO: Implement git hash retrieval
		return "", errors.New(errors.ErrConfig, "git versioning not implemented yet")

	default:
		return "", errors.New(errors.ErrConfig, "unknown versioning strategy: %s", m.strategy)
	}
}

// GetVersionedOutputPath returns the output path with version
func (m *Manager) GetVersionedOutputPath(baseOutputPath, version string) string {
	if !m.enabled {
		return baseOutputPath
	}

	// For directory output format, insert version subdirectory
	switch m.outputFormat {
	case FormatDirectory:
		// Insert version as subdirectory: ./generated -> ./generated/v1
		return filepath.Join(baseOutputPath, version)

	case FormatSuffix:
		// Add version suffix to the directory name: ./generated -> ./generated_v1
		return baseOutputPath + "_" + version

	default:
		return baseOutputPath
	}
}

// SaveVersion saves the version state
func (m *Manager) SaveVersion(protoPath, version, outputPath string) error {
	if !m.enabled {
		return nil
	}

	hash, err := m.ComputeHash(protoPath)
	if err != nil {
		return err
	}

	stateFile := m.getStateFile(protoPath)
	if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to create state directory")
	}

	// Write state (hash|version|timestamp|outputPath)
	state := fmt.Sprintf("%s|%s|%s|%s", hash, version, time.Now().Format(time.RFC3339), outputPath)
	if err := os.WriteFile(stateFile, []byte(state), 0644); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to write state file: %s", stateFile)
	}

	return nil
}

// CleanupOldVersions removes old versions keeping only the latest N versions
func (m *Manager) CleanupOldVersions(outputDir string) error {
	if !m.enabled || m.keepVersions == 0 {
		return nil // Keep all versions
	}

	if m.outputFormat != FormatDirectory {
		return nil // Only cleanup directory-based versions
	}

	// List version directories
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to read output directory: %s", outputDir)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "v") {
			versions = append(versions, entry.Name())
		}
	}

	// Sort versions (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})

	// Remove old versions
	if len(versions) > m.keepVersions {
		for _, v := range versions[m.keepVersions:] {
			versionDir := filepath.Join(outputDir, v)
			if err := os.RemoveAll(versionDir); err != nil {
				return errors.Wrap(err, errors.ErrIO, "failed to remove old version: %s", versionDir)
			}
		}
	}

	return nil
}

// getStateFile returns the path to the state file for a proto file
func (m *Manager) getStateFile(protoPath string) string {
	// Use proto file path as base for state file name
	hash := sha256.Sum256([]byte(protoPath))
	stateFileName := hex.EncodeToString(hash[:])[:16] + ".state"
	return filepath.Join(m.stateDir, stateFileName)
}
