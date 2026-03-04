package models

import (
	"fmt"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  TypeScript generators: None, zod
// ══════════════════════════════════════════════════════════════════

func newTypescriptGenerator(orm ORMPlugin) (ModelCodeGenerator, error) {
	switch {
	case orm.IsNone():
		return &TypescriptNoneGenerator{}, nil
	case orm.Name == "zod":
		return &TypescriptZodGenerator{}, nil
	default:
		return nil, fmt.Errorf("unsupported TypeScript ORM plugin: %s", orm.Name)
	}
}

// ──────────────────────────────────────────────────────────────────
//  TypeScript None (plain interfaces)
// ──────────────────────────────────────────────────────────────────

// TypescriptNoneGenerator generates plain TypeScript interfaces.
type TypescriptNoneGenerator struct{}

func (g *TypescriptNoneGenerator) Language() string { return "typescript" }
func (g *TypescriptNoneGenerator) ORMName() string  { return "None" }

func (g *TypescriptNoneGenerator) GenerateBaseModel(_ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models"))
	b.WriteString("\n")
	b.WriteString("/** Base model for all buffalo-models generated models. */\n")
	b.WriteString("export interface BaseModel {\n")
	b.WriteString("  id: string;\n")
	b.WriteString("  createdAt: string;\n")
	b.WriteString("  updatedAt: string;\n")
	b.WriteString("  deletedAt?: string | null;\n")
	b.WriteString("}\n\n")

	b.WriteString("/** Creates a new BaseModel with default values. */\n")
	b.WriteString("export function createBaseModel(overrides?: Partial<BaseModel>): BaseModel {\n")
	b.WriteString("  return {\n")
	b.WriteString("    id: crypto.randomUUID(),\n")
	b.WriteString("    createdAt: new Date().toISOString(),\n")
	b.WriteString("    updatedAt: new Date().toISOString(),\n")
	b.WriteString("    deletedAt: null,\n")
	b.WriteString("    ...overrides,\n")
	b.WriteString("  };\n")
	b.WriteString("}\n\n")

	b.WriteString("/** Fields belonging to the base model (excluded from equality checks). */\n")
	b.WriteString("export const BASE_MODEL_FIELDS = new Set<string>(['id', 'createdAt', 'updatedAt', 'deletedAt']);\n\n")

	b.WriteString("/**\n")
	b.WriteString(" * Compares two model objects by their own fields only,\n")
	b.WriteString(" * ignoring base model fields (id, timestamps).\n")
	b.WriteString(" */\n")
	b.WriteString("export function modelsEqual<T extends BaseModel>(a: T, b: T): boolean {\n")
	b.WriteString("  const keys = Object.keys(a).filter((k) => !BASE_MODEL_FIELDS.has(k));\n")
	b.WriteString("  return keys.every(\n")
	b.WriteString("    (k) => JSON.stringify((a as Record<string, unknown>)[k]) === JSON.stringify((b as Record<string, unknown>)[k]),\n")
	b.WriteString("  );\n")
	b.WriteString("}\n")

	return GeneratedFile{Path: "base_model.ts", Content: b.String()}, nil
}

func (g *TypescriptNoneGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models"))
	b.WriteString("\n")
	b.WriteString("import type { BaseModel } from './base_model';\n")
	b.WriteString("import { createBaseModel } from './base_model';\n")

	className := model.EffectiveName()

	// Custom type imports
	if customImports := tsCustomTypeImports(model, className); customImports != "" {
		b.WriteString(customImports)
	}
	b.WriteString("\n")

	// Nested enums
	for _, e := range model.Enums {
		b.WriteString(generateTypescriptEnum(e))
		b.WriteString("\n")
	}

	// Oneof type aliases
	for _, o := range model.Oneofs {
		b.WriteString(generateTypescriptOneofType(o))
	}

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("/** @deprecated %s */\n", deprecatedComment(true, model.DeprecatedMessage)))
	}

	// Interface
	if model.Description != "" {
		b.WriteString(fmt.Sprintf("/** %s */\n", model.Description))
	}

	parentType := "BaseModel"
	if model.Extends != "" {
		parentType = model.Extends
	}
	b.WriteString(fmt.Sprintf("export interface %s extends %s {\n", className, parentType))

	hasFields := false
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		hasFields = true
		line := g.fieldToInterface(f)
		b.WriteString(line)
	}
	if !hasFields {
		b.WriteString("  // no additional fields\n")
	}
	b.WriteString("}\n\n")

	// Factory function
	b.WriteString(fmt.Sprintf("/** Creates a new %s with default values. */\n", className))
	b.WriteString(fmt.Sprintf("export function create%s(overrides?: Partial<%s>): %s {\n", className, className, className))
	b.WriteString("  return {\n")
	b.WriteString("    ...createBaseModel(),\n")
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		def := tsDefaultValue(f)
		b.WriteString(fmt.Sprintf("    %s: %s,\n", tsCamelCase(f.Name), def))
	}
	b.WriteString("    ...overrides,\n")
	b.WriteString("  };\n")
	b.WriteString("}\n")

	fileName := toSnakeCase(model.MessageName) + ".ts"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *TypescriptNoneGenerator) fieldToInterface(f FieldDef) string {
	var b strings.Builder

	if f.Description != "" {
		b.WriteString(fmt.Sprintf("  /** %s */\n", f.Description))
	}
	if f.Deprecated {
		b.WriteString(fmt.Sprintf("  /** @deprecated %s */\n", deprecatedComment(true, f.DeprecatedMessage)))
	}

	tsType := fieldTypeTypescript(f)
	fieldName := tsCamelCase(f.Name)

	if f.Nullable {
		b.WriteString(fmt.Sprintf("  %s?: %s | null;\n", fieldName, tsType))
	} else {
		b.WriteString(fmt.Sprintf("  %s: %s;\n", fieldName, tsType))
	}
	return b.String()
}

