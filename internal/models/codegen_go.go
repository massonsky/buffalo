package models

import (
	"fmt"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  Go generators: None, gorm, sqlx
// ══════════════════════════════════════════════════════════════════

func newGoGenerator(orm ORMPlugin) (ModelCodeGenerator, error) {
	switch {
	case orm.IsNone():
		return &GoNoneGenerator{}, nil
	case orm.Name == "gorm":
		return &GoGORMGenerator{}, nil
	case orm.Name == "sqlx":
		return &GoSQLXGenerator{}, nil
	default:
		return nil, fmt.Errorf("unsupported Go ORM plugin: %s", orm.Name)
	}
}

// ──────────────────────────────────────────────────────────────────
//  Go None (plain structs)
// ──────────────────────────────────────────────────────────────────

// GoNoneGenerator generates plain Go structs with json tags.
type GoNoneGenerator struct{}

func (g *GoNoneGenerator) Language() string { return "go" }
func (g *GoNoneGenerator) ORMName() string  { return "None" }

func (g *GoNoneGenerator) GenerateBaseModel(opts GenerateOptions) (GeneratedFile, error) {
	pkg := goPackageName(opts.Package)
	if pkg == "" {
		pkg = "models"
	}

	var b strings.Builder
	b.WriteString(header("buffalo-models"))
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	b.WriteString("import (\n")
	b.WriteString("\t\"time\"\n\n")
	b.WriteString("\t\"github.com/google/uuid\"\n")
	b.WriteString(")\n\n")

	b.WriteString("// BaseModel is the base for all buffalo-models generated models.\n")
	b.WriteString("type BaseModel struct {\n")
	b.WriteString("\tID        uuid.UUID  `json:\"id\"`\n")
	b.WriteString("\tCreatedAt time.Time  `json:\"created_at\"`\n")
	b.WriteString("\tUpdatedAt time.Time  `json:\"updated_at\"`\n")
	b.WriteString("\tDeletedAt *time.Time `json:\"deleted_at,omitempty\"`\n")
	b.WriteString("}\n")

	return GeneratedFile{Path: "base_model.go", Content: b.String()}, nil
}

func (g *GoNoneGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	pkg := goPackageName(opts.Package)
	if pkg == "" {
		pkg = "models"
	}

	var b strings.Builder
	b.WriteString(header("buffalo-models"))
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))

	className := model.EffectiveName()

	// Deprecation + description
	if model.Deprecated {
		b.WriteString(fmt.Sprintf("// Deprecated: %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}
	b.WriteString(fmt.Sprintf("// %s", className))
	if model.Description != "" {
		b.WriteString(fmt.Sprintf(" — %s", model.Description))
	}
	b.WriteString("\n")
	if model.TableName != "" {
		b.WriteString(fmt.Sprintf("// Table: %s\n", model.TableName))
	}

	parentStruct := "BaseModel"
	if model.Extends != "" {
		parentStruct = model.Extends
	}

	b.WriteString(fmt.Sprintf("type %s struct {\n", className))
	b.WriteString(fmt.Sprintf("\t%s\n", parentStruct))

	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		line := g.fieldToGoStruct(f)
		b.WriteString(line)
	}
	b.WriteString("}\n")

	// TableName method
	if model.TableName != "" {
		b.WriteString(fmt.Sprintf("\n// TableName returns the table name for %s.\n", className))
		b.WriteString(fmt.Sprintf("func (%s) TableName() string { return \"%s\" }\n", className, model.TableName))
	}

	fileName := toSnakeCase(model.MessageName) + ".go"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *GoNoneGenerator) fieldToGoStruct(f FieldDef) string {
	var b strings.Builder

	if f.Deprecated {
		b.WriteString(fmt.Sprintf("\t// Deprecated: %s\n", deprecatedComment(true, f.DeprecatedMessage)))
	}
	if f.Description != "" {
		b.WriteString(fmt.Sprintf("\t// %s\n", f.Description))
	}
	if f.Visibility != VisibilityDefault {
		b.WriteString(fmt.Sprintf("\t// [%s]\n", f.Visibility.String()))
	}
	if f.Behavior != BehaviorDefault {
		b.WriteString(fmt.Sprintf("\t// [%s]\n", f.Behavior.String()))
	}

	goName := toPascalCase(f.Name)
	goType := protoTypeToGo(f.ProtoType, f.Nullable)
	if f.Repeated {
		goType = "[]" + protoTypeToGo(f.ProtoType, false)
	}
	if f.CustomType != "" {
		goType = f.CustomType
	}

	// JSON tag
	jsonName := f.EffectiveJSONName()
	jsonTag := jsonName
	if f.OmitEmpty || f.Nullable {
		jsonTag += ",omitempty"
	}

	b.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", goName, goType, jsonTag))
	return b.String()
}

func (g *GoNoneGenerator) GenerateInit(_ []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	return GeneratedFile{}, nil // Go doesn't need init files
}

// ──────────────────────────────────────────────────────────────────
//  Go GORM
// ──────────────────────────────────────────────────────────────────

// GoGORMGenerator generates GORM-tagged Go structs.
type GoGORMGenerator struct{}

func (g *GoGORMGenerator) Language() string { return "go" }
func (g *GoGORMGenerator) ORMName() string  { return "gorm" }

func (g *GoGORMGenerator) GenerateBaseModel(opts GenerateOptions) (GeneratedFile, error) {
	pkg := goPackageName(opts.Package)
	if pkg == "" {
		pkg = "models"
	}

	var b strings.Builder
	b.WriteString(header("buffalo-models (gorm)"))
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	b.WriteString("import (\n")
	b.WriteString("\t\"time\"\n\n")
	b.WriteString("\t\"github.com/google/uuid\"\n")
	b.WriteString("\t\"gorm.io/gorm\"\n")
	b.WriteString(")\n\n")

	b.WriteString("// BaseModel is the GORM base for all buffalo-models generated models.\n")
	b.WriteString("type BaseModel struct {\n")
	b.WriteString("\tID        uuid.UUID      `gorm:\"type:uuid;primaryKey;default:gen_random_uuid()\" json:\"id\"`\n")
	b.WriteString("\tCreatedAt time.Time      `gorm:\"autoCreateTime\" json:\"created_at\"`\n")
	b.WriteString("\tUpdatedAt time.Time      `gorm:\"autoUpdateTime\" json:\"updated_at\"`\n")
	b.WriteString("\tDeletedAt gorm.DeletedAt `gorm:\"index\" json:\"deleted_at,omitempty\"`\n")
	b.WriteString("}\n")

	return GeneratedFile{Path: "base_model.go", Content: b.String()}, nil
}

func (g *GoGORMGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	pkg := goPackageName(opts.Package)
	if pkg == "" {
		pkg = "models"
	}

	var b strings.Builder
	b.WriteString(header("buffalo-models (gorm)"))
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))

	className := model.EffectiveName()

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("// Deprecated: %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}
	b.WriteString(fmt.Sprintf("// %s", className))
	if model.Description != "" {
		b.WriteString(fmt.Sprintf(" — %s", model.Description))
	}
	b.WriteString("\n")

	parentStruct := "BaseModel"
	if model.Extends != "" {
		parentStruct = model.Extends
	}

	b.WriteString(fmt.Sprintf("type %s struct {\n", className))
	b.WriteString(fmt.Sprintf("\t%s\n", parentStruct))

	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		line := g.fieldToGORM(f, model)
		b.WriteString(line)
	}
	b.WriteString("}\n")

	// TableName method
	tableName := model.TableName
	if tableName == "" {
		tableName = toSnakeCase(model.MessageName) + "s"
	}
	b.WriteString(fmt.Sprintf("\n// TableName returns the table name for %s.\n", className))
	b.WriteString(fmt.Sprintf("func (%s) TableName() string { return \"%s\" }\n", className, tableName))

	fileName := toSnakeCase(model.MessageName) + ".go"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *GoGORMGenerator) fieldToGORM(f FieldDef, model ModelDef) string {
	var b strings.Builder

	if f.Description != "" {
		b.WriteString(fmt.Sprintf("\t// %s\n", f.Description))
	}
	if f.Visibility != VisibilityDefault {
		b.WriteString(fmt.Sprintf("\t// [%s]\n", f.Visibility.String()))
	}
	if f.Behavior != BehaviorDefault {
		b.WriteString(fmt.Sprintf("\t// [%s]\n", f.Behavior.String()))
	}

	goName := toPascalCase(f.Name)
	goType := protoTypeToGo(f.ProtoType, f.Nullable)
	if f.Repeated {
		goType = "[]" + protoTypeToGo(f.ProtoType, false)
	}
	if f.CustomType != "" {
		goType = f.CustomType
	}

	// Relation override
	if f.Relation != nil {
		switch f.Relation.Type {
		case RelationHasMany:
			goType = "[]" + f.Relation.Model
		case RelationHasOne, RelationBelongsTo:
			goType = "*" + f.Relation.Model
		}
	}

	// Build GORM tag
	var gormParts []string
	if f.DBType != "" {
		gormParts = append(gormParts, fmt.Sprintf("type:%s", f.DBType))
	} else if f.MaxLength > 0 && f.ProtoType == "string" {
		gormParts = append(gormParts, fmt.Sprintf("size:%d", f.MaxLength))
	}
	if f.Unique {
		// Check if part of a named index
		gormParts = append(gormParts, "unique")
	}
	if !f.Nullable && f.ProtoType != "bool" {
		gormParts = append(gormParts, "not null")
	}
	if f.DefaultValue != "" {
		gormParts = append(gormParts, fmt.Sprintf("default:'%s'", f.DefaultValue))
	}
	if f.Index {
		gormParts = append(gormParts, "index")
	}
	if f.Comment != "" {
		gormParts = append(gormParts, fmt.Sprintf("comment:%s", f.Comment))
	}
	if f.Relation != nil && f.Relation.ForeignKey != "" {
		gormParts = append(gormParts, fmt.Sprintf("foreignKey:%s", toPascalCase(f.Relation.ForeignKey)))
	}

	// JSON tag
	jsonName := f.EffectiveJSONName()
	jsonTag := jsonName
	if f.OmitEmpty || f.Nullable {
		jsonTag += ",omitempty"
	}

	// Build combined tag
	tags := fmt.Sprintf("`json:\"%s\"", jsonTag)
	if len(gormParts) > 0 {
		tags += fmt.Sprintf(" gorm:\"%s\"", strings.Join(gormParts, ";"))
	}
	tags += "`"

	b.WriteString(fmt.Sprintf("\t%s %s %s\n", goName, goType, tags))
	return b.String()
}

