package models

import (
	"fmt"
	"strings"
)

// ══════════════════════════════════════════════════════════════════
//  Python generators: None, pydantic, sqlalchemy
// ══════════════════════════════════════════════════════════════════

func newPythonGenerator(orm ORMPlugin) (ModelCodeGenerator, error) {
	switch {
	case orm.IsNone():
		return &PythonNoneGenerator{}, nil
	case orm.Name == "pydantic":
		return &PythonPydanticGenerator{version: orm.Version}, nil
	case orm.Name == "sqlalchemy":
		return &PythonSQLAlchemyGenerator{version: orm.Version}, nil
	default:
		return nil, fmt.Errorf("unsupported Python ORM plugin: %s", orm.Name)
	}
}

// ──────────────────────────────────────────────────────────────────
//  Python None (pure dataclass)
// ──────────────────────────────────────────────────────────────────

// PythonNoneGenerator generates pure Python dataclasses.
type PythonNoneGenerator struct{}

func (g *PythonNoneGenerator) Language() string { return "python" }
func (g *PythonNoneGenerator) ORMName() string  { return "None" }

func (g *PythonNoneGenerator) GenerateBaseModel(opts GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from dataclasses import dataclass, field\n")
	b.WriteString("from datetime import datetime\n")
	b.WriteString("from typing import Any, Dict, Optional\n")
	b.WriteString("from uuid import UUID, uuid4\n\n\n")

	b.WriteString("@dataclass\n")
	b.WriteString("class BaseModel:\n")
	b.WriteString("    \"\"\"Base model for all buffalo-models generated models.\"\"\"\n\n")
	b.WriteString("    id: UUID = field(default_factory=uuid4)\n")
	b.WriteString("    created_at: datetime = field(default_factory=datetime.utcnow)\n")
	b.WriteString("    updated_at: datetime = field(default_factory=datetime.utcnow)\n")
	b.WriteString("    deleted_at: Optional[datetime] = None\n\n")

	b.WriteString("    def to_dict(self) -> Dict[str, Any]:\n")
	b.WriteString("        \"\"\"Convert model to dictionary.\"\"\"\n")
	b.WriteString("        result: Dict[str, Any] = {}\n")
	b.WriteString("        for k, v in self.__dict__.items():\n")
	b.WriteString("            if isinstance(v, UUID):\n")
	b.WriteString("                result[k] = str(v)\n")
	b.WriteString("            elif isinstance(v, datetime):\n")
	b.WriteString("                result[k] = v.isoformat()\n")
	b.WriteString("            elif isinstance(v, BaseModel):\n")
	b.WriteString("                result[k] = v.to_dict()\n")
	b.WriteString("            elif isinstance(v, list):\n")
	b.WriteString("                result[k] = [\n")
	b.WriteString("                    item.to_dict() if isinstance(item, BaseModel) else item\n")
	b.WriteString("                    for item in v\n")
	b.WriteString("                ]\n")
	b.WriteString("            else:\n")
	b.WriteString("                result[k] = v\n")
	b.WriteString("        return result\n\n")

	b.WriteString("    def __repr__(self) -> str:\n")
	b.WriteString("        cls = type(self).__name__\n")
	b.WriteString("        fields = \", \".join(f\"{k}={v!r}\" for k, v in self.__dict__.items())\n")
	b.WriteString("        return f\"{cls}({fields})\"\n")

	return GeneratedFile{Path: "base_model.py", Content: b.String()}, nil
}

