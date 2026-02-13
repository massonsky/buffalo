package models

import (
	"fmt"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  C++ generator: None
// ══════════════════════════════════════════════════════════════════

func newCppGenerator(orm ORMPlugin) (ModelCodeGenerator, error) {
	if orm.IsNone() {
		return &CppNoneGenerator{}, nil
	}
	return nil, fmt.Errorf("unsupported C++ ORM plugin: %s", orm.Name)
}

// ──────────────────────────────────────────────────────────────────
//  C++ None (structs with nlohmann/json)
// ──────────────────────────────────────────────────────────────────

// CppNoneGenerator generates plain C++ structs with optional JSON serialization.
type CppNoneGenerator struct{}

func (g *CppNoneGenerator) Language() string { return "cpp" }
func (g *CppNoneGenerator) ORMName() string  { return "None" }

func (g *CppNoneGenerator) GenerateBaseModel(_ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(cppHeader("buffalo-models"))
	b.WriteString("#pragma once\n\n")
	b.WriteString("#include <chrono>\n")
	b.WriteString("#include <cstdint>\n")
	b.WriteString("#include <optional>\n")
	b.WriteString("#include <string>\n\n")

	b.WriteString("namespace buffalo::models {\n\n")

	b.WriteString("/// Base model for all buffalo-models generated structs.\n")
	b.WriteString("struct BaseModel {\n")
	b.WriteString("    std::string id;\n")
	b.WriteString("    std::chrono::system_clock::time_point created_at;\n")
	b.WriteString("    std::chrono::system_clock::time_point updated_at;\n")
	b.WriteString("    std::optional<std::chrono::system_clock::time_point> deleted_at;\n")
	b.WriteString("};\n\n")

	b.WriteString("}  // namespace buffalo::models\n")

	return GeneratedFile{Path: "base_model.h", Content: b.String()}, nil
}

func (g *CppNoneGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	ns := cppNamespace(opts.Package)

	// ── Header (.h) ──
	var h strings.Builder
	h.WriteString(cppHeader("buffalo-models"))
	h.WriteString("#pragma once\n\n")

	// Collect includes
	includes := map[string]bool{
		"<string>":   true,
		"<cstdint>":  true,
		"<optional>": true,
	}
	for _, f := range model.Fields {
		if f.Repeated {
			includes["<vector>"] = true
		}
		if f.ProtoType == "bytes" {
			includes["<vector>"] = true
		}
	}
	for inc := range includes {
		h.WriteString(fmt.Sprintf("#include %s\n", inc))
	}
	h.WriteString("\n#include \"base_model.h\"\n\n")

	h.WriteString(fmt.Sprintf("namespace %s {\n\n", ns))

	className := model.EffectiveName()

	if model.Deprecated {
		h.WriteString(fmt.Sprintf("/// @deprecated %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}
	if model.Description != "" {
		h.WriteString(fmt.Sprintf("/// %s\n", model.Description))
	}

	h.WriteString(fmt.Sprintf("struct %s : buffalo::models::BaseModel {\n", className))

	// Group fields by access specifier — emit specifier only on change
	currentAccess := "public" // structs default to public
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		newAccess := g.cppAccess(f.Visibility)
		if newAccess != currentAccess {
			h.WriteString(fmt.Sprintf("  %s:\n", newAccess))
			currentAccess = newAccess
		}
		line := g.fieldToCpp(f)
		h.WriteString(line)
	}

	h.WriteString("};\n\n")
	h.WriteString(fmt.Sprintf("}  // namespace %s\n", ns))

	fileName := toSnakeCase(model.MessageName)
	files := []GeneratedFile{
		{Path: fileName + ".h", Content: h.String()},
	}

	return files, nil
}

func (g *CppNoneGenerator) fieldToCpp(f FieldDef) string {
	var b strings.Builder

	if f.Description != "" {
		b.WriteString(fmt.Sprintf("    /// %s\n", f.Description))
	}
	if f.Visibility != VisibilityDefault {
		b.WriteString(fmt.Sprintf("    /// [%s]\n", f.Visibility.String()))
	}
	if f.Behavior != BehaviorDefault {
		b.WriteString(fmt.Sprintf("    /// [%s]\n", f.Behavior.String()))
	}
	if f.Deprecated {
		b.WriteString(fmt.Sprintf("    [[deprecated(\"%s\")]]\n", deprecatedComment(true, f.DeprecatedMessage)))
	}

	cppType := protoTypeToCpp(f.ProtoType, f.Nullable)
	if f.Repeated {
		elem := protoTypeToCpp(f.ProtoType, false)
		cppType = fmt.Sprintf("std::vector<%s>", elem)
	}
	if f.CustomType != "" {
		cppType = f.CustomType
	}

	if f.DefaultValue != "" {
		b.WriteString(fmt.Sprintf("    %s %s = %s;\n", cppType, f.Name, cppDefaultLiteral(f)))
	} else {
		b.WriteString(fmt.Sprintf("    %s %s;\n", cppType, f.Name))
	}

	return b.String()
}

// cppAccess returns the C++ access specifier for given visibility.
func (g *CppNoneGenerator) cppAccess(v FieldVisibility) string {
	switch v {
	case VisibilityPrivate:
		return "private"
	case VisibilityProtected:
		return "protected"
	default:
		return "public"
	}
}

func (g *CppNoneGenerator) GenerateInit(_ []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	return GeneratedFile{}, nil // C++ doesn't need init files
}

// ══════════════════════════════════════════════════════════════════
//  Shared C++ helpers
// ══════════════════════════════════════════════════════════════════

func cppHeader(generator string) string {
	return fmt.Sprintf("// Code generated by %s. DO NOT EDIT.\n\n", generator)
}

func cppNamespace(pkg string) string {
	if pkg == "" {
		return "buffalo::models"
	}
	return strings.ReplaceAll(pkg, ".", "::")
}

func cppDefaultLiteral(f FieldDef) string {
	switch f.ProtoType {
	case "string":
		return fmt.Sprintf("\"%s\"", f.DefaultValue)
	case "bool":
		return f.DefaultValue
	case "float", "double":
		v := f.DefaultValue
		if !strings.Contains(v, ".") {
			v += ".0"
		}
		return v
	default:
		return f.DefaultValue
	}
}