func (g *TypescriptNoneGenerator) GenerateInit(models []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models"))
	b.WriteString("\n")
	b.WriteString("export * from './base_model';\n")
	for _, m := range models {
		fileName := toSnakeCase(m.MessageName)
		b.WriteString(fmt.Sprintf("export * from './%s';\n", fileName))
	}

	return GeneratedFile{Path: "index.ts", Content: b.String()}, nil
}

func (g *TypescriptNoneGenerator) GenerateEnum(enum EnumDef, _ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models"))
	b.WriteString("\n")
	b.WriteString(generateTypescriptEnum(enum))

	fileName := toSnakeCase(enum.Name) + ".ts"
	return GeneratedFile{Path: fileName, Content: b.String()}, nil
}

// ──────────────────────────────────────────────────────────────────
//  TypeScript Zod (runtime validation)
// ──────────────────────────────────────────────────────────────────

// TypescriptZodGenerator generates TypeScript models with Zod schemas.
type TypescriptZodGenerator struct{}

func (g *TypescriptZodGenerator) Language() string { return "typescript" }
func (g *TypescriptZodGenerator) ORMName() string  { return "zod" }

func (g *TypescriptZodGenerator) GenerateBaseModel(_ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models (zod)"))
	b.WriteString("\nimport { z } from 'zod';\n\n")

	b.WriteString("/** Zod schema for the base model. */\n")
	b.WriteString("export const BaseModelSchema = z.object({\n")
	b.WriteString("  id: z.string().uuid(),\n")
	b.WriteString("  createdAt: z.string().datetime(),\n")
	b.WriteString("  updatedAt: z.string().datetime(),\n")
	b.WriteString("  deletedAt: z.string().datetime().nullable().optional(),\n")
	b.WriteString("});\n\n")

	b.WriteString("/** Base model type inferred from Zod schema. */\n")
	b.WriteString("export type BaseModel = z.infer<typeof BaseModelSchema>;\n\n")

	b.WriteString("/** Creates a new BaseModel with default values. */\n")
	b.WriteString("export function createBaseModel(overrides?: Partial<BaseModel>): BaseModel {\n")
	b.WriteString("  return {\n")
	b.WriteString("    id: crypto.randomUUID(),\n")
	b.WriteString("    createdAt: new Date().toISOString(),\n")
	b.WriteString("    updatedAt: new Date().toISOString(),\n")
	b.WriteString("    deletedAt: null,\n")
	b.WriteString("    ...overrides,\n")
	b.WriteString("  };\n")
	b.WriteString("}\n\n")

	b.WriteString("/** Fields belonging to the base model (excluded from equality checks). */\n")
	b.WriteString("export const BASE_MODEL_FIELDS = new Set<string>(['id', 'createdAt', 'updatedAt', 'deletedAt']);\n")

	return GeneratedFile{Path: "base_model.ts", Content: b.String()}, nil
}

