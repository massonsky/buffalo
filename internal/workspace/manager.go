package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
)

// Manager manages workspace operations.
type Manager struct {
	config     *Config
	configPath string
	rootDir    string
	logger     *logger.Logger
	states     map[string]*ProjectState
	mu         sync.RWMutex
}

// NewManager creates a new workspace manager.
func NewManager(configPath string, log *logger.Logger) (*Manager, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Dir(configPath)

	m := &Manager{
		config:     cfg,
		configPath: configPath,
		rootDir:    rootDir,
		logger:     log,
		states:     make(map[string]*ProjectState),
	}

	// Initialize project states
	for i := range cfg.Projects {
		m.states[cfg.Projects[i].Name] = &ProjectState{
			Project: &cfg.Projects[i],
			Status:  StatusUnknown,
		}
	}

	return m, nil
}

// Config returns the workspace configuration.
func (m *Manager) Config() *Config {
	return m.config
}

// RootDir returns the workspace root directory.
func (m *Manager) RootDir() string {
	return m.rootDir
}

// Validate validates the workspace configuration.
func (m *Manager) Validate() *ValidationResult {
	result := m.config.Validate()

	// Additional validation: check project paths exist
	for _, p := range m.config.Projects {
		projectPath := filepath.Join(m.rootDir, p.Path)
		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Project: p.Name,
				Field:   "path",
				Message: fmt.Sprintf("project path does not exist: %s", projectPath),
			})
		}
	}

	return result
}

// Build builds projects in the workspace.
func (m *Manager) Build(ctx context.Context, opts BuildOptions) ([]*BuildResult, error) {
	projects := m.selectProjects(opts)
	if len(projects) == 0 {
		return nil, errors.New(errors.ErrNotFound, "no projects to build")
	}

	// Sort by dependency order
	sorted, err := m.topologicalSort(projects)
	if err != nil {
		return nil, err
	}

	results := make([]*BuildResult, 0, len(sorted))

	for _, p := range sorted {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := m.buildProject(ctx, p, opts)
		results = append(results, result)

		// Stop on first failure unless continue-on-error
		if result.Status == StatusFailed && !opts.ContinueOnError {
			break
		}
	}

	return results, nil
}

// BuildOptions contains options for building.
type BuildOptions struct {
	// Projects to build (empty means all).
	Projects []string
	// Tags to filter by.
	Tags []string
	// Affected only builds projects affected by changes.
	Affected bool
	// Since is the git ref to compare against for affected.
	Since string
	// DryRun only shows what would be built.
	DryRun bool
	// Force rebuilds even if cached.
	Force bool
	// ContinueOnError continues building after failures.
	ContinueOnError bool
	// Parallel enables parallel builds.
	Parallel bool
	// Workers is the number of parallel workers.
	Workers int
}

func (m *Manager) selectProjects(opts BuildOptions) []*Project {
	var projects []*Project

	if opts.Affected {
		affected, _ := m.GetAffected(opts.Since)
		for _, p := range affected.DirectlyAffected {
			projects = append(projects, p)
		}
		for _, p := range affected.TransitivelyAffected {
			projects = append(projects, p)
		}
		return projects
	}

	if len(opts.Projects) > 0 {
		for _, name := range opts.Projects {
			if p := m.config.GetProject(name); p != nil {
				projects = append(projects, p)
			}
		}
		return projects
	}

	if len(opts.Tags) > 0 {
		seen := make(map[string]bool)
		for _, tag := range opts.Tags {
			for _, p := range m.config.GetProjectsByTag(tag) {
				if !seen[p.Name] {
					seen[p.Name] = true
					projects = append(projects, p)
				}
			}
		}
		return projects
	}

	return m.config.GetEnabledProjects()
}

func (m *Manager) buildProject(ctx context.Context, p *Project, opts BuildOptions) *BuildResult {
	start := time.Now()
	result := &BuildResult{
		Project: p,
		Status:  StatusBuilding,
	}

	m.mu.Lock()
	m.states[p.Name].Status = StatusBuilding
	m.mu.Unlock()

	projectPath := filepath.Join(m.rootDir, p.Path)

	// Check cache
	if !opts.Force {
		hash, _ := m.calculateProjectHash(projectPath)
		m.mu.RLock()
		state := m.states[p.Name]
		m.mu.RUnlock()

		if state.Hash == hash && state.Status == StatusSuccess {
			result.Status = StatusSuccess
			result.Duration = time.Since(start)
			result.CacheHit = true
			m.logger.Info("⏭️  Skipped (cached)", logger.String("project", p.Name))
			return result
		}
	}

	if opts.DryRun {
		m.logger.Info("🔨 Would build", logger.String("project", p.Name))
		result.Status = StatusSkipped
		result.Duration = time.Since(start)
		return result
	}

	m.logger.Info("🔨 Building", logger.String("project", p.Name))

	// Find buffalo config in project
	configFile := "buffalo.yaml"
	if p.Config != "" {
		configFile = p.Config
	}

	// Run buffalo build
	cmd := exec.CommandContext(ctx, "buffalo", "build", "--config", configFile)
	cmd.Dir = projectPath
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		result.Status = StatusFailed
		result.Error = err
		m.logger.Error("❌ Build failed",
			logger.String("project", p.Name),
			logger.Any("error", err))
	} else {
		result.Status = StatusSuccess
		m.logger.Info("✅ Build succeeded", logger.String("project", p.Name))
	}

	result.Duration = time.Since(start)

	// Update state
	m.mu.Lock()
	state := m.states[p.Name]
	state.Status = result.Status
	state.LastBuild = time.Now()
	state.Error = result.Error
	if result.Status == StatusSuccess {
		state.Hash, _ = m.calculateProjectHash(projectPath)
	}
	m.mu.Unlock()

	return result
}