func (g *PythonNoneGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from dataclasses import dataclass, field\n")
	b.WriteString("from typing import List, Optional\n\n")
	b.WriteString("from .base_model import BaseModel\n\n\n")

	className := model.EffectiveName()

	// Deprecation
	if model.Deprecated {
		b.WriteString(fmt.Sprintf("# DEPRECATED: %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}

	b.WriteString("@dataclass\n")
	parentClass := "BaseModel"
	if model.Extends != "" {
		parentClass = model.Extends
	}
	b.WriteString(fmt.Sprintf("class %s(%s):\n", className, parentClass))

	// Docstring
	if model.Description != "" {
		b.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n\n", model.Description))
	}

	// Table metadata as class vars
	if model.TableName != "" {
		b.WriteString(fmt.Sprintf("    __tablename__ = \"%s\"\n", model.TableName))
	}
	if model.Schema != "" {
		b.WriteString(fmt.Sprintf("    __schema__ = \"%s\"\n", model.Schema))
	}
	if model.TableName != "" || model.Schema != "" {
		b.WriteString("\n")
	}

	// Fields
	hasFields := false
	for _, f := range model.Fields {
		if f.Ignore {
			continue
		}
		hasFields = true
		line := g.fieldToDataclass(f)
		b.WriteString(line)
	}

	if !hasFields {
		b.WriteString("    pass\n")
	}

	fileName := toSnakeCase(model.MessageName) + ".py"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *PythonNoneGenerator) fieldToDataclass(f FieldDef) string {
	var b strings.Builder

	// Comment / deprecation
	if f.Deprecated {
		b.WriteString(fmt.Sprintf("    # DEPRECATED: %s\n", deprecatedComment(true, f.DeprecatedMessage)))
	}
	if f.Description != "" {
		b.WriteString(fmt.Sprintf("    # %s\n", f.Description))
	}

	// Visibility marker
	if f.Visibility == VisibilityInternal {
		b.WriteString("    # [internal]\n")
	} else if f.Visibility == VisibilityExternal {
		b.WriteString("    # [external]\n")
	} else if f.Visibility == VisibilityPrivate {
		b.WriteString("    # [private]\n")
	}

	// Behavior marker
	if f.Behavior == BehaviorReadOnly {
		b.WriteString("    # [readonly]\n")
	} else if f.Behavior == BehaviorWriteOnly {
		b.WriteString("    # [writeonly]\n")
	} else if f.Behavior == BehaviorComputed {
		b.WriteString("    # [computed]\n")
	} else if f.Behavior == BehaviorImmutable {
		b.WriteString("    # [immutable]\n")
	}

	typeHint := protoTypeToPython(f.ProtoType, f.Nullable)
	if f.Repeated {
		innerType := protoTypeToPython(f.ProtoType, false)
		typeHint = fmt.Sprintf("List[%s]", innerType)
	}

	defaultVal := pythonDefaultValue(f)
	if f.Repeated {
		b.WriteString(fmt.Sprintf("    %s: %s = field(default_factory=list)\n", f.Name, typeHint))
	} else if f.Sensitive {
		b.WriteString(fmt.Sprintf("    %s: %s = %s  # sensitive\n", f.Name, typeHint, defaultVal))
	} else {
		b.WriteString(fmt.Sprintf("    %s: %s = %s\n", f.Name, typeHint, defaultVal))
	}
	return b.String()
}

func (g *PythonNoneGenerator) GenerateInit(models []ModelDef, opts GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models"))
	b.WriteString("\nfrom .base_model import BaseModel\n")
	for _, m := range models {
		className := m.EffectiveName()
		fileName := toSnakeCase(m.MessageName)
		b.WriteString(fmt.Sprintf("from .%s import %s\n", fileName, className))
	}
	b.WriteString("\n__all__ = [\n")
	b.WriteString("    \"BaseModel\",\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("    \"%s\",\n", m.EffectiveName()))
	}
	b.WriteString("]\n")

	return GeneratedFile{Path: "__init__.py", Content: b.String()}, nil
}

// ──────────────────────────────────────────────────────────────────
//  Python Pydantic v1 / v2
// ──────────────────────────────────────────────────────────────────

// PythonPydanticGenerator uses pydantic BaseModel.
type PythonPydanticGenerator struct {
	version string
}

func (g *PythonPydanticGenerator) Language() string { return "python" }
func (g *PythonPydanticGenerator) ORMName() string  { return "pydantic" }

func (g *PythonPydanticGenerator) isV2() bool {
	return g.version == "" || strings.HasPrefix(g.version, "2")
}

func (g *PythonPydanticGenerator) GenerateBaseModel(opts GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (pydantic)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from datetime import datetime\n")
	b.WriteString("from typing import Any, Dict, Optional\n")
	b.WriteString("from uuid import UUID, uuid4\n\n")

	if g.isV2() {
		b.WriteString("from pydantic import BaseModel as PydanticBaseModel, ConfigDict, Field\n\n\n")
		b.WriteString("class BaseModel(PydanticBaseModel):\n")
		b.WriteString("    \"\"\"Base model for all buffalo-models generated models (pydantic v2).\"\"\"\n\n")
		b.WriteString("    model_config = ConfigDict(\n")
		b.WriteString("        from_attributes=True,\n")
		b.WriteString("        populate_by_name=True,\n")
		b.WriteString("        json_schema_extra={\"generator\": \"buffalo-models\"},\n")
		b.WriteString("    )\n\n")
	} else {
		b.WriteString("from pydantic import BaseModel as PydanticBaseModel, Field\n\n\n")
		b.WriteString("class BaseModel(PydanticBaseModel):\n")
		b.WriteString("    \"\"\"Base model for all buffalo-models generated models (pydantic v1).\"\"\"\n\n")
		b.WriteString("    class Config:\n")
		b.WriteString("        orm_mode = True\n")
		b.WriteString("        allow_population_by_field_name = True\n\n")
	}

	b.WriteString("    id: UUID = Field(default_factory=uuid4)\n")
	b.WriteString("    created_at: datetime = Field(default_factory=datetime.utcnow)\n")
	b.WriteString("    updated_at: datetime = Field(default_factory=datetime.utcnow)\n")
	b.WriteString("    deleted_at: Optional[datetime] = None\n")

	return GeneratedFile{Path: "base_model.py", Content: b.String()}, nil
}

func (g *PythonPydanticGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (pydantic)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from typing import List, Optional\n\n")

	if g.isV2() {
		b.WriteString("from pydantic import ConfigDict, Field\n\n")
	} else {
		b.WriteString("from pydantic import Field\n\n")
	}
	b.WriteString("from .base_model import BaseModel\n\n\n")

	className := model.EffectiveName()

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("# DEPRECATED: %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}

	parentClass := "BaseModel"
	if model.Extends != "" {
		parentClass = model.Extends
	}
	b.WriteString(fmt.Sprintf("class %s(%s):\n", className, parentClass))

	// Docstring
	desc := model.Description
	if model.TableName != "" {
		desc += fmt.Sprintf("\n\n    Table: %s", model.TableName)
		if model.Schema != "" {
			desc = model.Description + fmt.Sprintf("\n\n    Table: %s.%s", model.Schema, model.TableName)
		}
	}
	if desc != "" {
		b.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n\n", desc))
	}

	// Model config (v2) with table metadata
	if g.isV2() {
		b.WriteString("    model_config = ConfigDict(\n")
		b.WriteString("        json_schema_extra={\n")
		if model.TableName != "" {
			b.WriteString(fmt.Sprintf("            \"tablename\": \"%s\",\n", model.TableName))
		}
		if model.Schema != "" {
			b.WriteString(fmt.Sprintf("            \"schema\": \"%s\",\n", model.Schema))
		}
		b.WriteString("        },\n")
		b.WriteString("    )\n\n")
	}

	// Fields
	hasFields := false
	for _, f := range model.Fields {
		if f.Ignore {
			continue
		}
		hasFields = true
		line := g.fieldToPydantic(f)
		b.WriteString(line)
	}

	if !hasFields {
		b.WriteString("    pass\n")
	}

	fileName := toSnakeCase(model.MessageName) + ".py"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *PythonPydanticGenerator) fieldToPydantic(f FieldDef) string {
	var b strings.Builder

	// Comments for behavior / visibility
	if f.Deprecated {
		b.WriteString(fmt.Sprintf("    # DEPRECATED: %s\n", deprecatedComment(true, f.DeprecatedMessage)))
	}
	if f.Description != "" {
		b.WriteString(fmt.Sprintf("    # %s\n", f.Description))
	}
	if f.Visibility != VisibilityDefault {
		b.WriteString(fmt.Sprintf("    # [%s]\n", f.Visibility.String()))
	}
	if f.Behavior != BehaviorDefault {
		b.WriteString(fmt.Sprintf("    # [%s]\n", f.Behavior.String()))
	}

	typeHint := protoTypeToPython(f.ProtoType, f.Nullable)
	if f.Repeated {
		innerType := protoTypeToPython(f.ProtoType, false)
		typeHint = fmt.Sprintf("List[%s]", innerType)
	}

	// Build Field(...) arguments
	var fieldArgs []string

	// Default value
	if f.Repeated {
		fieldArgs = append(fieldArgs, "default_factory=list")
	} else if f.DefaultValue != "" {
		if f.ProtoType == "string" {
			fieldArgs = append(fieldArgs, fmt.Sprintf("default=\"%s\"", f.DefaultValue))
		} else if f.ProtoType == "bool" {
			fieldArgs = append(fieldArgs, fmt.Sprintf("default=%s", pythonBool(f.DefaultValue)))
		} else {
			fieldArgs = append(fieldArgs, fmt.Sprintf("default=%s", f.DefaultValue))
		}
	} else if f.Nullable {
		fieldArgs = append(fieldArgs, "default=None")
	} else {
		switch f.ProtoType {
		case "string":
			fieldArgs = append(fieldArgs, "default=\"\"")
		case "bool":
			fieldArgs = append(fieldArgs, "default=False")
		default:
			// numeric defaults
			if strings.Contains(f.ProtoType, "int") || strings.Contains(f.ProtoType, "fixed") {
				fieldArgs = append(fieldArgs, "default=0")
			} else if f.ProtoType == "float" || f.ProtoType == "double" {
				fieldArgs = append(fieldArgs, "default=0.0")
			}
		}
	}

	if f.MaxLength > 0 {
		fieldArgs = append(fieldArgs, fmt.Sprintf("max_length=%d", f.MaxLength))
	}
	if f.MinLength > 0 {
		fieldArgs = append(fieldArgs, fmt.Sprintf("min_length=%d", f.MinLength))
	}
	if f.Description != "" {
		fieldArgs = append(fieldArgs, fmt.Sprintf("description=\"%s\"", f.Description))
	}
	if f.Example != "" {
		if g.isV2() {
			fieldArgs = append(fieldArgs, fmt.Sprintf("examples=[\"%s\"]", f.Example))
		} else {
			fieldArgs = append(fieldArgs, fmt.Sprintf("example=\"%s\"", f.Example))
		}
	}
	if f.Alias != "" {
		fieldArgs = append(fieldArgs, fmt.Sprintf("alias=\"%s\"", f.Alias))
	}
	if f.JSONName != "" && f.JSONName != f.Name {
		fieldArgs = append(fieldArgs, fmt.Sprintf("serialization_alias=\"%s\"", f.JSONName))
	}

	// json_schema_extra for metadata
	var extras []string
	if f.Unique {
		extras = append(extras, "\"unique\": True")
	}
	if f.Sensitive {
		extras = append(extras, "\"sensitive\": True")
	}
	if f.Visibility != VisibilityDefault {
		extras = append(extras, fmt.Sprintf("\"visibility\": \"%s\"", f.Visibility.String()))
	}
	if f.Behavior != BehaviorDefault {
		extras = append(extras, fmt.Sprintf("\"behavior\": \"%s\"", f.Behavior.String()))
	}
	if len(extras) > 0 && g.isV2() {
		fieldArgs = append(fieldArgs, fmt.Sprintf("json_schema_extra={%s}", strings.Join(extras, ", ")))
	}

	b.WriteString(fmt.Sprintf("    %s: %s = Field(%s)\n", f.Name, typeHint, strings.Join(fieldArgs, ", ")))
	return b.String()
}

func (g *PythonPydanticGenerator) GenerateInit(models []ModelDef, opts GenerateOptions) (GeneratedFile, error) {
	// Reuse the None generator's init
	none := &PythonNoneGenerator{}
	return none.GenerateInit(models, opts)
}

// ──────────────────────────────────────────────────────────────────
//  Python SQLAlchemy
// ──────────────────────────────────────────────────────────────────

// PythonSQLAlchemyGenerator generates SQLAlchemy ORM models.
type PythonSQLAlchemyGenerator struct {
	version string
}

func (g *PythonSQLAlchemyGenerator) Language() string { return "python" }
func (g *PythonSQLAlchemyGenerator) ORMName() string  { return "sqlalchemy" }

func (g *PythonSQLAlchemyGenerator) isV2() bool {
	return g.version == "" || strings.HasPrefix(g.version, "2")
}

func (g *PythonSQLAlchemyGenerator) GenerateBaseModel(opts GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (sqlalchemy)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("import uuid\nfrom datetime import datetime\n\n")

	if g.isV2() {
		b.WriteString("from sqlalchemy import DateTime\n")
		b.WriteString("from sqlalchemy.dialects.postgresql import UUID\n")
		b.WriteString("from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column\n\n\n")
		b.WriteString("class BaseModel(DeclarativeBase):\n")
		b.WriteString("    \"\"\"Base model for all buffalo-models generated models (SQLAlchemy v2).\"\"\"\n\n")
		b.WriteString("    __abstract__ = True\n\n")
		b.WriteString("    id: Mapped[uuid.UUID] = mapped_column(\n")
		b.WriteString("        UUID(as_uuid=True), primary_key=True, default=uuid.uuid4\n")
		b.WriteString("    )\n")
		b.WriteString("    created_at: Mapped[datetime] = mapped_column(\n")
		b.WriteString("        DateTime, default=datetime.utcnow\n")
		b.WriteString("    )\n")
		b.WriteString("    updated_at: Mapped[datetime] = mapped_column(\n")
		b.WriteString("        DateTime, default=datetime.utcnow, onupdate=datetime.utcnow\n")
		b.WriteString("    )\n")
		b.WriteString("    deleted_at: Mapped[datetime | None] = mapped_column(\n")
		b.WriteString("        DateTime, nullable=True, default=None\n")
		b.WriteString("    )\n")
	} else {
		b.WriteString("from sqlalchemy import Column, DateTime\n")
		b.WriteString("from sqlalchemy.dialects.postgresql import UUID\n")
		b.WriteString("from sqlalchemy.ext.declarative import declarative_base\n\n")
		b.WriteString("_Base = declarative_base()\n\n\n")
		b.WriteString("class BaseModel(_Base):\n")
		b.WriteString("    \"\"\"Base model for all buffalo-models generated models (SQLAlchemy v1).\"\"\"\n\n")
		b.WriteString("    __abstract__ = True\n\n")
		b.WriteString("    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)\n")
		b.WriteString("    created_at = Column(DateTime, default=datetime.utcnow)\n")
		b.WriteString("    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)\n")
		b.WriteString("    deleted_at = Column(DateTime, nullable=True)\n")
	}

	return GeneratedFile{Path: "base_model.py", Content: b.String()}, nil
}

func (g *PythonSQLAlchemyGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (sqlalchemy)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")

	if g.isV2() {
		b.WriteString("from sqlalchemy import CheckConstraint, Index, String, Integer, Float, Boolean\n")
		b.WriteString("from sqlalchemy.orm import Mapped, mapped_column, relationship\n\n")
	} else {
		b.WriteString("from sqlalchemy import CheckConstraint, Column, Index, String, Integer, Float, Boolean\n")
		b.WriteString("from sqlalchemy.orm import relationship\n\n")
	}
	b.WriteString("from .base_model import BaseModel\n\n\n")

	className := model.EffectiveName()

	parentClass := "BaseModel"
	if model.Extends != "" {
		parentClass = model.Extends
	}
	b.WriteString(fmt.Sprintf("class %s(%s):\n", className, parentClass))

	if model.Description != "" {
		b.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n\n", model.Description))
	}

	// __tablename__
	tableName := model.TableName
	if tableName == "" {
		tableName = toSnakeCase(model.MessageName) + "s"
	}
	b.WriteString(fmt.Sprintf("    __tablename__ = \"%s\"\n", tableName))

	// __table_args__
	var tableArgs []string
	for _, idx := range model.Indexes {
		cols := strings.Join(idx.Columns, "\", \"")
		unique := ""
		if idx.Unique {
			unique = ", unique=True"
		}
		tableArgs = append(tableArgs,
			fmt.Sprintf("        Index(\"%s\", \"%s\"%s)", idx.Name, cols, unique))
	}
	for _, chk := range model.Checks {
		tableArgs = append(tableArgs,
			fmt.Sprintf("        CheckConstraint(\"%s\", name=\"%s\")", chk.Expression, chk.Name))
	}
	// Schema
	schemaArg := ""
	if model.Schema != "" {
		schemaArg = fmt.Sprintf("\"schema\": \"%s\"", model.Schema)
	}
	commentArg := ""
	if model.Description != "" {
		commentArg = fmt.Sprintf("\"comment\": \"%s\"", model.Description)
	}
	var dictParts []string
	if schemaArg != "" {
		dictParts = append(dictParts, schemaArg)
	}
	if commentArg != "" {
		dictParts = append(dictParts, commentArg)
	}

	if len(tableArgs) > 0 || len(dictParts) > 0 {
		b.WriteString("    __table_args__ = (\n")
		for _, ta := range tableArgs {
			b.WriteString(ta + ",\n")
		}
		if len(dictParts) > 0 {
			b.WriteString(fmt.Sprintf("        {%s},\n", strings.Join(dictParts, ", ")))
		}
		b.WriteString("    )\n")
	}
	b.WriteString("\n")

	// Fields
	for _, f := range model.Fields {
		if f.Ignore || f.PrimaryKey {
			continue // PK is in BaseModel
		}
		line := g.fieldToSQLAlchemy(f)
		b.WriteString(line)
	}

	fileName := toSnakeCase(model.MessageName) + ".py"
	return []GeneratedFile{{Path: fileName, Content: b.String()}}, nil
}

func (g *PythonSQLAlchemyGenerator) fieldToSQLAlchemy(f FieldDef) string {
	var b strings.Builder

	if f.Description != "" {
		b.WriteString(fmt.Sprintf("    # %s\n", f.Description))
	}
	if f.Visibility != VisibilityDefault {
		b.WriteString(fmt.Sprintf("    # [%s]\n", f.Visibility.String()))
	}
	if f.Behavior != BehaviorDefault {
		b.WriteString(fmt.Sprintf("    # [%s]\n", f.Behavior.String()))
	}

	if f.Relation != nil && (f.Relation.Type == RelationHasMany || f.Relation.Type == RelationHasOne) {
		// Relationship field
		b.WriteString(fmt.Sprintf("    %s = relationship(\"%s\"", f.Name, f.Relation.Model))
		if f.Relation.InverseOf != "" {
			b.WriteString(fmt.Sprintf(", back_populates=\"%s\"", f.Relation.InverseOf))
		}
		b.WriteString(")\n")
		return b.String()
	}

	// Determine SQLAlchemy column type
	saType := g.protoToSAType(f)
	var colArgs []string
	colArgs = append(colArgs, saType)
	if f.Unique {
		colArgs = append(colArgs, "unique=True")
	}
	if f.Nullable {
		colArgs = append(colArgs, "nullable=True")
	} else {
		colArgs = append(colArgs, "nullable=False")
	}
	if f.DefaultValue != "" {
		colArgs = append(colArgs, fmt.Sprintf("server_default=\"%s\"", f.DefaultValue))
	}
	if f.Index {
		colArgs = append(colArgs, "index=True")
	}
	if f.Comment != "" {
		colArgs = append(colArgs, fmt.Sprintf("comment=\"%s\"", f.Comment))
	}

	if g.isV2() {
		goType := protoTypeToPython(f.ProtoType, f.Nullable)
		b.WriteString(fmt.Sprintf("    %s: Mapped[%s] = mapped_column(%s)\n",
			f.Name, goType, strings.Join(colArgs, ", ")))
	} else {
		b.WriteString(fmt.Sprintf("    %s = Column(%s)\n", f.Name, strings.Join(colArgs, ", ")))
	}

	return b.String()
}

func (g *PythonSQLAlchemyGenerator) protoToSAType(f FieldDef) string {
	if f.DBType != "" {
		return fmt.Sprintf("\"%s\"", f.DBType)
	}
	switch f.ProtoType {
	case "string":
		if f.MaxLength > 0 {
			return fmt.Sprintf("String(%d)", f.MaxLength)
		}
		return "String"
	case "int32", "sint32", "sfixed32", "int64", "sint64", "sfixed64",
		"uint32", "fixed32", "uint64", "fixed64":
		return "Integer"
	case "float", "double":
		return "Float"
	case "bool":
		return "Boolean"
	default:
		return "String"
	}
}

func (g *PythonSQLAlchemyGenerator) GenerateInit(models []ModelDef, opts GenerateOptions) (GeneratedFile, error) {
	none := &PythonNoneGenerator{}
	return none.GenerateInit(models, opts)
}
