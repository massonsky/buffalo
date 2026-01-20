// Package config provides configuration management for Buffalo.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LockFile represents the buffalo.lock file structure
type LockFile struct {
	// Version of the lock file format
	Version string `yaml:"version" json:"version"`

	// GeneratedAt is when the lock file was generated
	GeneratedAt time.Time `yaml:"generated_at" json:"generated_at"`

	// ConfigHash is the SHA256 hash of the buffalo.yaml file
	ConfigHash string `yaml:"config_hash" json:"config_hash"`

	// Project information
	Project ProjectLock `yaml:"project" json:"project"`

	// Proto files information
	Proto ProtoLock `yaml:"proto" json:"proto"`

	// Languages configuration with resolved values
	Languages LanguagesLock `yaml:"languages" json:"languages"`

	// Dependencies with resolved versions and hashes
	Dependencies []DependencyLock `yaml:"dependencies" json:"dependencies"`

	// Tools information
	Tools ToolsLock `yaml:"tools" json:"tools"`
}

// ProjectLock contains locked project information
type ProjectLock struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}

// ProtoLock contains locked proto files information
type ProtoLock struct {
	// Files is a map of proto file path to its hash
	Files map[string]ProtoFileLock `yaml:"files" json:"files"`

	// ImportPaths are the resolved import paths
	ImportPaths []string `yaml:"import_paths" json:"import_paths"`

	// TotalFiles count
	TotalFiles int `yaml:"total_files" json:"total_files"`
}

// ProtoFileLock contains locked information for a single proto file
type ProtoFileLock struct {
	Hash         string    `yaml:"hash" json:"hash"`
	Size         int64     `yaml:"size" json:"size"`
	ModifiedAt   time.Time `yaml:"modified_at" json:"modified_at"`
	Package      string    `yaml:"package,omitempty" json:"package,omitempty"`
	Dependencies []string  `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

// LanguagesLock contains locked language configurations
type LanguagesLock struct {
	Python *PythonLock `yaml:"python,omitempty" json:"python,omitempty"`
	Go     *GoLock     `yaml:"go,omitempty" json:"go,omitempty"`
	Rust   *RustLock   `yaml:"rust,omitempty" json:"rust,omitempty"`
	Cpp    *CppLock    `yaml:"cpp,omitempty" json:"cpp,omitempty"`
}

// PythonLock contains locked Python configuration
type PythonLock struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// WorkDir is the working directory prefix for imports
	WorkDir string `yaml:"workdir,omitempty" json:"workdir,omitempty"`

	// ExcludeImports are prefixes to exclude from import fixing (auto-detected + user-defined)
	ExcludeImports []string `yaml:"exclude_imports" json:"exclude_imports"`

	// OutputDir is the resolved output directory
	OutputDir string `yaml:"output_dir" json:"output_dir"`
}

// GoLock contains locked Go configuration
type GoLock struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Module    string `yaml:"module,omitempty" json:"module,omitempty"`
	OutputDir string `yaml:"output_dir" json:"output_dir"`
}

// RustLock contains locked Rust configuration
type RustLock struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Generator string `yaml:"generator,omitempty" json:"generator,omitempty"`
	OutputDir string `yaml:"output_dir" json:"output_dir"`
}

// CppLock contains locked C++ configuration
type CppLock struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	OutputDir string `yaml:"output_dir" json:"output_dir"`
}

// DependencyLock contains locked dependency information
type DependencyLock struct {
	Name    string `yaml:"name" json:"name"`
	Type    string `yaml:"type" json:"type"` // git, http, local
	URL     string `yaml:"url,omitempty" json:"url,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
	Hash    string `yaml:"hash" json:"hash"`

	// ExternalPrefixes are the detected external import prefixes from this dependency
	ExternalPrefixes []string `yaml:"external_prefixes,omitempty" json:"external_prefixes,omitempty"`
}

// ToolsLock contains locked tools information
type ToolsLock struct {
	Protoc        string `yaml:"protoc,omitempty" json:"protoc,omitempty"`
	ProtocVersion string `yaml:"protoc_version,omitempty" json:"protoc_version,omitempty"`
}

// DefaultExternalPrefixes returns the default list of external import prefixes
// that should not be modified during import fixing
func DefaultExternalPrefixes() []string {
	return []string{
		"google.",       // google.protobuf, google.api, google.rpc, etc.
		"grpc",          // grpc module itself
		"protobuf",      // protobuf module
		"googleapis",    // googleapis packages
		"opentelemetry", // OpenTelemetry protos
		"envoy.",        // Envoy proxy protos
		"validate",      // protoc-gen-validate
		"buf.",          // Buf ecosystem
	}
}