func (m *Manager) topologicalSort(projects []*Project) ([]*Project, error) {
	// Build subset graph
	inSubset := make(map[string]bool)
	for _, p := range projects {
		inSubset[p.Name] = true
	}

	// Calculate in-degree
	inDegree := make(map[string]int)
	for _, p := range projects {
		inDegree[p.Name] = 0
	}

	for _, p := range projects {
		for _, dep := range p.DependsOn {
			if inSubset[dep] {
				inDegree[p.Name]++
			}
		}
	}

	// Kahn's algorithm
	var queue []*Project
	for _, p := range projects {
		if inDegree[p.Name] == 0 {
			queue = append(queue, p)
		}
	}

	var sorted []*Project
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		sorted = append(sorted, p)

		// Find dependents
		for _, other := range projects {
			for _, dep := range other.DependsOn {
				if dep == p.Name && inSubset[other.Name] {
					inDegree[other.Name]--
					if inDegree[other.Name] == 0 {
						queue = append(queue, other)
					}
				}
			}
		}
	}

	if len(sorted) != len(projects) {
		return nil, errors.New(errors.ErrDependency, "circular dependency detected")
	}

	return sorted, nil
}

// GetAffected returns projects affected by changes.
func (m *Manager) GetAffected(since string) (*AffectedResult, error) {
	if since == "" {
		since = "HEAD~1"
	}

	result := &AffectedResult{
		ChangedFiles:         []string{},
		DirectlyAffected:     []*Project{},
		TransitivelyAffected: []*Project{},
		RecommendedBuild:     []string{},
	}

	// Get changed files from git
	cmd := exec.Command("git", "diff", "--name-only", since)
	cmd.Dir = m.rootDir
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to get git diff")
	}

	changedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(changedFiles) == 1 && changedFiles[0] == "" {
		changedFiles = []string{}
	}
	result.ChangedFiles = changedFiles

	// Find directly affected projects
	directlyAffected := make(map[string]bool)
	for _, file := range changedFiles {
		for _, p := range m.config.Projects {
			if strings.HasPrefix(file, p.Path+"/") || file == p.Path {
				if !directlyAffected[p.Name] {
					directlyAffected[p.Name] = true
					result.DirectlyAffected = append(result.DirectlyAffected, m.config.GetProject(p.Name))
				}
			}
		}
	}

	// Find transitively affected projects
	graph := m.config.BuildDependencyGraph()
	transitivelyAffected := make(map[string]bool)

	var findDependents func(name string)
	findDependents = func(name string) {
		for _, dependent := range graph.Reverse[name] {
			if !directlyAffected[dependent] && !transitivelyAffected[dependent] {
				transitivelyAffected[dependent] = true
				result.TransitivelyAffected = append(result.TransitivelyAffected, m.config.GetProject(dependent))
				findDependents(dependent)
			}
		}
	}

	for name := range directlyAffected {
		findDependents(name)
	}

	// Build recommended list
	for _, p := range result.DirectlyAffected {
		result.RecommendedBuild = append(result.RecommendedBuild, p.Name)
	}
	for _, p := range result.TransitivelyAffected {
		result.RecommendedBuild = append(result.RecommendedBuild, p.Name)
	}

	return result, nil
}

// List returns all projects in the workspace.
func (m *Manager) List() []*Project {
	return m.config.GetEnabledProjects()
}

// ListByTag returns projects matching the given tag.
func (m *Manager) ListByTag(tag string) []*Project {
	return m.config.GetProjectsByTag(tag)
}

// GetState returns the current state of a project.
func (m *Manager) GetState(name string) *ProjectState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.states[name]
}

// GetAllTags returns all unique tags in the workspace.
func (m *Manager) GetAllTags() []string {
	tags := make(map[string]bool)
	for _, p := range m.config.Projects {
		for _, t := range p.Tags {
			tags[t] = true
		}
	}

	result := make([]string, 0, len(tags))
	for t := range tags {
		result = append(result, t)
	}
	sort.Strings(result)
	return result
}

func (m *Manager) calculateProjectHash(projectPath string) (string, error) {
	h := sha256.New()

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
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

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		h.Write([]byte(path))
		h.Write(data)
		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
