package workspace

import (
	"os"
	"path/filepath"

	"github.com/massonsky/buffalo/pkg/errors"
	"gopkg.in/yaml.v3"
)

// DefaultConfigFiles are the default workspace configuration file names.
var DefaultConfigFiles = []string{
	"buffalo-workspace.yaml",
	"buffalo-workspace.yml",
	".buffalo-workspace.yaml",
	".buffalo-workspace.yml",
}

// LoadConfig loads workspace configuration from a file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to read workspace config")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "failed to parse workspace config")
	}

	// Set defaults
	if cfg.Workspace.Name == "" {
		cfg.Workspace.Name = filepath.Base(filepath.Dir(path))
	}

	return &cfg, nil
}

// FindConfig searches for a workspace configuration file.
func FindConfig(dir string) (string, error) {
	for _, name := range DefaultConfigFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try parent directory
	parent := filepath.Dir(dir)
	if parent != dir {
		return FindConfig(parent)
	}

	return "", errors.New(errors.ErrNotFound, "workspace configuration not found")
}

// SaveConfig saves workspace configuration to a file.
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, errors.ErrInvalidInput, "failed to marshal workspace config")
	}

	header := "# Buffalo Workspace Configuration\n# https://github.com/massonsky/buffalo\n\n"

	if err := os.WriteFile(path, []byte(header+string(data)), 0644); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to write workspace config")
	}

	return nil
}

// InitConfig creates a new workspace configuration with defaults.
func InitConfig(name string) *Config {
	return &Config{
		Workspace: WorkspaceSettings{
			Name:    name,
			Version: "1.0.0",
		},
		Projects: []Project{},
		Policies: Policies{
			ConsistentVersions: true,
			SharedDependencies: true,
			NoCircularDeps:     true,
		},
	}
}

// Validate validates the workspace configuration.
func (c *Config) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
	}

	// Check workspace name
	if c.Workspace.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "workspace.name",
			Message: "workspace name is required",
		})
	}

	// Check for duplicate project names
	names := make(map[string]bool)
	for _, p := range c.Projects {
		if p.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "projects[].name",
				Message: "project name is required",
			})
			continue
		}

		if names[p.Name] {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Project: p.Name,
				Field:   "name",
				Message: "duplicate project name",
			})
		}
		names[p.Name] = true

		// Check path
		if p.Path == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Project: p.Name,
				Field:   "path",
				Message: "project path is required",
			})
		}

		// Check tags policy
		if c.Policies.RequireTags && len(p.Tags) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Project: p.Name,
				Field:   "tags",
				Message: "at least one tag is required by policy",
			})
		}

		// Check allowed tags
		if len(c.Policies.AllowedTags) > 0 {
			allowed := make(map[string]bool)
			for _, t := range c.Policies.AllowedTags {
				allowed[t] = true
			}
			for _, t := range p.Tags {
				if !allowed[t] {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Project: p.Name,
						Field:   "tags",
						Message: "tag '" + t + "' is not in allowed tags list",
					})
				}
			}
		}

		// Check dependencies exist
		for _, dep := range p.DependsOn {
			if !names[dep] && dep != p.Name {
				// Will be checked after all projects are processed
			}
		}
	}

	// Second pass: check dependencies
	for _, p := range c.Projects {
		for _, dep := range p.DependsOn {
			if !names[dep] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Project: p.Name,
					Field:   "depends_on",
					Message: "dependency '" + dep + "' not found in workspace",
				})
			}
		}
	}

	// Check for circular dependencies if policy requires
	if c.Policies.NoCircularDeps {
		cycles := c.DetectCycles()
		for _, cycle := range cycles {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "depends_on",
				Message: "circular dependency detected: " + formatCycle(cycle),
			})
		}
	}

	// Warnings
	for _, p := range c.Projects {
		if len(p.Tags) == 0 && !c.Policies.RequireTags {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Project: p.Name,
				Message: "project has no tags, consider adding tags for filtering",
			})
		}
	}

	return result
}

// DetectCycles finds circular dependencies in the workspace.
func (c *Config) DetectCycles() [][]string {
	graph := c.BuildDependencyGraph()
	var cycles [][]string

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		currentPath := append(path, node)

		for _, dep := range graph.Projects[node] {
			if !visited[dep] {
				if dfs(dep, currentPath) {
					return true
				}
			} else if recStack[dep] {
				// Found cycle
				cycleStart := -1
				for i, n := range currentPath {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := append(currentPath[cycleStart:], dep)
					cycles = append(cycles, cycle)
				}
			}
		}

		recStack[node] = false
		return false
	}

	for _, p := range c.Projects {
		if !visited[p.Name] {
			dfs(p.Name, []string{})
		}
	}

	return cycles
}

// BuildDependencyGraph builds the project dependency graph.
func (c *Config) BuildDependencyGraph() *DependencyGraph {
	graph := &DependencyGraph{
		Projects: make(map[string][]string),
		Reverse:  make(map[string][]string),
	}

	for _, p := range c.Projects {
		graph.Projects[p.Name] = p.DependsOn
		for _, dep := range p.DependsOn {
			graph.Reverse[dep] = append(graph.Reverse[dep], p.Name)
		}
	}

	return graph
}

// GetProject returns a project by name.
func (c *Config) GetProject(name string) *Project {
	for i := range c.Projects {
		if c.Projects[i].Name == name {
			return &c.Projects[i]
		}
	}
	return nil
}

// GetProjectsByTag returns all projects with the given tag.
func (c *Config) GetProjectsByTag(tag string) []*Project {
	var result []*Project
	for i := range c.Projects {
		for _, t := range c.Projects[i].Tags {
			if t == tag {
				result = append(result, &c.Projects[i])
				break
			}
		}
	}
	return result
}

// GetEnabledProjects returns all enabled projects.
func (c *Config) GetEnabledProjects() []*Project {
	var result []*Project
	for i := range c.Projects {
		if c.Projects[i].IsEnabled() {
			result = append(result, &c.Projects[i])
		}
	}
	return result
}

func formatCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	result := cycle[0]
	for i := 1; i < len(cycle); i++ {
		result += " → " + cycle[i]
	}
	return result
}