func (g *TypescriptZodGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models (zod)"))
	b.WriteString("\nimport { z } from 'zod';\n")
	b.WriteString("import { BaseModelSchema, createBaseModel } from './base_model';\n")
	b.WriteString("import type { BaseModel } from './base_model';\n")

	className := model.EffectiveName()

	// Custom type imports
	if customImports := tsCustomTypeImports(model, className); customImports != "" {
		b.WriteString(customImports)
	}
	b.WriteString("\n")

	// Nested enums
	for _, e := range model.Enums {
		b.WriteString(generateTypescriptEnum(e))
		b.WriteString("\n")
	}

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("/** @deprecated %s */\n", deprecatedComment(true, model.DeprecatedMessage)))
	}
	if model.Description != "" {
		b.WriteString(fmt.Sprintf("/** %s */\n", model.Description))
	}

	// Zod schema
	b.WriteString(fmt.Sprintf("export const %sSchema = BaseModelSchema.extend({\n", className))
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		line := g.fieldToZod(f)
		b.WriteString(line)
	}
	b.WriteString("});\n\n")

	// Type alias
	b.WriteString(fmt.Sprintf("/** %s type inferred from Zod schema. */\n", className))
	b.WriteString(fmt.Sprintf("export type %s = z.infer<typeof %sSchema>;\n\n", className, className))

	// Factory function
	b.WriteString(fmt.Sprintf("/** Creates a new %s with default values. */\n", className))
	b.WriteString(fmt.Sprintf("export function create%s(overrides?: Partial<%s>): %s {\n", className, className, className))
	b.WriteString("  return {\n")
	b.WriteString("    ...createBaseModel(),\n")
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		def := tsDefaultValue(f)
		b.WriteString(fmt.Sprintf("    %s: %s,\n", tsCamelCase(f.Name), def))
	}
	b.WriteString("    ...overrides,\n")
	b.WriteString("  };\n")
	b.WriteString("}\n")

	fileName := toSnakeCase(model.MessageName) + ".ts"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *TypescriptZodGenerator) fieldToZod(f FieldDef) string {
	var b strings.Builder

	if f.Description != "" {
		b.WriteString(fmt.Sprintf("  /** %s */\n", f.Description))
	}

	fieldName := tsCamelCase(f.Name)
	zodType := fieldZodType(f)

	b.WriteString(fmt.Sprintf("  %s: %s,\n", fieldName, zodType))
	return b.String()
}

func (g *TypescriptZodGenerator) GenerateInit(models []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models (zod)"))
	b.WriteString("\n")
	b.WriteString("export * from './base_model';\n")
	for _, m := range models {
		fileName := toSnakeCase(m.MessageName)
		b.WriteString(fmt.Sprintf("export * from './%s';\n", fileName))
	}

	return GeneratedFile{Path: "index.ts", Content: b.String()}, nil
}

func (g *TypescriptZodGenerator) GenerateEnum(enum EnumDef, _ GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(tsHeader("buffalo-models (zod)"))
	b.WriteString("\n")
	b.WriteString(generateTypescriptEnum(enum))

	fileName := toSnakeCase(enum.Name) + ".ts"
	return GeneratedFile{Path: fileName, Content: b.String()}, nil
}

// ══════════════════════════════════════════════════════════════════
//  Shared TypeScript helpers
// ══════════════════════════════════════════════════════════════════

func tsHeader(generator string) string {
	return fmt.Sprintf("// Code generated by %s. DO NOT EDIT.\n", generator)
}

// tsCamelCase converts snake_case to camelCase for TypeScript field names.
func tsCamelCase(s string) string {
	return toCamelCase(s)
}

