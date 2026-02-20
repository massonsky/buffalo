package models

import (
	"fmt"
	"strconv"
	"strings"
)

func pythonStringLiteral(s string) string {
	return strconv.Quote(s)
}

// protoPb2Module converts a .proto file path to a dotted Python pb2 module path.
func protoPb2Module(protoPath string) string {
	p := strings.ReplaceAll(protoPath, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, ".proto")
	parts := strings.Split(p, "/")
	return strings.Join(parts, ".") + "_pb2"
}

// pb2ImportPrefix returns the pb2 module prefix to use.
// If opts.Pb2ImportPrefix is set explicitly, it is used as-is.
// Otherwise returns empty string (proto file path used as-is).
func pb2ImportPrefix(opts GenerateOptions) string {
	return strings.TrimSuffix(strings.TrimSpace(opts.Pb2ImportPrefix), ".")
}

// pythonSysPathSetup generates Python code that adds the parent of the models
// output directory (relative to cwd) to sys.path so that pb2 imports resolve.
//
// Given outputDir="./generated/models", the parent is "generated".
// The generated code does: sys.path.insert(0, os.path.join(os.getcwd(), "generated"))
func pythonSysPathSetup(outputDir, prefix string) string {
	// Normalise outputDir and take its parent directory
	norm := strings.ReplaceAll(outputDir, "\\", "/")
	norm = strings.TrimPrefix(norm, "./")
	norm = strings.TrimSuffix(norm, "/")
	parent := ""
	if idx := strings.LastIndex(norm, "/"); idx >= 0 {
		parent = norm[:idx]
	}
	// parent is e.g. "generated" — the folder containing both models/ and python/

	var b strings.Builder
	b.WriteString("\nimport os as _os\n")
	b.WriteString("import sys as _sys\n\n")
	b.WriteString("# Auto-configure sys.path so that pb2 imports resolve correctly.\n")
	b.WriteString("# Uses working directory as base (not __file__).\n")
	if parent != "" {
		b.WriteString(fmt.Sprintf("_pb2_root = _os.path.join(_os.getcwd(), %q)\n", parent))
	} else {
		b.WriteString("_pb2_root = _os.getcwd()\n")
	}
	b.WriteString("if _pb2_root not in _sys.path:\n")
	b.WriteString("    _sys.path.insert(0, _pb2_root)\n")
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  Python generators: None, pydantic, sqlalchemy
// ══════════════════════════════════════════════════════════════════

// pythonExtraImports scans model fields and returns import lines
// for types that need additional Python imports (datetime, timedelta, UUID, etc.)
func pythonExtraImports(model ModelDef) string {
	needDatetime := false
	needTimedelta := false
	for _, f := range model.Fields {
		ft := fieldTypePython(f)
		if strings.Contains(ft, "datetime") {
			needDatetime = true
		}
		if strings.Contains(ft, "timedelta") {
			needTimedelta = true
		}
	}
	var parts []string
	if needDatetime {
		parts = append(parts, "datetime")
	}
	if needTimedelta {
		parts = append(parts, "timedelta")
	}
	if len(parts) > 0 {
		return fmt.Sprintf("from datetime import %s\n", strings.Join(parts, ", "))
	}
	return ""
}

// pythonPrimitiveTypes are proto types that map to Python builtins (no import needed).
var pythonPrimitiveTypes = map[string]bool{
	"string": true, "bool": true, "bytes": true,
	"int32": true, "int64": true, "uint32": true, "uint64": true,
	"sint32": true, "sint64": true, "fixed32": true, "fixed64": true,
	"sfixed32": true, "sfixed64": true,
	"float": true, "double": true,
}

// isCustomProtoType returns true if the proto type is a custom message
// (not a primitive and not a well-known type).
func isCustomProtoType(protoType string) bool {
	if pythonPrimitiveTypes[protoType] {
		return false
	}
	if _, ok := wellKnownTypePython(protoType); ok {
		return false
	}
	return protoType != ""
}

// pythonCustomTypeImports generates from-import lines for custom message types
// referenced in the model's fields (cross-package or same-package forward refs).
// Nested enums (defined inline in the model) are skipped — they have no standalone file.
func pythonCustomTypeImports(model ModelDef, ownClassName string) string {
	// Build a set of nested enum names so we don't import them from non-existent files.
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
		// Skip nested enums — they are rendered inline, not in separate files.
		if nestedEnumNames[className] {
			return
		}
		seen[className] = true
		module := toSnakeCase(stripPackagePrefix(protoType))
		lines = append(lines, fmt.Sprintf("from .%s import %s", module, className))
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

func newPythonGenerator(orm ORMPlugin) (ModelCodeGenerator, error) {
	// Python always uses Pydantic v2 — None and sqlalchemy are not supported
	// as separate generators.
	version := orm.Version
	if version == "" {
		version = "2.0"
	}
	return &PythonPydanticGenerator{version: version}, nil
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
	b.WriteString("from typing import Any, ClassVar, Dict, Optional\n")
	b.WriteString("from uuid import UUID, uuid4\n\n\n")

	b.WriteString("@dataclass(eq=False)\n")
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
	b.WriteString("        return f\"{cls}({fields})\"\n\n")

	// Operator overloads: compare only model-specific fields, ignoring base class fields.
	b.WriteString("    _base_model_fields: ClassVar[frozenset] = frozenset(\n")
	b.WriteString("        {\"id\", \"created_at\", \"updated_at\", \"deleted_at\"}\n")
	b.WriteString("    )\n\n")
	b.WriteString("    def __eq__(self, other: object) -> bool:\n")
	b.WriteString("        \"\"\"Compare models by own fields only, ignoring base class fields.\"\"\"\n")
	b.WriteString("        if not isinstance(other, self.__class__):\n")
	b.WriteString("            return NotImplemented\n")
	b.WriteString("        for k in self.__dict__:\n")
	b.WriteString("            if k in self._base_model_fields:\n")
	b.WriteString("                continue\n")
	b.WriteString("            if self.__dict__[k] != other.__dict__.get(k):\n")
	b.WriteString("                return False\n")
	b.WriteString("        return True\n\n")
	b.WriteString("    def __ne__(self, other: object) -> bool:\n")
	b.WriteString("        result = self.__eq__(other)\n")
	b.WriteString("        if result is NotImplemented:\n")
	b.WriteString("            return result\n")
	b.WriteString("        return not result\n\n")
	b.WriteString("    def __hash__(self) -> int:\n")
	b.WriteString("        \"\"\"Hash by own fields only, ignoring base class fields.\"\"\"\n")
	b.WriteString("        values: list = []\n")
	b.WriteString("        for k in sorted(self.__dict__):\n")
	b.WriteString("            if k in self._base_model_fields:\n")
	b.WriteString("                continue\n")
	b.WriteString("            v = self.__dict__[k]\n")
	b.WriteString("            try:\n")
	b.WriteString("                hash(v)\n")
	b.WriteString("                values.append((k, v))\n")
	b.WriteString("            except TypeError:\n")
	b.WriteString("                values.append((k, repr(v)))\n")
	b.WriteString("        return hash(tuple(values))\n")

	return GeneratedFile{Path: "base_model.py", Content: b.String()}, nil
}

func (g *PythonNoneGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from dataclasses import dataclass, field\n")
	// Dynamic imports for well-known types (datetime, timedelta, etc.)
	if extra := pythonExtraImports(model); extra != "" {
		b.WriteString(extra)
	}
	b.WriteString("from enum import Enum\n")
	b.WriteString("from typing import Dict, List, Optional, Union\n\n")
	b.WriteString("from .base_model import BaseModel\n")

	// Cross-package / custom type imports
	className := model.EffectiveName()
	if customImports := pythonCustomTypeImports(model, className); customImports != "" {
		b.WriteString(customImports)
	}
	b.WriteString("\n\n")

	// Nested enums
	for _, e := range model.Enums {
		b.WriteString(generatePythonEnum(e))
		b.WriteString("\n")
	}

	// Oneof type aliases
	for _, o := range model.Oneofs {
		b.WriteString(generatePythonOneofType(o))
	}

	if model.Extends != "" {
		extendsModule := toSnakeCase(model.Extends)
		b.WriteString(fmt.Sprintf("try:\n    from .%s import %s\nexcept ImportError:\n    %s = BaseModel\n\n\n", extendsModule, model.Extends, model.Extends))
	}

	// Deprecation
	if model.Deprecated {
		b.WriteString(fmt.Sprintf("# DEPRECATED: %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}

	b.WriteString("@dataclass(eq=False)\n")
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

	typeHint := fieldTypePython(f)

	defaultVal := pythonDefaultForField(f)
	if f.IsMap {
		b.WriteString(fmt.Sprintf("    %s: %s = field(default_factory=dict)\n", f.Name, typeHint))
	} else if f.Repeated {
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

	// If pb2ImportPrefix is set, add sys.path setup
	if prefix := pb2ImportPrefix(opts); prefix != "" {
		b.WriteString(pythonSysPathSetup(opts.OutputDir, prefix))
	}

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

func (g *PythonNoneGenerator) GenerateEnum(enum EnumDef, opts GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from enum import Enum\n\n\n")
	b.WriteString(generatePythonEnum(enum))

	fileName := toSnakeCase(enum.Name) + ".py"
	return GeneratedFile{Path: fileName, Content: b.String()}, nil
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
	b.WriteString("from typing import Any, ClassVar, Dict, Optional, Type, TypeVar\n")
	b.WriteString("from uuid import UUID, uuid4\n\n")
	b.WriteString("from google.protobuf.json_format import MessageToDict, ParseDict\n\n")
	b.WriteString("try:\n")
	b.WriteString("    from typing import Self\n")
	b.WriteString("except ImportError:\n")
	b.WriteString("    from typing_extensions import Self\n\n")
	b.WriteString("T = TypeVar(\"T\")\n\n")

	if g.isV2() {
		b.WriteString("from pydantic import BaseModel as PydanticBaseModel, ConfigDict, Field\n\n\n")
		b.WriteString("class BaseModel(PydanticBaseModel):\n")
		b.WriteString("    \"\"\"Base model for all buffalo-models generated models (pydantic v2).\"\"\"\n\n")
		b.WriteString("    model_config = ConfigDict(\n")
		b.WriteString("        from_attributes=True,\n")
		b.WriteString("        populate_by_name=True,\n")
		b.WriteString("        arbitrary_types_allowed=True,\n")
		b.WriteString("        extra=\"forbid\",\n")
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
	b.WriteString("    deleted_at: Optional[datetime] = None\n\n")

	// Operator overloads: compare only model-specific fields, ignoring base class fields.
	b.WriteString("    _base_model_fields: ClassVar[frozenset] = frozenset(\n")
	b.WriteString("        {\"id\", \"created_at\", \"updated_at\", \"deleted_at\"}\n")
	b.WriteString("    )\n\n")
	b.WriteString("    def __eq__(self, other: object) -> bool:\n")
	b.WriteString("        \"\"\"Compare models by own fields only, ignoring base class fields.\"\"\"\n")
	b.WriteString("        if not isinstance(other, self.__class__):\n")
	b.WriteString("            return NotImplemented\n")
	if g.isV2() {
		b.WriteString("        _fields = type(self).model_fields\n")
	} else {
		b.WriteString("        _fields = type(self).__fields__\n")
	}
	b.WriteString("        for name in _fields:\n")
	b.WriteString("            if name in self._base_model_fields:\n")
	b.WriteString("                continue\n")
	b.WriteString("            if getattr(self, name) != getattr(other, name):\n")
	b.WriteString("                return False\n")
	b.WriteString("        return True\n\n")
	b.WriteString("    def __ne__(self, other: object) -> bool:\n")
	b.WriteString("        result = self.__eq__(other)\n")
	b.WriteString("        if result is NotImplemented:\n")
	b.WriteString("            return result\n")
	b.WriteString("        return not result\n\n")
	b.WriteString("    def __hash__(self) -> int:\n")
	b.WriteString("        \"\"\"Hash by own fields only, ignoring base class fields.\"\"\"\n")
	if g.isV2() {
		b.WriteString("        _fields = type(self).model_fields\n")
	} else {
		b.WriteString("        _fields = type(self).__fields__\n")
	}
	b.WriteString("        values: list = []\n")
	b.WriteString("        for name in sorted(_fields):\n")
	b.WriteString("            if name in self._base_model_fields:\n")
	b.WriteString("                continue\n")
	b.WriteString("            val = getattr(self, name, None)\n")
	b.WriteString("            try:\n")
	b.WriteString("                hash(val)\n")
	b.WriteString("                values.append((name, val))\n")
	b.WriteString("            except TypeError:\n")
	b.WriteString("                values.append((name, repr(val)))\n")
	b.WriteString("        return hash(tuple(values))\n")
	b.WriteString("\n\n")

	if g.isV2() {
		b.WriteString("class ProtoBaseModel(BaseModel):\n")
		b.WriteString("    \"\"\"Базовая модель с поддержкой автоматической конвертации Protobuf.\"\"\"\n\n")
		b.WriteString("    proto_class: ClassVar[Type[Any] | None] = None\n")
		b.WriteString("    # Поля базового класса, которые НЕ являются частью proto-схемы.\n")
		b.WriteString("    # Исключаются при конвертации to_proto / from_proto.\n")
		b.WriteString("    @classmethod\n")
		b.WriteString("    def _strip_base_fields(cls, d: Dict[str, Any]) -> Dict[str, Any]:\n")
		b.WriteString("        \"\"\"Рекурсивно удаляет поля базового класса из словаря.\n\n")
		b.WriteString("        Обрабатывает вложенные модели (dict) и списки моделей (list[dict]),\n")
		b.WriteString("        чтобы ParseDict не получал поля, отсутствующие в proto-схеме.\n")
		b.WriteString("        \"\"\"\n")
		b.WriteString("        result: Dict[str, Any] = {}\n")
		b.WriteString("        for k, v in d.items():\n")
		b.WriteString("            if k in cls._base_model_fields:\n")
		b.WriteString("                continue\n")
		b.WriteString("            if isinstance(v, dict):\n")
		b.WriteString("                result[k] = cls._strip_base_fields(v)\n")
		b.WriteString("            elif isinstance(v, list):\n")
		b.WriteString("                result[k] = [\n")
		b.WriteString("                    cls._strip_base_fields(item) if isinstance(item, dict) else item\n")
		b.WriteString("                    for item in v\n")
		b.WriteString("                ]\n")
		b.WriteString("            else:\n")
		b.WriteString("                result[k] = v\n")
		b.WriteString("        return result\n\n")
		b.WriteString("    @classmethod\n")
		b.WriteString("    def from_proto(cls, proto_msg: Any) -> Self:\n")
		b.WriteString("        \"\"\"Конвертирует Protobuf сообщение в Pydantic модель.\n\n")
		b.WriteString("        Поля базового класса (id, created_at, updated_at, deleted_at)\n")
		b.WriteString("        не заполняются из proto-сообщения, так как они отсутствуют в схеме.\n")
		b.WriteString("        Они будут проинициализированы значениями по умолчанию.\n")
		b.WriteString("        \"\"\"\n")
		b.WriteString("        try:\n")
		b.WriteString("            dict_obj = MessageToDict(\n")
		b.WriteString("                proto_msg,\n")
		b.WriteString("                preserving_proto_field_name=True,\n")
		b.WriteString("                use_integers_for_enums=True,\n")
		b.WriteString("                including_default_value_fields=True,\n")
		b.WriteString("            )\n")
		b.WriteString("        except TypeError:\n")
		b.WriteString("            dict_obj = MessageToDict(\n")
		b.WriteString("                proto_msg,\n")
		b.WriteString("                preserving_proto_field_name=True,\n")
		b.WriteString("                use_integers_for_enums=True,\n")
		b.WriteString("            )\n")
		b.WriteString("        return cls.model_validate(cls._strip_base_fields(dict_obj))\n\n")
		b.WriteString("    def to_proto_dict(self) -> Dict[str, Any]:\n")
		b.WriteString("        \"\"\"Возвращает dict, пригодный для передачи в ParseDict.\n\n")
		b.WriteString("        В отличие от model_dump(), этот метод рекурсивно удаляет поля\n")
		b.WriteString("        базового класса (id, created_at, updated_at, deleted_at) на всех\n")
		b.WriteString("        уровнях вложенности, поэтому результат совместим с proto-схемой.\n\n")
		b.WriteString("        Пример использования в servicer:\n")
		b.WriteString("            d = my_model.to_proto_dict()\n")
		b.WriteString("            ParseDict(d, SomeProtoMessage())\n")
		b.WriteString("        \"\"\"\n")
		b.WriteString("        dict_obj = self.model_dump(\n")
		b.WriteString("            mode=\"json\",\n")
		b.WriteString("            exclude_none=True,\n")
		b.WriteString("            by_alias=True,\n")
		b.WriteString("            exclude=self._base_model_fields,\n")
		b.WriteString("        )\n")
		b.WriteString("        return self._strip_base_fields(dict_obj)\n\n")
		b.WriteString("    def to_proto(self, proto_class: Type[T] | None = None) -> Any:\n")
		b.WriteString("        \"\"\"Конвертирует Pydantic модель в Protobuf сообщение.\"\"\"\n")
		b.WriteString("        target_proto_class = proto_class or self.proto_class\n")
		b.WriteString("        if target_proto_class is None:\n")
		b.WriteString("            raise NotImplementedError(\n")
		b.WriteString("                \"Subclasses must define proto_class or pass proto_class to to_proto\"\n")
		b.WriteString("            )\n")
		b.WriteString("        # mode=\"json\" гарантирует, что datetime/timedelta станут строками/числами\n")
		b.WriteString("        # которые ParseDict умеет превращать в Timestamp/Duration.\n")
		b.WriteString("        # to_proto_dict() рекурсивно убирает поля базового класса (id, timestamps)\n")
		b.WriteString("        # на всех уровнях вложенности.\n")
		b.WriteString("        return ParseDict(self.to_proto_dict(), target_proto_class())\n")
	} else {
		b.WriteString("class ProtoBaseModel(BaseModel):\n")
		b.WriteString("    \"\"\"Base model with protobuf conversion support (pydantic v1).\"\"\"\n\n")
		b.WriteString("    proto_class: ClassVar[Type[Any] | None] = None\n")
		b.WriteString("    # Fields from BaseModel that are NOT part of the proto schema.\n")
		b.WriteString("    # Excluded during to_proto / from_proto conversion.\n")
		b.WriteString("    _base_model_fields: ClassVar[frozenset] = frozenset(\n")
		b.WriteString("        {\"id\", \"created_at\", \"updated_at\", \"deleted_at\"}\n")
		b.WriteString("    )\n\n")
		b.WriteString("    @classmethod\n")
		b.WriteString("    def _strip_base_fields(cls, d):\n")
		b.WriteString("        \"\"\"Recursively remove base model fields from dict.\"\"\"\n")
		b.WriteString("        result = {}\n")
		b.WriteString("        for k, v in d.items():\n")
		b.WriteString("            if k in cls._base_model_fields:\n")
		b.WriteString("                continue\n")
		b.WriteString("            if isinstance(v, dict):\n")
		b.WriteString("                result[k] = cls._strip_base_fields(v)\n")
		b.WriteString("            elif isinstance(v, list):\n")
		b.WriteString("                result[k] = [\n")
		b.WriteString("                    cls._strip_base_fields(item) if isinstance(item, dict) else item\n")
		b.WriteString("                    for item in v\n")
		b.WriteString("                ]\n")
		b.WriteString("            else:\n")
		b.WriteString("                result[k] = v\n")
		b.WriteString("        return result\n\n")
		b.WriteString("    @classmethod\n")
		b.WriteString("    def from_proto(cls, proto_msg: Any) -> \"ProtoBaseModel\":\n")
		b.WriteString("        \"\"\"Converts a Protobuf message to a Pydantic model.\n\n")
		b.WriteString("        Base model fields (id, created_at, updated_at, deleted_at)\n")
		b.WriteString("        are not populated from the proto message as they are not in the schema.\n")
		b.WriteString("        They will be initialized with default values.\n")
		b.WriteString("        \"\"\"\n")
		b.WriteString("        try:\n")
		b.WriteString("            dict_obj = MessageToDict(\n")
		b.WriteString("                proto_msg,\n")
		b.WriteString("                preserving_proto_field_name=True,\n")
		b.WriteString("                use_integers_for_enums=True,\n")
		b.WriteString("                including_default_value_fields=True,\n")
		b.WriteString("            )\n")
		b.WriteString("        except TypeError:\n")
		b.WriteString("            dict_obj = MessageToDict(\n")
		b.WriteString("                proto_msg,\n")
		b.WriteString("                preserving_proto_field_name=True,\n")
		b.WriteString("                use_integers_for_enums=True,\n")
		b.WriteString("            )\n")
		b.WriteString("        return cls.parse_obj(cls._strip_base_fields(dict_obj))\n\n")
		b.WriteString("    def to_proto_dict(self):\n")
		b.WriteString("        \"\"\"Returns a dict suitable for passing to ParseDict.\n\n")
		b.WriteString("        Unlike model_dump(), this method recursively strips base model\n")
		b.WriteString("        fields (id, created_at, updated_at, deleted_at) at all nesting\n")
		b.WriteString("        levels so the result is compatible with the proto schema.\n\n")
		b.WriteString("        Example usage in servicer:\n")
		b.WriteString("            d = my_model.to_proto_dict()\n")
		b.WriteString("            ParseDict(d, SomeProtoMessage())\n")
		b.WriteString("        \"\"\"\n")
		b.WriteString("        return self._strip_base_fields(\n")
		b.WriteString("            self.dict(exclude_none=True, by_alias=True, exclude=self._base_model_fields)\n")
		b.WriteString("        )\n\n")
		b.WriteString("    def to_proto(self, proto_class: Type[T] | None = None) -> Any:\n")
		b.WriteString("        target_proto_class = proto_class or self.proto_class\n")
		b.WriteString("        if target_proto_class is None:\n")
		b.WriteString("            raise NotImplementedError(\n")
		b.WriteString("                \"Subclasses must define proto_class or pass proto_class to to_proto\"\n")
		b.WriteString("            )\n")
		b.WriteString("        # Recursively exclude base model fields at all nesting levels\n")
		b.WriteString("        return ParseDict(self.to_proto_dict(), target_proto_class())\n")
	}

	return GeneratedFile{Path: "base_model.py", Content: b.String()}, nil
}

func (g *PythonPydanticGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (pydantic)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from enum import Enum\n")
	b.WriteString("from typing import Any, ClassVar, Dict, List, Optional, Type, Union\n\n")
	b.WriteString("try:\n")
	b.WriteString("    from typing import Self, override\n")
	b.WriteString("except ImportError:\n")
	b.WriteString("    from typing_extensions import Self, override\n\n")

	// Dynamic imports for well-known types (datetime, timedelta, etc.)
	if extra := pythonExtraImports(model); extra != "" {
		b.WriteString(extra)
		b.WriteString("\n")
	}

	if g.isV2() {
		b.WriteString("from pydantic import ConfigDict, Field\n\n")
	} else {
		b.WriteString("from pydantic import Field\n\n")
	}
	b.WriteString("from google.protobuf.message import Message\n\n")
	b.WriteString("from .base_model import ProtoBaseModel\n")

	// Cross-package / custom type imports
	className := model.EffectiveName()
	if customImports := pythonCustomTypeImports(model, className); customImports != "" {
		b.WriteString(customImports)
	}

	// Derive pb2 import path from proto file path
	pb2Module := protoPb2Module(model.FilePath)
	if prefix := pb2ImportPrefix(opts); prefix != "" {
		pb2Module = prefix + "." + pb2Module
	}
	b.WriteString(fmt.Sprintf("\ntry:\n    from %s import %s as _ProtoClass\nexcept ImportError:\n    _ProtoClass = None  # type: ignore[assignment]\n", pb2Module, model.MessageName))
	b.WriteString("\n\n")

	// Nested enums
	for _, e := range model.Enums {
		b.WriteString(generatePythonEnum(e))
		b.WriteString("\n")
	}

	// Oneof type aliases
	for _, o := range model.Oneofs {
		b.WriteString(generatePythonOneofType(o))
	}

	if model.Extends != "" {
		extendsModule := toSnakeCase(model.Extends)
		b.WriteString(fmt.Sprintf("try:\n    from .%s import %s\nexcept ImportError:\n    %s = ProtoBaseModel\n\n\n", extendsModule, model.Extends, model.Extends))
	}

	if model.Deprecated {
		b.WriteString(fmt.Sprintf("# DEPRECATED: %s\n", deprecatedComment(true, model.DeprecatedMessage)))
	}

	parentClass := "ProtoBaseModel"
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

	b.WriteString("\n")
	b.WriteString("    proto_class: ClassVar[Type[Message] | None] = _ProtoClass\n\n")
	b.WriteString("    @classmethod\n")
	b.WriteString("    @override\n")
	b.WriteString("    def from_proto(cls, proto_msg: Message) -> Self:\n")
	b.WriteString("        \"\"\"Override-friendly protobuf -> model conversion.\"\"\"\n")
	b.WriteString("        return super().from_proto(proto_msg)\n\n")
	b.WriteString("    @override\n")
	b.WriteString("    def to_proto(self) -> Message:\n")
	b.WriteString("        \"\"\"Override-friendly model -> protobuf conversion.\"\"\"\n")
	b.WriteString("        return super().to_proto(proto_class=type(self).proto_class)\n")
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

	typeHint := fieldTypePython(f)

	// Build Field(...) arguments
	var fieldArgs []string

	// Default value
	if f.IsMap {
		fieldArgs = append(fieldArgs, "default_factory=dict")
	} else if f.Repeated {
		fieldArgs = append(fieldArgs, "default_factory=list")
	} else if f.DefaultValue != "" {
		if f.ProtoType == "string" {
			fieldArgs = append(fieldArgs, fmt.Sprintf("default=%s", pythonStringLiteral(f.DefaultValue)))
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
		case "int32", "int64", "uint32", "uint64", "sint32", "sint64",
			"fixed32", "fixed64", "sfixed32", "sfixed64":
			fieldArgs = append(fieldArgs, "default=0")
		case "float", "double":
			fieldArgs = append(fieldArgs, "default=0.0")
		default:
			if f.IsEnum {
				// Enum fields default to first value (0)
				fieldArgs = append(fieldArgs, "default=0")
			} else if isCustomProtoType(f.ProtoType) {
				// Message-type fields default to None (proto3 semantics)
				fieldArgs = append(fieldArgs, "default=None")
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
		fieldArgs = append(fieldArgs, fmt.Sprintf("description=%s", pythonStringLiteral(f.Description)))
	}
	if f.Example != "" {
		if g.isV2() {
			fieldArgs = append(fieldArgs, fmt.Sprintf("examples=[%s]", pythonStringLiteral(f.Example)))
		} else {
			fieldArgs = append(fieldArgs, fmt.Sprintf("example=%s", pythonStringLiteral(f.Example)))
		}
	}
	if f.Alias != "" {
		fieldArgs = append(fieldArgs, fmt.Sprintf("alias=%s", pythonStringLiteral(f.Alias)))
	}
	if f.JSONName != "" && f.JSONName != f.Name {
		fieldArgs = append(fieldArgs, fmt.Sprintf("serialization_alias=%s", pythonStringLiteral(f.JSONName)))
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
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (pydantic)"))

	// If pb2ImportPrefix is set, add sys.path setup so that
	// pb2 imports like "from <prefix>.<proto_module>_pb2 import ..." resolve correctly.
	if prefix := pb2ImportPrefix(opts); prefix != "" {
		b.WriteString(pythonSysPathSetup(opts.OutputDir, prefix))
	}

	b.WriteString("\nfrom .base_model import BaseModel, ProtoBaseModel\n")
	for _, m := range models {
		className := m.EffectiveName()
		fileName := toSnakeCase(m.MessageName)
		b.WriteString(fmt.Sprintf("from .%s import %s\n", fileName, className))
	}
	b.WriteString("\n__all__ = [\n")
	b.WriteString("    \"BaseModel\",\n")
	b.WriteString("    \"ProtoBaseModel\",\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("    \"%s\",\n", m.EffectiveName()))
	}
	b.WriteString("]\n")

	return GeneratedFile{Path: "__init__.py", Content: b.String()}, nil
}

func (g *PythonPydanticGenerator) GenerateEnum(enum EnumDef, opts GenerateOptions) (GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (pydantic)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")
	b.WriteString("from enum import Enum\n\n\n")
	b.WriteString(generatePythonEnum(enum))

	fileName := toSnakeCase(enum.Name) + ".py"
	return GeneratedFile{Path: fileName, Content: b.String()}, nil
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

	// Operator overloads: compare only model-specific fields, ignoring base class fields.
	b.WriteString("\n")
	b.WriteString("    _base_model_fields = frozenset({\"id\", \"created_at\", \"updated_at\", \"deleted_at\"})\n\n")
	b.WriteString("    def __eq__(self, other: object) -> bool:\n")
	b.WriteString("        \"\"\"Compare models by own fields only, ignoring base class fields.\"\"\"\n")
	b.WriteString("        if not isinstance(other, self.__class__):\n")
	b.WriteString("            return NotImplemented\n")
	b.WriteString("        for k in self.__dict__:\n")
	b.WriteString("            if k.startswith(\"_\") or k in self._base_model_fields:\n")
	b.WriteString("                continue\n")
	b.WriteString("            if self.__dict__[k] != other.__dict__.get(k):\n")
	b.WriteString("                return False\n")
	b.WriteString("        return True\n\n")
	b.WriteString("    def __ne__(self, other: object) -> bool:\n")
	b.WriteString("        result = self.__eq__(other)\n")
	b.WriteString("        if result is NotImplemented:\n")
	b.WriteString("            return result\n")
	b.WriteString("        return not result\n\n")
	b.WriteString("    def __hash__(self) -> int:\n")
	b.WriteString("        \"\"\"Hash by own fields only, ignoring base class fields.\"\"\"\n")
	b.WriteString("        values: list = []\n")
	b.WriteString("        for k in sorted(self.__dict__):\n")
	b.WriteString("            if k.startswith(\"_\") or k in self._base_model_fields:\n")
	b.WriteString("                continue\n")
	b.WriteString("            v = self.__dict__[k]\n")
	b.WriteString("            try:\n")
	b.WriteString("                hash(v)\n")
	b.WriteString("                values.append((k, v))\n")
	b.WriteString("            except TypeError:\n")
	b.WriteString("                values.append((k, repr(v)))\n")
	b.WriteString("        return hash(tuple(values))\n")

	return GeneratedFile{Path: "base_model.py", Content: b.String()}, nil
}

func (g *PythonSQLAlchemyGenerator) GenerateModel(model ModelDef, opts GenerateOptions) ([]GeneratedFile, error) {
	var b strings.Builder
	b.WriteString(pythonHeader("buffalo-models (sqlalchemy)"))
	b.WriteString("\nfrom __future__ import annotations\n\n")

	// Dynamic imports for well-known types (datetime, timedelta, etc.)
	if extra := pythonExtraImports(model); extra != "" {
		b.WriteString(extra)
		b.WriteString("\n")
	}

	if g.isV2() {
		b.WriteString("from sqlalchemy import CheckConstraint, Index, String, Integer, Float, Boolean\n")
		b.WriteString("from sqlalchemy.orm import Mapped, mapped_column, relationship\n\n")
	} else {
		b.WriteString("from sqlalchemy import CheckConstraint, Column, Index, String, Integer, Float, Boolean\n")
		b.WriteString("from sqlalchemy.orm import relationship\n\n")
	}
	b.WriteString("from .base_model import BaseModel\n")

	// Cross-package / custom type imports
	className := model.EffectiveName()
	if customImports := pythonCustomTypeImports(model, className); customImports != "" {
		b.WriteString(customImports)
	}
	b.WriteString("\n\n")
	if model.Extends != "" {
		extendsModule := toSnakeCase(model.Extends)
		b.WriteString(fmt.Sprintf("try:\n    from .%s import %s\nexcept ImportError:\n    %s = BaseModel\n\n\n", extendsModule, model.Extends, model.Extends))
	}

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
		pyType := fieldTypePython(f)
		b.WriteString(fmt.Sprintf("    %s: Mapped[%s] = mapped_column(%s)\n",
			f.Name, pyType, strings.Join(colArgs, ", ")))
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

func (g *PythonSQLAlchemyGenerator) GenerateEnum(enum EnumDef, opts GenerateOptions) (GeneratedFile, error) {
	none := &PythonNoneGenerator{}
	return none.GenerateEnum(enum, opts)
}