func (g *GoGORMGenerator) GenerateInit(_ []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	return GeneratedFile{}, nil
}

// ──────────────────────────────────────────────────────────────────
//  Go sqlx
// ──────────────────────────────────────────────────────────────────

// GoSQLXGenerator generates Go structs with db tags for sqlx.
type GoSQLXGenerator struct{}

func (g *GoSQLXGenerator) Language() string { return "go" }
func (g *GoSQLXGenerator) ORMName() string  { return "sqlx" }

func (g *GoSQLXGenerator) GenerateBaseModel(opts GenerateOptions) (GeneratedFile, error) {
	pkg := goPackageName(opts.Package)
	if pkg == "" {
		pkg = "models"
	}

	var b strings.Builder
	b.WriteString(header("buffalo-models (sqlx)"))
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	b.WriteString("import (\n")
	b.WriteString("\t\"time\"\n\n")
	b.WriteString("\t\"github.com/google/uuid\"\n")
	b.WriteString(")\n\n")

	b.WriteString("// BaseModel is the sqlx base for all buffalo-models generated models.\n")
	b.WriteString("type BaseModel struct {\n")
	b.WriteString("\tID        uuid.UUID  `db:\"id\" json:\"id\"`\n")
	b.WriteString("\tCreatedAt time.Time  `db:\"created_at\" json:\"created_at\"`\n")
	b.WriteString("\tUpdatedAt time.Time  `db:\"updated_at\" json:\"updated_at\"`\n")
	b.WriteString("\tDeletedAt *time.Time `db:\"deleted_at\" json:\"deleted_at,omitempty\"`\n")
	b.WriteString("}\n")

	return GeneratedFile{Path: "base_model.go", Content: b.String()}, nil
}

