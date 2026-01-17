package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/template"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	templateCmd = &cobra.Command{
		Use:   "template",
		Short: "Manage code generation templates",
		Long: `Manage custom code generation templates for Buffalo.

Templates allow you to customize how generated code looks and add
additional processing steps to your build pipeline.

Templates are defined in buffalo.yaml:
  templates:
    - name: "custom-go"
      language: "go"
      path: "./templates/go"
      patterns:
        - "**/*.tmpl"
      vars:
        packagePrefix: "github.com/myorg"`,
		Example: `  # List all available templates
  buffalo template list

  # List templates for specific language
  buffalo template list --lang go

  # Generate code using custom template
  buffalo template generate --template custom-go

  # Validate template configuration
  buffalo template validate --template custom-go`,
	}

	templateListCmd = &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long:  "List all registered code generation templates",
		RunE:  runTemplateList,
	}

	templateGenerateCmd = &cobra.Command{
		Use:   "generate",
		Short: "Generate code using template",
		Long:  "Generate code using a specific template",
		RunE:  runTemplateGenerate,
	}

	templateValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate template configuration",
		Long:  "Validate a template configuration and check if template files exist",
		RunE:  runTemplateValidate,
	}

	templateLang   string
	templateName   string
	templateOutput string
	templateData   string
)

func init() {
	rootCmd.AddCommand(templateCmd)

	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateGenerateCmd)
	templateCmd.AddCommand(templateValidateCmd)

	// List command flags
	templateListCmd.Flags().StringVarP(&templateLang, "lang", "l", "", "filter by language")

	// Generate command flags
	templateGenerateCmd.Flags().StringVarP(&templateName, "template", "t", "", "template name to use (required)")
	templateGenerateCmd.Flags().StringVarP(&templateOutput, "output", "o", "", "output directory (required)")
	templateGenerateCmd.Flags().StringVar(&templateData, "data", "", "JSON data to pass to template")
	templateGenerateCmd.MarkFlagRequired("template")
	templateGenerateCmd.MarkFlagRequired("output")

	// Validate command flags
	templateValidateCmd.Flags().StringVarP(&templateName, "template", "t", "", "template name to validate (required)")
	templateValidateCmd.MarkFlagRequired("template")
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Error("Failed to load configuration", logger.Any("error", err))
		return err
	}

	// Create template manager
	mgr := template.NewManager(*log)

	// Register templates from config
	if cfg.Templates != nil {
		for _, tmplCfg := range cfg.Templates {
			tmpl := &template.Template{
				Name:     tmplCfg.Name,
				Language: tmplCfg.Language,
				Path:     tmplCfg.Path,
				Patterns: tmplCfg.Patterns,
				Vars:     tmplCfg.Vars,
			}

			if err := mgr.Register(tmpl); err != nil {
				log.Warn("Failed to register template",
					logger.String("name", tmplCfg.Name),
					logger.Any("error", err),
				)
				continue
			}
		}
	}

	// Get templates
	var templates []*template.Template
	if templateLang != "" {
		templates = mgr.ListByLanguage(templateLang)
	} else {
		templates = mgr.List()
	}

	if len(templates) == 0 {
		if templateLang != "" {
			log.Info(fmt.Sprintf("No templates found for language: %s", templateLang))
		} else {
			log.Info("No templates registered")
			log.Info("\nTo add templates, define them in buffalo.yaml:")
			log.Info(`
templates:
  - name: "custom-go"
    language: "go"
    path: "./templates/go"
    patterns:
      - "**/*.tmpl"
    vars:
      packagePrefix: "github.com/myorg"`)
		}
		return nil
	}

	// Print templates
	log.Info("📋 Registered Templates:\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tLANGUAGE\tPATH\tPATTERNS")
	fmt.Fprintln(w, "----\t--------\t----\t--------")

	for _, tmpl := range templates {
		patterns := ""
		if len(tmpl.Patterns) > 0 {
			patterns = tmpl.Patterns[0]
			if len(tmpl.Patterns) > 1 {
				patterns += fmt.Sprintf(" (+%d more)", len(tmpl.Patterns)-1)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			tmpl.Name,
			tmpl.Language,
			tmpl.Path,
			patterns,
		)
	}

	w.Flush()

	log.Info(fmt.Sprintf("\nTotal: %d template(s)", len(templates)))

	return nil
}