// protoTypeToTypescript maps proto types to TypeScript types.
func protoTypeToTypescript(protoType string, nullable bool) string {
	if wkt, ok := wellKnownTypeTypescript(protoType); ok {
		if nullable {
			return wkt + " | null"
		}
		return wkt
	}
	base := ""
	switch protoType {
	case "string":
		base = "string"
	case "int32", "sint32", "sfixed32", "uint32", "fixed32",
		"float", "double":
		base = "number"
	case "int64", "sint64", "sfixed64", "uint64", "fixed64":
		base = "string" // 64-bit integers are strings in JSON
	case "bool":
		base = "boolean"
	case "bytes":
		base = "Uint8Array"
	default:
		base = toPascalCase(stripPackagePrefix(protoType))
	}
	if nullable {
		return base + " | null"
	}
	return base
}

// fieldTypeTypescript returns the full TypeScript type for a field (handles map, repeated).
func fieldTypeTypescript(f FieldDef) string {
	if f.IsMap {
		kt := protoTypeToTypescript(f.MapKeyType, false)
		vt := protoTypeToTypescript(f.MapValueType, false)
		return fmt.Sprintf("Record<%s, %s>", kt, vt)
	}
	if f.Repeated {
		inner := protoTypeToTypescript(f.ProtoType, false)
		return inner + "[]"
	}
	return protoTypeToTypescript(f.ProtoType, false)
}

// fieldZodType returns a Zod schema expression for a field.
func fieldZodType(f FieldDef) string {
	base := protoZodType(f.ProtoType)

	if f.IsMap {
		kt := protoZodType(f.MapKeyType)
		vt := protoZodType(f.MapValueType)
		base = fmt.Sprintf("z.record(%s, %s)", kt, vt)
	} else if f.Repeated {
		base = fmt.Sprintf("z.array(%s)", base)
	}

	if f.Nullable {
		base += ".nullable().optional()"
	}

	if f.MaxLength > 0 && !f.IsMap && !f.Repeated {
		if f.ProtoType == "string" {
			base += fmt.Sprintf(".max(%d)", f.MaxLength)
		}
	}
	if f.MinLength > 0 && !f.IsMap && !f.Repeated {
		if f.ProtoType == "string" {
			base += fmt.Sprintf(".min(%d)", f.MinLength)
		}
	}

	return base
}

// protoZodType returns a Zod schema expression for a proto type.
func protoZodType(protoType string) string {
	switch protoType {
	case "string":
		return "z.string()"
	case "int32", "sint32", "sfixed32", "uint32", "fixed32",
		"float", "double":
		return "z.number()"
	case "int64", "sint64", "sfixed64", "uint64", "fixed64":
		return "z.string()" // 64-bit as string
	case "bool":
		return "z.boolean()"
	case "bytes":
		return "z.instanceof(Uint8Array)"
	default:
		// Well-known types
		if wkt, ok := wellKnownZodType(protoType); ok {
			return wkt
		}
		// Custom message — use z.any() as a safe fallback
		return "z.any()"
	}
}

// wellKnownTypeTypescript maps well-known protobuf types to TypeScript types.
func wellKnownTypeTypescript(pt string) (string, bool) {
	m := map[string]string{
		"google.protobuf.Timestamp":   "string",
		"google.protobuf.Duration":    "string",
		"google.protobuf.Empty":       "Record<string, never>",
		"google.protobuf.Any":         "unknown",
		"google.protobuf.Struct":      "Record<string, unknown>",
		"google.protobuf.Value":       "unknown",
		"google.protobuf.BoolValue":   "boolean",
		"google.protobuf.Int32Value":  "number",
		"google.protobuf.Int64Value":  "string",
		"google.protobuf.UInt32Value": "number",
		"google.protobuf.UInt64Value": "string",
		"google.protobuf.FloatValue":  "number",
		"google.protobuf.DoubleValue": "number",
		"google.protobuf.StringValue": "string",
		"google.protobuf.BytesValue":  "Uint8Array",
	}
	v, ok := m[pt]
	return v, ok
}

