// Package dependency provides dependency management for proto files.
package dependency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/massonsky/buffalo/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Manager handles proto dependencies.
type Manager struct {
	workspaceDir string
	dependsDir   string
	lockFile     *LockFile
	mu           sync.RWMutex
	log          *logger.Logger
}

// NewManager creates a new dependency manager.
func NewManager(workspaceDir string, log *logger.Logger) (*Manager, error) {
	if workspaceDir == "" {
		workspaceDir = ".buffalo"
	}

	dependsDir := filepath.Join(workspaceDir, "depends")

	m := &Manager{
		workspaceDir: workspaceDir,
		dependsDir:   dependsDir,
		log:          log,
	}

	// Create depends directory if not exists
	if err := os.MkdirAll(dependsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create depends directory: %w", err)
	}

	// Load lock file if exists
	if err := m.loadLockFile(); err != nil {
		// Lock file doesn't exist yet, that's ok
		m.lockFile = &LockFile{
			Version:      "1.0",
			Dependencies: []LockedDependency{},
		}
	}

	return m, nil
}

// Install installs a dependency.
func (m *Manager) Install(ctx context.Context, dep Dependency, opts InstallOptions) (*DownloadResult, error) {
	m.log.Info("Installing dependency", logger.String("name", dep.Name))

	// Check if already installed
	if !opts.Force && !opts.Update {
		if m.isInstalled(dep.Name) {
			m.log.Info("Dependency already installed (use --force to reinstall)",
				logger.String("name", dep.Name))
			locked := m.getLockedDependency(dep.Name)
			return &DownloadResult{
				Name:      dep.Name,
				Version:   locked.Version,
				LocalPath: filepath.Join(m.dependsDir, dep.Name),
				ProtoPath: filepath.Join(m.dependsDir, dep.Name, dep.SubPath),
			}, nil
		}
	}

	// Download dependency
	downloader := NewDownloader(m.dependsDir, m.log)
	result, err := downloader.Download(ctx, dep)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", dep.Name, err)
	}

	// Update lock file
	if err := m.updateLockFile(dep, result); err != nil {
		return nil, fmt.Errorf("failed to update lock file: %w", err)
	}

	m.log.Info("Successfully installed",
		logger.String("name", dep.Name),
		logger.String("version", result.Version))
	return result, nil
}

// InstallAll installs all dependencies from config.
func (m *Manager) InstallAll(ctx context.Context, deps []Dependency, opts InstallOptions) ([]*DownloadResult, error) {
	m.log.Info("Installing dependencies", logger.Int("count", len(deps)))

	results := make([]*DownloadResult, 0, len(deps))
	for _, dep := range deps {
		result, err := m.Install(ctx, dep, opts)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	m.log.Info("All dependencies installed successfully")
	return results, nil
}

// Update updates a dependency to latest version.
func (m *Manager) Update(ctx context.Context, name string) error {
	m.log.Info("Updating dependency", logger.String("name", name))

	locked := m.getLockedDependency(name)
	if locked == nil {
		return fmt.Errorf("dependency %s not found in lock file", name)
	}

	// Create dependency from locked version
	dep := Dependency{
		Name: name,
		Source: DependencySource{
			Type: "git", // Assuming git for now
			URL:  locked.Source,
		},
		Version: "latest",
	}

	opts := InstallOptions{
		Force:        true,
		Update:       true,
		WorkspaceDir: m.workspaceDir,
	}

	_, err := m.Install(ctx, dep, opts)
	return err
}

// Remove removes a dependency.
func (m *Manager) Remove(name string) error {
	m.log.Info("Removing dependency", logger.String("name", name))

	depPath := filepath.Join(m.dependsDir, name)
	if err := os.RemoveAll(depPath); err != nil {
		return fmt.Errorf("failed to remove dependency directory: %w", err)
	}

	// Remove from lock file
	m.mu.Lock()
	defer m.mu.Unlock()

	newDeps := make([]LockedDependency, 0)
	for _, d := range m.lockFile.Dependencies {
		if d.Name != name {
			newDeps = append(newDeps, d)
		}
	}
	m.lockFile.Dependencies = newDeps

	if err := m.saveLockFile(); err != nil {
		return fmt.Errorf("failed to update lock file: %w", err)
	}

	m.log.Info("Successfully removed", logger.String("name", name))
	return nil
}

// List lists all installed dependencies.
func (m *Manager) List() []LockedDependency {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.lockFile.Dependencies
}

// GetProtoPaths returns all proto paths from dependencies.
func (m *Manager) GetProtoPaths() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	paths := make([]string, 0, len(m.lockFile.Dependencies))
	for _, dep := range m.lockFile.Dependencies {
		depPath := filepath.Join(m.dependsDir, dep.Name)
		paths = append(paths, depPath)
	}
	return paths
}

// isInstalled checks if dependency is already installed.
func (m *Manager) isInstalled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, d := range m.lockFile.Dependencies {
		if d.Name == name {
			return true
		}
	}
	return false
}

// getLockedDependency retrieves locked dependency by name.
func (m *Manager) getLockedDependency(name string) *LockedDependency {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, d := range m.lockFile.Dependencies {
		if d.Name == name {
			return &d
		}
	}
	return nil
}

// updateLockFile updates lock file with new dependency.
func (m *Manager) updateLockFile(dep Dependency, result *DownloadResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing entry
	newDeps := make([]LockedDependency, 0)
	for _, d := range m.lockFile.Dependencies {
		if d.Name != dep.Name {
			newDeps = append(newDeps, d)
		}
	}

	// Add new entry
	locked := LockedDependency{
		Name:    dep.Name,
		Version: result.Version,
		Source:  dep.Source.URL,
		Hash:    result.Hash,
		Updated: time.Now(),
	}
	newDeps = append(newDeps, locked)

	m.lockFile.Generated = time.Now()
	m.lockFile.Dependencies = newDeps
	return m.saveLockFile()
}

// loadLockFile loads the lock file.
func (m *Manager) loadLockFile() error {
	lockPath := filepath.Join(m.workspaceDir, "buffalo.lock")

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return err
	}

	var lockFile LockFile
	if err := yaml.Unmarshal(data, &lockFile); err != nil {
		return fmt.Errorf("failed to parse lock file: %w", err)
	}

	m.lockFile = &lockFile
	return nil
}

// saveLockFile saves the lock file.
func (m *Manager) saveLockFile() error {
	lockPath := filepath.Join(m.workspaceDir, "buffalo.lock")

	data, err := yaml.Marshal(m.lockFile)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	if err := os.WriteFile(lockPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}