// LockFileManager manages buffalo.lock file operations
type LockFileManager struct {
	configPath string
	lockPath   string
}

// NewLockFileManager creates a new lock file manager
func NewLockFileManager(configPath string) *LockFileManager {
	dir := filepath.Dir(configPath)
	return &LockFileManager{
		configPath: configPath,
		lockPath:   filepath.Join(dir, "buffalo.lock"),
	}
}

// GetLockPath returns the path to the lock file
func (m *LockFileManager) GetLockPath() string {
	return m.lockPath
}

// NeedsRegeneration checks if the lock file needs to be regenerated
func (m *LockFileManager) NeedsRegeneration() (bool, string, error) {
	// Check if lock file exists
	if _, err := os.Stat(m.lockPath); os.IsNotExist(err) {
		return true, "lock file does not exist", nil
	}

	// Load existing lock file
	lock, err := m.Load()
	if err != nil {
		return true, fmt.Sprintf("failed to load lock file: %v", err), nil
	}

	// Calculate current config hash
	currentHash, err := m.calculateConfigHash()
	if err != nil {
		return true, fmt.Sprintf("failed to calculate config hash: %v", err), nil
	}

	// Compare hashes
	if lock.ConfigHash != currentHash {
		return true, "config file has changed", nil
	}

	return false, "", nil
}

// Generate generates a new lock file from the config
func (m *LockFileManager) Generate(cfg *Config, protoFiles []string) (*LockFile, error) {
	// Calculate config hash
	configHash, err := m.calculateConfigHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate config hash: %w", err)
	}

	lock := &LockFile{
		Version:     "1",
		GeneratedAt: time.Now().UTC(),
		ConfigHash:  configHash,
		Project: ProjectLock{
			Name:    cfg.Project.Name,
			Version: cfg.Project.Version,
		},
		Proto: ProtoLock{
			Files:       make(map[string]ProtoFileLock),
			ImportPaths: cfg.Proto.ImportPaths,
			TotalFiles:  len(protoFiles),
		},
	}

	// Hash proto files
	for _, protoFile := range protoFiles {
		fileLock, err := m.hashProtoFile(protoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to hash proto file %s: %w", protoFile, err)
		}
		// Use relative path as key
		relPath, _ := filepath.Rel(filepath.Dir(m.configPath), protoFile)
		if relPath == "" {
			relPath = protoFile
		}
		lock.Proto.Files[relPath] = fileLock
	}

	// Lock languages
	if cfg.Languages.Python.Enabled {
		excludeImports := m.resolveExcludeImports(cfg, lock.Dependencies)
		lock.Languages.Python = &PythonLock{
			Enabled:        true,
			WorkDir:        cfg.Languages.Python.WorkDir,
			ExcludeImports: excludeImports,
			OutputDir:      cfg.GetOutputDir("python"),
		}
	}

	if cfg.Languages.Go.Enabled {
		lock.Languages.Go = &GoLock{
			Enabled:   true,
			Module:    cfg.Languages.Go.Module,
			OutputDir: cfg.GetOutputDir("go"),
		}
	}

	if cfg.Languages.Rust.Enabled {
		lock.Languages.Rust = &RustLock{
			Enabled:   true,
			Generator: cfg.Languages.Rust.Generator,
			OutputDir: cfg.GetOutputDir("rust"),
		}
	}

	if cfg.Languages.Cpp.Enabled {
		lock.Languages.Cpp = &CppLock{
			Enabled:   true,
			Namespace: cfg.Languages.Cpp.Namespace,
			OutputDir: cfg.GetOutputDir("cpp"),
		}
	}

	// Lock dependencies
	for _, dep := range cfg.Dependencies {
		depLock := DependencyLock{
			Name:    dep.Name,
			Type:    dep.Source.Type,
			URL:     dep.Source.URL,
			Version: dep.Version,
			Path:    dep.Source.Path,
		}

		// Calculate dependency hash if possible
		if dep.Source.Path != "" {
			hash, err := m.hashDirectory(dep.Source.Path)
			if err == nil {
				depLock.Hash = hash
			}
		}

		// Detect external prefixes from dependency
		depLock.ExternalPrefixes = m.detectExternalPrefixes(dep)

		lock.Dependencies = append(lock.Dependencies, depLock)
	}

	// Detect protoc version
	lock.Tools.Protoc = "protoc"
	lock.Tools.ProtocVersion = m.detectProtocVersion()

	return lock, nil
}

