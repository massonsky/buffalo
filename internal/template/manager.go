package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/massonsky/buffalo/pkg/logger"
)

// Template represents a code generation template
type Template struct {
	Name     string            // Template name
	Language string            // Target language
	Path     string            // Template files path
	Patterns []string          // File patterns to match (*.tmpl)
	Vars     map[string]string // Template variables
	Funcs    template.FuncMap  // Custom template functions
}

// TemplateConfig holds template configuration
type TemplateConfig struct {
	Templates []Template `yaml:"templates" mapstructure:"templates"`
}

// Manager manages code generation templates
type Manager struct {
	templates map[string]*Template
	log       logger.Logger
}

// NewManager creates a new template manager
func NewManager(log logger.Logger) *Manager {
	return &Manager{
		templates: make(map[string]*Template),
		log:       log,
	}
}

// Register registers a new template
func (m *Manager) Register(tmpl *Template) error {
	if tmpl.Name == "" {
		return fmt.Errorf("template name is required")
	}

	if tmpl.Language == "" {
		return fmt.Errorf("template language is required")
	}

	if tmpl.Path == "" {
		return fmt.Errorf("template path is required")
	}

	// Convert to absolute path if relative
	absPath, err := filepath.Abs(tmpl.Path)
	if err != nil {
		return fmt.Errorf("failed to resolve template path: %w", err)
	}
	tmpl.Path = absPath

	// Check if template path exists
	if _, err := os.Stat(tmpl.Path); os.IsNotExist(err) {
		return fmt.Errorf("template path does not exist: %s", tmpl.Path)
	}

	// Initialize default patterns if not provided
	if len(tmpl.Patterns) == 0 {
		tmpl.Patterns = []string{"*.tmpl", "**/*.tmpl"}
	}

	// Initialize vars map if nil
	if tmpl.Vars == nil {
		tmpl.Vars = make(map[string]string)
	}

	// Initialize funcs map if nil
	if tmpl.Funcs == nil {
		tmpl.Funcs = GetDefaultFuncs()
	}

	m.templates[tmpl.Name] = tmpl
	m.log.Debug("Template registered",
		logger.String("name", tmpl.Name),
		logger.String("language", tmpl.Language),
		logger.String("path", tmpl.Path),
	)

	return nil
}

// Get retrieves a template by name
func (m *Manager) Get(name string) (*Template, error) {
	tmpl, ok := m.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

// List returns all registered templates
func (m *Manager) List() []*Template {
	templates := make([]*Template, 0, len(m.templates))
	for _, tmpl := range m.templates {
		templates = append(templates, tmpl)
	}
	return templates
}

// ListByLanguage returns templates for a specific language
func (m *Manager) ListByLanguage(lang string) []*Template {
	templates := make([]*Template, 0)
	for _, tmpl := range m.templates {
		if tmpl.Language == lang {
			templates = append(templates, tmpl)
		}
	}
	return templates
}

// Render renders a template with given data
func (m *Manager) Render(ctx context.Context, templateName string, data interface{}, outputPath string) error {
	tmpl, err := m.Get(templateName)
	if err != nil {
		return err
	}

	m.log.Info("Rendering template",
		logger.String("template", templateName),
		logger.String("output", outputPath),
	)

	// Find template files
	templateFiles, err := m.findTemplateFiles(tmpl)
	if err != nil {
		return fmt.Errorf("failed to find template files: %w", err)
	}

	if len(templateFiles) == 0 {
		return fmt.Errorf("no template files found in: %s", tmpl.Path)
	}

	m.log.Debug("Found template files",
		logger.Int("count", len(templateFiles)),
	)

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Process each template file
	for _, templateFile := range templateFiles {
		if err := m.renderTemplateFile(ctx, tmpl, templateFile, data, outputPath); err != nil {
			m.log.Error("Failed to render template file",
				logger.String("file", templateFile),
				logger.Any("error", err),
			)
			return err
		}
	}

	m.log.Info("Template rendered successfully",
		logger.Int("files", len(templateFiles)),
	)

	return nil
}

// findTemplateFiles finds all template files matching patterns
func (m *Manager) findTemplateFiles(tmpl *Template) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	// Walk directory recursively
	err := filepath.Walk(tmpl.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file matches any pattern
		for _, pattern := range tmpl.Patterns {
			// Remove **/ prefix from pattern for matching
			cleanPattern := pattern
			if len(pattern) > 3 && pattern[:3] == "**/" {
				cleanPattern = pattern[3:]
			}
			
			matched, _ := filepath.Match(cleanPattern, filepath.Base(path))
			if matched && !seen[path] {
				files = append(files, path)
				seen[path] = true
				break
			}
		}

		return nil
	})

	return files, err
}

// renderTemplateFile renders a single template file
func (m *Manager) renderTemplateFile(ctx context.Context, tmpl *Template, templateFile string, data interface{}, outputPath string) error {
	// Read template file
	content, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template
	t := template.New(filepath.Base(templateFile))
	t = t.Funcs(tmpl.Funcs)

	t, err = t.Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare template data - merge data map with vars
	templateData := make(map[string]interface{})
	
	// If data is a map, merge it directly
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			templateData[k] = v
		}
	} else {
		templateData["Data"] = data
	}
	
	// Add vars
	templateData["Vars"] = tmpl.Vars

	// Determine output file name (remove .tmpl extension)
	relPath, err := filepath.Rel(tmpl.Path, templateFile)
	if err != nil {
		relPath = filepath.Base(templateFile)
	}

	outputFile := filepath.Join(outputPath, relPath)
	if filepath.Ext(outputFile) == ".tmpl" {
		outputFile = outputFile[:len(outputFile)-5] // Remove .tmpl
	}

	// Create output file directory
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := t.Execute(f, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	m.log.Debug("Rendered template file",
		logger.String("input", templateFile),
		logger.String("output", outputFile),
	)

	return nil
}

// GetDefaultFuncs returns default template functions
func GetDefaultFuncs() template.FuncMap {
	return template.FuncMap{
		"toLower":   func(s string) string { return filepath.ToSlash(s) },
		"toUpper":   func(s string) string { return filepath.ToSlash(s) },
		"trimSpace": func(s string) string { return s },
		"join":      filepath.Join,
	}
}