func runTemplateGenerate(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔨 Generating code from template",
		logger.String("template", templateName),
		logger.String("output", templateOutput),
	)

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Error("Failed to load configuration", logger.Any("error", err))
		return err
	}

	// Create template manager
	mgr := template.NewManager(*log)

	// Register templates from config
	if cfg.Templates != nil {
		for _, tmplCfg := range cfg.Templates {
			tmpl := &template.Template{
				Name:     tmplCfg.Name,
				Language: tmplCfg.Language,
				Path:     tmplCfg.Path,
				Patterns: tmplCfg.Patterns,
				Vars:     tmplCfg.Vars,
			}

			if err := mgr.Register(tmpl); err != nil {
				log.Warn("Failed to register template",
					logger.String("name", tmplCfg.Name),
					logger.Any("error", err),
				)
				continue
			}
		}
	}

	// Get template
	tmpl, err := mgr.Get(templateName)
	if err != nil {
		log.Error("Template not found", logger.String("name", templateName))
		return fmt.Errorf("template not found: %s", templateName)
	}

	log.Debug("Found template", logger.String("language", tmpl.Language))

	// Parse template data if provided
	var data interface{}
	if templateData != "" {
		// TODO: Parse JSON data
		log.Debug("Using template data", logger.String("data", templateData))
	}

	// Render template
	absOutput, err := filepath.Abs(templateOutput)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	ctx := cmd.Context()
	if err := mgr.Render(ctx, templateName, data, absOutput); err != nil {
		log.Error("Failed to render template", logger.Any("error", err))
		return err
	}

	log.Info("✅ Template generated successfully", logger.String("output", absOutput))

	return nil
}

func runTemplateValidate(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔍 Validating template", logger.String("name", templateName))

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Error("Failed to load configuration", logger.Any("error", err))
		return err
	}

	// Find template in config
	var tmplCfg *config.TemplateConfig

	if cfg.Templates != nil {
		for i := range cfg.Templates {
			if cfg.Templates[i].Name == templateName {
				tmplCfg = &cfg.Templates[i]
				break
			}
		}
	}

	if tmplCfg == nil {
		log.Error("Template not found in configuration", logger.String("name", templateName))
		return fmt.Errorf("template not found: %s", templateName)
	}

	// Validate template
	issues := []string{}

	// Check name
	if tmplCfg.Name == "" {
		issues = append(issues, "❌ Template name is empty")
	} else {
		log.Info("✅ Name:", logger.String("value", tmplCfg.Name))
	}

	// Check language
	if tmplCfg.Language == "" {
		issues = append(issues, "❌ Template language is empty")
	} else {
		log.Info("✅ Language:", logger.String("value", tmplCfg.Language))
	}

	// Check path
	if tmplCfg.Path == "" {
		issues = append(issues, "❌ Template path is empty")
	} else {
		absPath, _ := filepath.Abs(tmplCfg.Path)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("❌ Template path does not exist: %s", absPath))
		} else {
			log.Info("✅ Path:", logger.String("value", absPath))

			// Count template files
			fileCount := 0
			filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					for _, pattern := range tmplCfg.Patterns {
						if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
							fileCount++
							break
						}
					}
				}
				return nil
			})

			log.Info("✅ Template files found:", logger.Int("count", fileCount))
		}
	}

	// Check patterns
	if len(tmplCfg.Patterns) == 0 {
		log.Info("⚠️  No patterns defined, will use default: *.tmpl, **/*.tmpl")
	} else {
		log.Info("✅ Patterns:", logger.Any("value", tmplCfg.Patterns))
	}

	// Check vars
	if len(tmplCfg.Vars) > 0 {
		log.Info("✅ Variables:", logger.Int("count", len(tmplCfg.Vars)))
		for k, v := range tmplCfg.Vars {
			log.Info(fmt.Sprintf("  - %s: %s", k, v))
		}
	}

	// Print results
	if len(issues) > 0 {
		log.Error("\n❌ Validation failed:")
		for _, issue := range issues {
			log.Error("  " + issue)
		}
		return fmt.Errorf("template validation failed with %d issue(s)", len(issues))
	}

	log.Info("\n✅ Template validation passed!")

	return nil
}
