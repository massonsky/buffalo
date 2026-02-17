package models

import (
	"fmt"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  Rust generators: None, diesel
// ══════════════════════════════════════════════════════════════════

func newRustGenerator(orm ORMPlugin) (ModelCodeGenerator, error) {
	switch {
	case orm.IsNone():
		return &RustNoneGenerator{}, nil
	case orm.Name == "diesel":
		return &RustDieselGenerator{version: orm.Version}, nil
	default:
		return nil, fmt.Errorf("unsupported Rust ORM plugin: %s", orm.Name)
	}
}

// ──────────────────────────────────────────────────────────────────
//  Rust None (plain structs + serde)
// ──────────────────────────────────────────────────────────────────

// RustNoneGenerator generates plain Rust structs with serde derives.
type RustNoneGenerator struct{}

func (g *RustNoneGenerator) Language() string { return "rust" }
func (g *RustNoneGenerator) ORMName() string  { return "None" }

func (g *RustNoneGenerator) GenerateBaseModel(_ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models"))
	b.WriteString("use chrono::{DateTime, Utc};\n")
	b.WriteString("use serde::{Deserialize, Serialize};\n")
	b.WriteString("use serde_json::Value;\n")
	b.WriteString("use uuid::Uuid;\n\n")
	b.WriteString("/// Trait for protobuf JSON bridge conversion.\n")
	b.WriteString("///\n")
	b.WriteString("/// This trait intentionally works with JSON Value as an interoperability\n")
	b.WriteString("/// layer between protobuf messages and Rust models.\n")
	b.WriteString("pub trait ProtoConvertible: Sized {\n")
	b.WriteString("    fn from_proto_json(proto_json: Value) -> Result<Self, serde_json::Error>;\n")
	b.WriteString("    fn to_proto_json(&self) -> Result<Value, serde_json::Error>;\n")
	b.WriteString("}\n\n")

	b.WriteString("/// Base model for all buffalo-models generated structs.\n")
	b.WriteString("#[derive(Debug, Clone, Serialize, Deserialize)]\n")
	b.WriteString("pub struct BaseModel {\n")
	b.WriteString("    pub id: Uuid,\n")
	b.WriteString("    pub created_at: DateTime<Utc>,\n")
	b.WriteString("    pub updated_at: DateTime<Utc>,\n")
	b.WriteString("    pub deleted_at: Option<DateTime<Utc>>,\n")
	b.WriteString("}\n")
	b.WriteString("\n")
	b.WriteString("impl ProtoConvertible for BaseModel {\n")
	b.WriteString("    fn from_proto_json(proto_json: Value) -> Result<Self, serde_json::Error> {\n")
	b.WriteString("        serde_json::from_value(proto_json)\n")
	b.WriteString("    }\n\n")
	b.WriteString("    fn to_proto_json(&self) -> Result<Value, serde_json::Error> {\n")
	b.WriteString("        serde_json::to_value(self)\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return GeneratedFile{Path: "base_model.rs", Content: b.String()}, nil
}

func (g *RustNoneGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models"))
	b.WriteString("use serde_json::Value;\n")
	b.WriteString("use serde::{Deserialize, Serialize};\n")
	b.WriteString("use super::base_model::{BaseModel, ProtoConvertible};\n\n")

	className := model.EffectiveName()

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("#[deprecated(note = \"%s\")]\n", deprecatedComment(true, model.DeprecatedMessage)))
	}
	if model.Description != "" {
		b.WriteString(fmt.Sprintf("/// %s\n", model.Description))
	}

	var derives []string
	derives = append(derives, "Debug", "Clone", "Serialize", "Deserialize")

	b.WriteString(fmt.Sprintf("#[derive(%s)]\n", strings.Join(derives, ", ")))
	if model.TableName != "" {
		b.WriteString(fmt.Sprintf("// table_name: %s\n", model.TableName))
	}
	b.WriteString(fmt.Sprintf("pub struct %s {\n", className))
	b.WriteString("    pub base: BaseModel,\n")

	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		line := g.fieldToRust(f)
		b.WriteString(line)
	}
	b.WriteString("}\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("impl ProtoConvertible for %s {\n", className))
	b.WriteString("    fn from_proto_json(proto_json: Value) -> Result<Self, serde_json::Error> {\n")
	b.WriteString("        serde_json::from_value(proto_json)\n")
	b.WriteString("    }\n\n")
	b.WriteString("    fn to_proto_json(&self) -> Result<Value, serde_json::Error> {\n")
	b.WriteString("        serde_json::to_value(self)\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	fileName := toSnakeCase(model.MessageName) + ".rs"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *RustNoneGenerator) fieldToRust(f FieldDef) string {
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

	rustType := fieldTypeRust(f)
	if f.CustomType != "" {
		rustType = f.CustomType
	}

	jsonName := f.EffectiveJSONName()
	if jsonName != f.Name {
		b.WriteString(fmt.Sprintf("    #[serde(rename = \"%s\")]\n", jsonName))
	}
	if f.Nullable || f.OmitEmpty {
		b.WriteString("    #[serde(skip_serializing_if = \"Option::is_none\")]\n")
	}

	vis := "pub"
	if f.Visibility == VisibilityPrivate {
		vis = ""
	} else if f.Visibility == VisibilityProtected {
		vis = "pub(crate)"
	} else if f.Visibility == VisibilityInternal {
		vis = "pub(super)"
	}

	if vis == "" {
		b.WriteString(fmt.Sprintf("    %s: %s,\n", f.Name, rustType))
	} else {
		b.WriteString(fmt.Sprintf("    %s %s: %s,\n", vis, f.Name, rustType))
	}
	return b.String()
}

func (g *RustNoneGenerator) GenerateInit(models []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models"))
	b.WriteString("pub mod base_model;\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("pub mod %s;\n", toSnakeCase(m.MessageName)))
	}
	return GeneratedFile{Path: "mod.rs", Content: b.String()}, nil
}

func (g *RustNoneGenerator) GenerateEnum(enum EnumDef, opts GenerateOptions) (GeneratedFile, error) {
	return generateRustEnumFile(enum)
}

// ──────────────────────────────────────────────────────────────────
//  Rust Diesel
// ──────────────────────────────────────────────────────────────────

// RustDieselGenerator generates Diesel-annotated Rust structs.
type RustDieselGenerator struct {
	version string
}

func (g *RustDieselGenerator) Language() string { return "rust" }
func (g *RustDieselGenerator) ORMName() string  { return "diesel" }

func (g *RustDieselGenerator) GenerateBaseModel(_ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models (diesel)"))
	b.WriteString("use chrono::{DateTime, Utc};\n")
	b.WriteString("use diesel::prelude::*;\n")
	b.WriteString("use serde::{Deserialize, Serialize};\n")
	b.WriteString("use serde_json::Value;\n")
	b.WriteString("use uuid::Uuid;\n\n")
	b.WriteString("/// Trait for protobuf JSON bridge conversion.\n")
	b.WriteString("pub trait ProtoConvertible: Sized {\n")
	b.WriteString("    fn from_proto_json(proto_json: Value) -> Result<Self, serde_json::Error>;\n")
	b.WriteString("    fn to_proto_json(&self) -> Result<Value, serde_json::Error>;\n")
	b.WriteString("}\n\n")

	b.WriteString("/// Base model fields for all diesel models.\n")
	b.WriteString("#[derive(Debug, Clone, Serialize, Deserialize, Queryable, Identifiable)]\n")
	b.WriteString("pub struct BaseModel {\n")
	b.WriteString("    pub id: Uuid,\n")
	b.WriteString("    pub created_at: DateTime<Utc>,\n")
	b.WriteString("    pub updated_at: DateTime<Utc>,\n")
	b.WriteString("    pub deleted_at: Option<DateTime<Utc>>,\n")
	b.WriteString("}\n")
	b.WriteString("\n")
	b.WriteString("impl ProtoConvertible for BaseModel {\n")
	b.WriteString("    fn from_proto_json(proto_json: Value) -> Result<Self, serde_json::Error> {\n")
	b.WriteString("        serde_json::from_value(proto_json)\n")
	b.WriteString("    }\n\n")
	b.WriteString("    fn to_proto_json(&self) -> Result<Value, serde_json::Error> {\n")
	b.WriteString("        serde_json::to_value(self)\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return GeneratedFile{Path: "base_model.rs", Content: b.String()}, nil
}

func (g *RustDieselGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models (diesel)"))
	b.WriteString("use chrono::{DateTime, Utc};\n")
	b.WriteString("use diesel::prelude::*;\n")
	b.WriteString("use serde::{Deserialize, Serialize};\n")
	b.WriteString("use serde_json::Value;\n")
	b.WriteString("use uuid::Uuid;\n\n")
	b.WriteString("use super::base_model::ProtoConvertible;\n\n")

	className := model.EffectiveName()

	if model.Description != "" {
		b.WriteString(fmt.Sprintf("/// %s\n", model.Description))
	}

	tableName := model.TableName
	if tableName == "" {
		tableName = toSnakeCase(model.MessageName) + "s"
	}

	// Queryable model
	var derives []string
	derives = append(derives, "Debug", "Clone", "Serialize", "Deserialize",
		"Queryable", "Identifiable", "Selectable")

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("#[deprecated(note = \"%s\")]\n", deprecatedComment(true, model.DeprecatedMessage)))
	}
	b.WriteString(fmt.Sprintf("#[derive(%s)]\n", strings.Join(derives, ", ")))
	b.WriteString(fmt.Sprintf("#[diesel(table_name = %s)]\n", tableName))
	b.WriteString(fmt.Sprintf("pub struct %s {\n", className))
	b.WriteString("    pub id: Uuid,\n")
	b.WriteString("    pub created_at: DateTime<Utc>,\n")
	b.WriteString("    pub updated_at: DateTime<Utc>,\n")
	b.WriteString("    pub deleted_at: Option<DateTime<Utc>>,\n")

	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		line := g.fieldToRust(f)
		b.WriteString(line)
	}
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("impl ProtoConvertible for %s {\n", className))
	b.WriteString("    fn from_proto_json(proto_json: Value) -> Result<Self, serde_json::Error> {\n")
	b.WriteString("        serde_json::from_value(proto_json)\n")
	b.WriteString("    }\n\n")
	b.WriteString("    fn to_proto_json(&self) -> Result<Value, serde_json::Error> {\n")
	b.WriteString("        serde_json::to_value(self)\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")

	// Insertable (New) struct
	newName := "New" + className
	b.WriteString(fmt.Sprintf("/// Insertable form of %s.\n", className))
	b.WriteString(fmt.Sprintf("#[derive(Debug, Clone, Insertable, Serialize, Deserialize)]\n"))
	b.WriteString(fmt.Sprintf("#[diesel(table_name = %s)]\n", tableName))
	b.WriteString(fmt.Sprintf("pub struct %s {\n", newName))
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey || f.Behavior == BehaviorReadOnly || f.Behavior == BehaviorComputed {
			continue
		}
		rustType := protoTypeToRust(f.ProtoType, f.Nullable)
		if f.Repeated {
			continue // skip repeated in insertable
		}
		if f.IsMap {
			continue // skip maps in insertable
		}
		b.WriteString(fmt.Sprintf("    pub %s: %s,\n", f.Name, rustType))
	}
	b.WriteString("}\n")

	fileName := toSnakeCase(model.MessageName) + ".rs"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *RustDieselGenerator) fieldToRust(f FieldDef) string {
	var b strings.Builder

	if f.Description != "" {
		b.WriteString(fmt.Sprintf("    /// %s\n", f.Description))
	}

	rustType := fieldTypeRust(f)

	columnName := f.Name
	if f.Alias != "" {
		columnName = f.Alias
	}
	if columnName != f.Name {
		b.WriteString(fmt.Sprintf("    #[diesel(column_name = %s)]\n", columnName))
	}

	b.WriteString(fmt.Sprintf("    pub %s: %s,\n", f.Name, rustType))
	return b.String()
}

func (g *RustDieselGenerator) GenerateInit(models []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models (diesel)"))
	b.WriteString("pub mod base_model;\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("pub mod %s;\n", toSnakeCase(m.MessageName)))
	}
	return GeneratedFile{Path: "mod.rs", Content: b.String()}, nil
}

func (g *RustDieselGenerator) GenerateEnum(enum EnumDef, opts GenerateOptions) (GeneratedFile, error) {
	return generateRustEnumFile(enum)
}

// generateRustEnumFile creates a standalone Rust file with an enum.
func generateRustEnumFile(enum EnumDef) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(rustHeader("buffalo-models"))
	b.WriteString("use serde::{Deserialize, Serialize};\n\n")
	b.WriteString(generateRustEnum(enum))

	fileName := toSnakeCase(enum.Name) + ".rs"
	return GeneratedFile{Path: fileName, Content: b.String()}, nil
}

// ══════════════════════════════════════════════════════════════════
//  Shared Rust helpers
// ══════════════════════════════════════════════════════════════════

func rustHeader(generator string) string {
	return fmt.Sprintf("// Code generated by %s. DO NOT EDIT.\n\n", generator)
}