func (g *GoSQLXGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	pkg := goPackageName(opts.Package)
	if pkg == "" {
		pkg = "models"
	}

	var b strings.Builder
	b.WriteString(header("buffalo-models (sqlx)"))
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))

	className := model.EffectiveName()

	b.WriteString(fmt.Sprintf("// %s", className))
	if model.Description != "" {
		b.WriteString(fmt.Sprintf(" — %s", model.Description))
	}
	b.WriteString("\n")

	parentStruct := "BaseModel"
	if model.Extends != "" {
		parentStruct = model.Extends
	}

	b.WriteString(fmt.Sprintf("type %s struct {\n", className))
	b.WriteString(fmt.Sprintf("\t%s\n", parentStruct))

	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue
		}
		goName := toPascalCase(f.Name)
		goType := protoTypeToGo(f.ProtoType, f.Nullable)
		if f.Repeated {
			goType = "[]" + protoTypeToGo(f.ProtoType, false)
		}

		dbName := f.Name
		if f.Alias != "" {
			dbName = f.Alias
		}
		jsonName := f.EffectiveJSONName()
		jsonTag := jsonName
		if f.OmitEmpty || f.Nullable {
			jsonTag += ",omitempty"
		}

		b.WriteString(fmt.Sprintf("\t%s %s `db:\"%s\" json:\"%s\"`\n", goName, goType, dbName, jsonTag))
	}
	b.WriteString("}\n")

	fileName := toSnakeCase(model.MessageName) + ".go"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *GoSQLXGenerator) GenerateInit(_ []ModelDef, _ GenerateOptions) (GeneratedFile, error) {
	return GeneratedFile{}, nil
}