// Save saves the lock file to disk
func (m *LockFileManager) Save(lock *LockFile) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	header := []byte("# Buffalo Lock File - DO NOT EDIT MANUALLY\n# Generated by Buffalo build system\n# This file is auto-generated from buffalo.yaml\n\n")
	content := append(header, data...)

	if err := os.WriteFile(m.lockPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// Load loads the lock file from disk
func (m *LockFileManager) Load() (*LockFile, error) {
	data, err := os.ReadFile(m.lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lock LockFile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock file: %w", err)
	}

	return &lock, nil
}

// calculateConfigHash calculates SHA256 hash of the config file
func (m *LockFileManager) calculateConfigHash() (string, error) {
	return hashFile(m.configPath)
}

// hashProtoFile calculates hash and metadata for a proto file
func (m *LockFileManager) hashProtoFile(path string) (ProtoFileLock, error) {
	var lock ProtoFileLock

	info, err := os.Stat(path)
	if err != nil {
		return lock, err
	}

	hash, err := hashFile(path)
	if err != nil {
		return lock, err
	}

	lock.Hash = hash
	lock.Size = info.Size()
	lock.ModifiedAt = info.ModTime().UTC()

	// Try to extract package from proto file
	lock.Package = m.extractProtoPackage(path)

	return lock, nil
}

// extractProtoPackage extracts the package name from a proto file
func (m *LockFileManager) extractProtoPackage(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			// Extract package name: "package foo.bar;" -> "foo.bar"
			pkg := strings.TrimPrefix(line, "package ")
			pkg = strings.TrimSuffix(pkg, ";")
			return strings.TrimSpace(pkg)
		}
	}

	return ""
}

// hashDirectory calculates a combined hash of all files in a directory
func (m *LockFileManager) hashDirectory(dir string) (string, error) {
	h := sha256.New()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only hash proto files
		if !strings.HasSuffix(path, ".proto") {
			return nil
		}

		fileHash, err := hashFile(path)
		if err != nil {
			return err
		}

		h.Write([]byte(fileHash))
		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// resolveExcludeImports resolves the full list of import prefixes to exclude
func (m *LockFileManager) resolveExcludeImports(cfg *Config, deps []DependencyLock) []string {
	// Start with defaults
	prefixes := DefaultExternalPrefixes()

	// Add user-defined prefixes from config
	prefixes = append(prefixes, cfg.Languages.Python.ExcludeImports...)

	// Add prefixes detected from dependencies
	for _, dep := range deps {
		prefixes = append(prefixes, dep.ExternalPrefixes...)
	}

	// Deduplicate and sort
	return uniqueSortedStrings(prefixes)
}

// detectExternalPrefixes detects external import prefixes from a dependency
func (m *LockFileManager) detectExternalPrefixes(dep interface{}) []string {
	// For now, detect based on known dependency names
	// This can be extended to parse proto files in the dependency

	// Common well-known dependencies and their prefixes
	knownPrefixes := map[string][]string{
		"googleapis":          {"google."},
		"google-api":          {"google.api"},
		"protobuf":            {"google.protobuf"},
		"grpc":                {"grpc"},
		"envoy":               {"envoy."},
		"opentelemetry":       {"opentelemetry"},
		"buf":                 {"buf."},
		"validate":            {"validate"},
		"protoc-gen-validate": {"validate"},
	}

	var prefixes []string

	// Type assert to access dependency fields
	switch d := dep.(type) {
	case DependencyLock:
		name := strings.ToLower(d.Name)
		for key, pref := range knownPrefixes {
			if strings.Contains(name, key) {
				prefixes = append(prefixes, pref...)
			}
		}
	}

	return prefixes
}

// detectProtocVersion tries to detect the installed protoc version
func (m *LockFileManager) detectProtocVersion() string {
	// This would run "protoc --version" in a real implementation
	// For now, return empty string
	return ""
}

// hashFile calculates SHA256 hash of a file
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// uniqueSortedStrings returns unique sorted strings
func uniqueSortedStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, v := range s {
		if v != "" && !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	sort.Strings(result)
	return result
}

// ToJSON returns the lock file as JSON (for debugging)
func (l *LockFile) ToJSON() (string, error) {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetExcludeImports returns the resolved exclude imports for Python
func (l *LockFile) GetExcludeImports() []string {
	if l.Languages.Python != nil {
		return l.Languages.Python.ExcludeImports
	}
	return DefaultExternalPrefixes()
}

// HasProtoFileChanged checks if a specific proto file has changed
func (l *LockFile) HasProtoFileChanged(path string, currentHash string) bool {
	if fileLock, ok := l.Proto.Files[path]; ok {
		return fileLock.Hash != currentHash
	}
	// File not in lock = new file = changed
	return true
}
