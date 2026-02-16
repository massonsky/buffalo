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
	b.WriteString("#include <map>\n")
	b.WriteString("#include <stdexcept>\n")
	b.WriteString("#include <optional>\n")
	b.WriteString("#include <string>\n")
	b.WriteString("#include <vector>\n\n")
	b.WriteString("#include <google/protobuf/util/json_util.h>\n")
	b.WriteString("#include <nlohmann/json.hpp>\n\n")

	b.WriteString("namespace buffalo::models {\n\n")

	b.WriteString("/// Base model for all buffalo-models generated structs.\n")
	b.WriteString("struct BaseModel {\n")
	b.WriteString("    std::string id;\n")
	b.WriteString("    std::chrono::system_clock::time_point created_at;\n")
	b.WriteString("    std::chrono::system_clock::time_point updated_at;\n")
	b.WriteString("    std::optional<std::chrono::system_clock::time_point> deleted_at;\n")
	b.WriteString("\n")
	b.WriteString("    virtual ~BaseModel() = default;\n")
	b.WriteString("\n")
	b.WriteString("    // Override in generated/user models for custom mapping rules.\n")
	b.WriteString("    virtual nlohmann::json to_json_obj() const {\n")
	b.WriteString("        return nlohmann::json::object();\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("    // Override in generated/user models for custom mapping rules.\n")
	b.WriteString("    virtual void from_json_obj(const nlohmann::json&) {}\n")
	b.WriteString("\n")
	b.WriteString("    template <typename ProtoT>\n")
	b.WriteString("    static nlohmann::json proto_to_json(const ProtoT& proto_msg) {\n")
	b.WriteString("        std::string out;\n")
	b.WriteString("        google::protobuf::util::JsonPrintOptions opts;\n")
	b.WriteString("        opts.always_print_primitive_fields = true;\n")
	b.WriteString("        opts.preserve_proto_field_names = true;\n")
	b.WriteString("        auto status = google::protobuf::util::MessageToJsonString(proto_msg, &out, opts);\n")
	b.WriteString("        if (!status.ok()) {\n")
	b.WriteString("            throw std::runtime_error(status.message().as_string());\n")
	b.WriteString("        }\n")
	b.WriteString("        return nlohmann::json::parse(out);\n")
	b.WriteString("    }\n\n")
	b.WriteString("    template <typename ProtoT>\n")
	b.WriteString("    static ProtoT json_to_proto(const nlohmann::json& json_obj) {\n")
	b.WriteString("        ProtoT msg;\n")
	b.WriteString("        auto status = google::protobuf::util::JsonStringToMessage(json_obj.dump(), &msg);\n")
	b.WriteString("        if (!status.ok()) {\n")
	b.WriteString("            throw std::runtime_error(status.message().as_string());\n")
	b.WriteString("        }\n")
	b.WriteString("        return msg;\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("    template <typename ProtoT>\n")
	b.WriteString("    ProtoT to_proto() const {\n")
	b.WriteString("        return json_to_proto<ProtoT>(to_json_obj());\n")
	b.WriteString("    }\n")
	b.WriteString("\n")
	b.WriteString("    template <typename ProtoT>\n")
	b.WriteString("    void from_proto(const ProtoT& proto_msg) {\n")
	b.WriteString("        from_json_obj(proto_to_json(proto_msg));\n")
	b.WriteString("    }\n")
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
	h.WriteString("  public:\n")
	h.WriteString("    /// Override-friendly conversion hook to JSON object.\n")
	h.WriteString("    nlohmann::json to_json_obj() const override {\n")
	h.WriteString("        nlohmann::json j = nlohmann::json::object();\n")
	h.WriteString("        // TODO: fill generated field mappings here if strict conversion is needed.\n")
	h.WriteString("        return j;\n")
	h.WriteString("    }\n\n")
	h.WriteString("    /// Override-friendly conversion hook from JSON object.\n")
	h.WriteString("    void from_json_obj(const nlohmann::json& j) override {\n")
	h.WriteString("        (void)j;\n")
	h.WriteString("        // TODO: parse generated field mappings here if strict conversion is needed.\n")
	h.WriteString("    }\n")

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

	cppType := fieldTypeCpp(f)
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

func (g *CppNoneGenerator) GenerateInit(models []ModelDef, opts GenerateOptions) (GeneratedFile, error) {
	projectName := opts.Package
	if projectName == "" {
		projectName = "buffalo_models"
	}

	var b strings.Builder
	b.WriteString("# Code generated by buffalo-models. DO NOT EDIT.\n")
	b.WriteString("cmake_minimum_required(VERSION 3.14)\n")
	b.WriteString(fmt.Sprintf("project(%s CXX)\n\n", projectName))
	b.WriteString("set(CMAKE_CXX_STANDARD 17)\n")
	b.WriteString("set(CMAKE_CXX_STANDARD_REQUIRED ON)\n\n")

	b.WriteString("# Dependencies\n")
	b.WriteString("find_package(Protobuf REQUIRED)\n")
	b.WriteString("find_package(nlohmann_json 3.11 REQUIRED)\n\n")

	b.WriteString("# Header-only library from generated models\n")
	b.WriteString(fmt.Sprintf("add_library(%s INTERFACE)\n", projectName))
	b.WriteString(fmt.Sprintf("target_include_directories(%s INTERFACE ${CMAKE_CURRENT_SOURCE_DIR})\n", projectName))
	b.WriteString(fmt.Sprintf("target_link_libraries(%s INTERFACE\n", projectName))
	b.WriteString("    protobuf::libprotobuf\n")
	b.WriteString("    nlohmann_json::nlohmann_json\n")
	b.WriteString(")\n\n")

	b.WriteString("# Generated model headers\n")
	b.WriteString("set(MODEL_HEADERS\n")
	b.WriteString("    base_model.h\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("    %s.h\n", toSnakeCase(m.MessageName)))
	}
	b.WriteString(")\n")

	return GeneratedFile{Path: "CMakeLists.txt", Content: b.String()}, nil
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
