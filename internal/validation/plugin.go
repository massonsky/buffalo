package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massonsky/buffalo/internal/plugin"
)

// ValidatePlugin is a Buffalo plugin that parses proto field annotations
// ([(buffalo.validate.rules)...]) and generates Validate() methods for
// each target language.
type ValidatePlugin struct {
	config    plugin.Config
	languages []string
	strict    bool
}

// NewValidatePlugin creates the built-in buffalo-validate plugin.
func NewValidatePlugin() plugin.Plugin {
	return &ValidatePlugin{}
}

// DefaultValidateConfig returns the default plugin configuration.
func DefaultValidateConfig() plugin.Config {
	return plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		HookPoints: []plugin.HookPoint{
			plugin.HookPointPostParse,
			plugin.HookPointPostBuild,
		},
		Priority: 90,
		Options: map[string]interface{}{
			"strict":    false,
			"languages": []interface{}{"go", "python", "cpp", "rust"},
		},
	}
}

func (p *ValidatePlugin) Name() string            { return "buffalo-validate" }
func (p *ValidatePlugin) Version() string         { return "1.0.0" }
func (p *ValidatePlugin) Type() plugin.PluginType { return plugin.PluginTypeGenerator }

func (p *ValidatePlugin) Description() string {
	return "Generates Validate() methods from [(buffalo.validate.rules)...] proto annotations"
}

func (p *ValidatePlugin) Init(config plugin.Config) error {
	p.config = config

	if strict, ok := config.Options["strict"].(bool); ok {
		p.strict = strict
	}

	if langs, ok := config.Options["languages"].([]interface{}); ok {
		for _, l := range langs {
			if s, ok := l.(string); ok {
				p.languages = append(p.languages, s)
			}
		}
	}
	if len(p.languages) == 0 {
		p.languages = []string{"go", "python", "cpp", "rust"}
	}

	return nil
}

func (p *ValidatePlugin) Execute(ctx context.Context, input *plugin.Input) (*plugin.Output, error) {
	output := &plugin.Output{
		Success:  true,
		Messages: []string{},
		Warnings: []string{},
		Errors:   []string{},
		Metadata: make(map[string]interface{}),
	}

	// Phase 1: Parse validation annotations from proto files
	var allRules []MessageRules
	for _, protoPath := range input.ProtoFiles {
		content, err := os.ReadFile(protoPath)
		if err != nil {
			output.Warnings = append(output.Warnings,
				fmt.Sprintf("cannot read %s: %v", protoPath, err))
			continue
		}

		msgRules, err := ExtractValidationRules(string(content), protoPath)
		if err != nil {
			if p.strict {
				output.Errors = append(output.Errors, err.Error())
				output.Success = false
			} else {
				output.Warnings = append(output.Warnings, err.Error())
			}
			continue
		}
		allRules = append(allRules, msgRules...)
	}

	if len(allRules) == 0 {
		output.Messages = append(output.Messages,
			"No buffalo.validate annotations found — skipping validation codegen")
		return output, nil
	}

	output.Messages = append(output.Messages,
		fmt.Sprintf("Found validation rules for %d message(s)", len(allRules)))

	// Phase 2: Generate validation code for each language
	for _, lang := range p.languages {
		gen, err := NewCodeGenerator(lang)
		if err != nil {
			output.Warnings = append(output.Warnings, err.Error())
			continue
		}

		files, err := gen.Generate(allRules)
		if err != nil {
			output.Errors = append(output.Errors,
				fmt.Sprintf("codegen failed for %s: %v", lang, err))
			output.Success = false
			continue
		}

		for _, f := range files {
			outPath := filepath.Join(input.OutputDir, lang, f.Path)
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				output.Errors = append(output.Errors,
					fmt.Sprintf("mkdir failed: %v", err))
				output.Success = false
				continue
			}
			if err := os.WriteFile(outPath, []byte(f.Content), 0600); err != nil {
				output.Errors = append(output.Errors,
					fmt.Sprintf("write failed: %v", err))
				output.Success = false
				continue
			}
			output.GeneratedFiles = append(output.GeneratedFiles, outPath)
		}
	}

	if output.Success {
		output.Messages = append(output.Messages,
			fmt.Sprintf("Generated %d validation file(s)", len(output.GeneratedFiles)))
	}

	return output, nil
}

func (p *ValidatePlugin) Shutdown() error { return nil }

// ── ValidatorPlugin interface ─────────────────────────────────────

// ValidationRules returns the list of rule types this plugin checks.
func (p *ValidatePlugin) ValidationRules() []string {
	return []string{
		string(RuleRequired), string(RuleGte), string(RuleLte),
		string(RuleGt), string(RuleLt), string(RuleConst),
		string(RuleMinLen), string(RuleMaxLen), string(RulePattern),
		string(RuleEmail), string(RuleURI), string(RuleUUID),
		string(RuleNotEmpty), string(RuleMinItems), string(RuleMaxItems),
		string(RuleUnique), string(RuleIP), string(RuleHostname),
	}
}