// wellKnownZodType maps well-known protobuf types to Zod schema expressions.
func wellKnownZodType(pt string) (string, bool) {
	m := map[string]string{
		"google.protobuf.Timestamp":   "z.string().datetime()",
		"google.protobuf.Duration":    "z.string()",
		"google.protobuf.Empty":       "z.object({})",
		"google.protobuf.Any":         "z.unknown()",
		"google.protobuf.Struct":      "z.record(z.string(), z.unknown())",
		"google.protobuf.Value":       "z.unknown()",
		"google.protobuf.BoolValue":   "z.boolean()",
		"google.protobuf.Int32Value":  "z.number()",
		"google.protobuf.Int64Value":  "z.string()",
		"google.protobuf.UInt32Value": "z.number()",
		"google.protobuf.UInt64Value": "z.string()",
		"google.protobuf.FloatValue":  "z.number()",
		"google.protobuf.DoubleValue": "z.number()",
		"google.protobuf.StringValue": "z.string()",
		"google.protobuf.BytesValue":  "z.instanceof(Uint8Array)",
	}
	v, ok := m[pt]
	return v, ok
}

// tsDefaultValue returns a TypeScript default value for a field.
func tsDefaultValue(f FieldDef) string {
	if f.DefaultValue != "" {
		switch f.ProtoType {
		case "string":
			return fmt.Sprintf("'%s'", f.DefaultValue)
		case "bool":
			return f.DefaultValue
		default:
			return f.DefaultValue
		}
	}
	if f.IsMap {
		return "{}"
	}
	if f.Repeated {
		return "[]"
	}
	if f.Nullable {
		return "null"
	}
	switch f.ProtoType {
	case "string":
		return "''"
	case "int32", "sint32", "sfixed32", "uint32", "fixed32",
		"float", "double":
		return "0"
	case "int64", "sint64", "sfixed64", "uint64", "fixed64":
		return "'0'"
	case "bool":
		return "false"
	case "bytes":
		return "new Uint8Array()"
	default:
		if f.IsEnum {
			return "0"
		}
		return "null as any"
	}
}

// generateTypescriptEnum produces a TypeScript const enum from an EnumDef.
func generateTypescriptEnum(e EnumDef) string {
	var b strings.Builder

	if e.Comment != "" {
		b.WriteString(fmt.Sprintf("/** %s */\n", e.Comment))
	}
	b.WriteString(fmt.Sprintf("export enum %s {\n", e.Name))
	for _, v := range e.Values {
		if v.Comment != "" {
			b.WriteString(fmt.Sprintf("  /** %s */\n", v.Comment))
		}
		b.WriteString(fmt.Sprintf("  %s = %d,\n", v.Name, v.Number))
	}
	b.WriteString("}\n")
	return b.String()
}

// generateTypescriptOneofType produces a TypeScript union type for a oneof group.
func generateTypescriptOneofType(o OneofDef) string {
	var types []string
	for _, f := range o.Fields {
		types = append(types, protoTypeToTypescript(f.ProtoType, false))
	}
	return fmt.Sprintf("// oneof %s\nexport type %sType = %s;\n\n",
		o.Name, toPascalCase(o.Name), strings.Join(types, " | "))
}

// tsCustomTypeImports generates import statements for custom message types
// referenced in the model's fields.
func tsCustomTypeImports(model ModelDef, ownClassName string) string {
	nestedEnumNames := map[string]bool{}
	for _, e := range model.Enums {
		nestedEnumNames[e.Name] = true
	}

	seen := map[string]bool{}
	var lines []string

	collect := func(protoType string) {
		if !isCustomProtoType(protoType) {
			return
		}
		className := toPascalCase(stripPackagePrefix(protoType))
		if className == ownClassName || seen[className] {
			return
		}
		if nestedEnumNames[className] {
			return
		}
		seen[className] = true
		module := toSnakeCase(stripPackagePrefix(protoType))
		lines = append(lines, fmt.Sprintf("import type { %s } from './%s';", className, module))
	}

	for _, f := range model.Fields {
		if f.Ignore {
			continue
		}
		if f.IsMap {
			collect(f.MapKeyType)
			collect(f.MapValueType)
		} else {
			collect(f.ProtoType)
		}
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
