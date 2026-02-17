// Package models provides proto model annotation parsing, code generation,
// and ORM integration for buffalo.models.
//
// It generates typed model classes / structs for Go, Python, C++, and Rust
// from protobuf messages annotated with [(buffalo.models.model)] and
// [(buffalo.models.field)]. The generation output varies depending on
// the configured ORM plugin (e.g. pydantic, sqlalchemy, gorm, diesel).
package models

import "fmt"

// ══════════════════════════════════════════════════════════════════
//  Field visibility & behavior enums
// ══════════════════════════════════════════════════════════════════

// FieldVisibility mirrors the proto enum FieldVisibility.
type FieldVisibility int

const (
	VisibilityDefault   FieldVisibility = 0
	VisibilityPublic    FieldVisibility = 1
	VisibilityInternal  FieldVisibility = 2
	VisibilityExternal  FieldVisibility = 3
	VisibilityPrivate   FieldVisibility = 4
	VisibilityProtected FieldVisibility = 5
)

func (v FieldVisibility) String() string {
	switch v {
	case VisibilityPublic:
		return "public"
	case VisibilityInternal:
		return "internal"
	case VisibilityExternal:
		return "external"
	case VisibilityPrivate:
		return "private"
	case VisibilityProtected:
		return "protected"
	default:
		return "default"
	}
}

// FieldBehavior mirrors the proto enum FieldBehavior.
type FieldBehavior int

const (
	BehaviorDefault    FieldBehavior = 0
	BehaviorReadOnly   FieldBehavior = 1
	BehaviorWriteOnly  FieldBehavior = 2
	BehaviorImmutable  FieldBehavior = 3
	BehaviorComputed   FieldBehavior = 4
	BehaviorVirtual    FieldBehavior = 5
	BehaviorOutputOnly FieldBehavior = 6
	BehaviorInputOnly  FieldBehavior = 7
)

func (b FieldBehavior) String() string {
	switch b {
	case BehaviorReadOnly:
		return "readonly"
	case BehaviorWriteOnly:
		return "writeonly"
	case BehaviorImmutable:
		return "immutable"
	case BehaviorComputed:
		return "computed"
	case BehaviorVirtual:
		return "virtual"
	case BehaviorOutputOnly:
		return "output_only"
	case BehaviorInputOnly:
		return "input_only"
	default:
		return "default"
	}
}

// RelationType mirrors the proto enum RelationType.
type RelationType int

const (
	RelationUnspecified RelationType = 0
	RelationBelongsTo   RelationType = 1
	RelationHasOne      RelationType = 2
	RelationHasMany     RelationType = 3
	RelationManyToMany  RelationType = 4
)

func (r RelationType) String() string {
	switch r {
	case RelationBelongsTo:
		return "belongs_to"
	case RelationHasOne:
		return "has_one"
	case RelationHasMany:
		return "has_many"
	case RelationManyToMany:
		return "many_to_many"
	default:
		return "unspecified"
	}
}

// OnAction mirrors referential integrity actions.
type OnAction int

const (
	ActionUnspecified OnAction = 0
	ActionCascade     OnAction = 1
	ActionSetNull     OnAction = 2
	ActionSetDefault  OnAction = 3
	ActionRestrict    OnAction = 4
	ActionNoAction    OnAction = 5
)

func (a OnAction) String() string {
	switch a {
	case ActionCascade:
		return "CASCADE"
	case ActionSetNull:
		return "SET NULL"
	case ActionSetDefault:
		return "SET DEFAULT"
	case ActionRestrict:
		return "RESTRICT"
	case ActionNoAction:
		return "NO ACTION"
	default:
		return ""
	}
}

// IndexType defines index algorithm.
type IndexType int

const (
	IndexDefault IndexType = 0
	IndexBTree   IndexType = 1
	IndexHash    IndexType = 2
	IndexGIN     IndexType = 3
	IndexGIST    IndexType = 4
	IndexBRIN    IndexType = 5
)

func (t IndexType) String() string {
	switch t {
	case IndexBTree:
		return "btree"
	case IndexHash:
		return "hash"
	case IndexGIN:
		return "gin"
	case IndexGIST:
		return "gist"
	case IndexBRIN:
		return "brin"
	default:
		return ""
	}
}

// ══════════════════════════════════════════════════════════════════
//  Core types
// ══════════════════════════════════════════════════════════════════

// ModelDef represents a parsed model definition from a proto message.
type ModelDef struct {
	// Proto source
	MessageName string // original proto message name
	Package     string // proto package (e.g. "myservice")
	FilePath    string // source .proto file path

	// Model-level options (from [(buffalo.models.model)])
	Name              string   // class/struct name override
	TableName         string   // optional DB table name
	Schema            string   // optional DB schema
	Description       string   // docstring
	Tags              []string // arbitrary tags
	Abstract          bool     // abstract model
	Extends           string   // parent model name
	Mixins            []string // mixin model names
	SoftDelete        bool     // add deleted_at
	Timestamps        bool     // add created_at / updated_at
	Deprecated        bool     // deprecated model
	DeprecatedMessage string
	Generate          []string // what to generate: "model", "repo", "factory"

	// Composite constraints
	Indexes []IndexDef
	Uniques []UniqueDef
	Checks  []CheckDef

	// Nested enums within this message
	Enums []EnumDef

	// Oneof groups within this message
	Oneofs []OneofDef

	// Nested messages (sub-structs) within this message
	NestedMessages []ModelDef

	// Fields (ordered)
	Fields []FieldDef
}

// EffectiveName returns the model name: Name override or MessageName.
func (m *ModelDef) EffectiveName() string {
	if m.Name != "" {
		return m.Name
	}
	return m.MessageName
}

// EnumDef describes a proto enum extracted from a message or top-level scope.
type EnumDef struct {
	Name    string      // enum type name
	Comment string      // leading comment / docstring
	Values  []EnumValue // enum constants
}

// EnumValue describes a single enum constant.
type EnumValue struct {
	Name    string
	Number  int32
	Comment string // inline or leading comment
}

// OneofDef describes a proto oneof group within a message.
type OneofDef struct {
	Name    string     // oneof group name
	Comment string     // leading comment / docstring
	Fields  []FieldDef // fields within the oneof
}

// FieldDef represents a parsed field within a model.
type FieldDef struct {
	// Proto source
	Name      string // proto field name (snake_case)
	ProtoType string // proto type (string, int32, etc.)
	Number    int    // proto field number
	Repeated  bool   // repeated field

	// Map field support: map<KeyType, ValueType>
	IsMap        bool   // true when the field is a proto map
	MapKeyType   string // key type for map fields (e.g. "string")
	MapValueType string // value type for map fields (e.g. "string")

	// Oneof support
	OneofGroup string // name of the oneof group this field belongs to

	// Enum support
	IsEnum       bool   // true when the field type is a proto enum
	EnumTypeName string // original enum type name (e.g. "SourceStatus")

	// From [(buffalo.models.field)]
	Alias             string
	Description       string
	PrimaryKey        bool
	AutoIncrement     bool
	Nullable          bool
	Unique            bool
	DefaultValue      string
	MaxLength         int32
	MinLength         int32
	Precision         int32
	Scale             int32
	CustomType        string
	DBType            string
	Visibility        FieldVisibility
	Behavior          FieldBehavior
	Sensitive         bool
	Deprecated        bool
	DeprecatedMessage string
	Index             bool
	IndexType         IndexType
	JSONName          string
	XMLName           string
	OmitEmpty         bool
	Relation          *RelationDef
	Example           string
	Comment           string
	AutoGenerate      bool
	AutoNow           bool
	AutoNowAdd        bool
	Sequence          string
	Collation         string
	Ignore            bool
	DBIgnore          bool
	APIIgnore         bool
	Tags              []string
	Metadata          map[string]string
}

// EffectiveJSONName returns the serialization name for JSON.
func (f *FieldDef) EffectiveJSONName() string {
	if f.JSONName != "" {
		return f.JSONName
	}
	return f.Name
}

// IsSerializable returns true if the field should be included in serialization.
func (f *FieldDef) IsSerializable() bool {
	if f.Ignore || f.APIIgnore {
		return false
	}
	if f.Visibility == VisibilityPrivate {
		return false
	}
	if f.Behavior == BehaviorWriteOnly {
		return false
	}
	return true
}

// IsPersistable returns true if the field should be included in DB operations.
func (f *FieldDef) IsPersistable() bool {
	if f.Ignore || f.DBIgnore {
		return false
	}
	if f.Behavior == BehaviorVirtual || f.Behavior == BehaviorComputed {
		return false
	}
	return true
}

// RelationDef describes a relationship between models.
type RelationDef struct {
	Type       RelationType
	Model      string // target model name
	ForeignKey string
	References string // default "id"
	JoinTable  string // for many_to_many
	OnDelete   OnAction
	OnUpdate   OnAction
	Eager      bool
	Through    string // intermediate model
	InverseOf  string // inverse relation field
}

// IndexDef describes a composite index.
type IndexDef struct {
	Name    string
	Columns []string
	Unique  bool
	Type    IndexType
	Where   string // partial index condition
	Comment string
}

// UniqueDef describes a composite unique constraint.
type UniqueDef struct {
	Name    string
	Columns []string
}

// CheckDef describes a CHECK constraint.
type CheckDef struct {
	Name       string
	Expression string
}

// ══════════════════════════════════════════════════════════════════
//  ORM plugin configuration
// ══════════════════════════════════════════════════════════════════

// ORMPlugin describes which ORM framework to use for generation.
type ORMPlugin struct {
	Name    string // "None", "pydantic", "sqlalchemy", "gorm", "diesel", etc.
	Version string // optional required version (e.g. "2.0")
}

// ParseORMPlugin parses "plugin_name[@version]" notation.
//
// Examples:
//
//	"default"       → {Name: "None", Version: ""}
//	"None"          → {Name: "None", Version: ""}
//	"pydantic"      → {Name: "pydantic", Version: ""}
//	"pydantic@2.0"  → {Name: "pydantic", Version: "2.0"}
//	""              → {Name: "None", Version: ""}
func ParseORMPlugin(s string) ORMPlugin {
	if s == "" || s == "default" {
		return ORMPlugin{Name: "None"}
	}

	for i, c := range s {
		if c == '@' {
			return ORMPlugin{
				Name:    s[:i],
				Version: s[i+1:],
			}
		}
	}
	return ORMPlugin{Name: s}
}

// String formats the plugin back to "name[@version]".
func (p ORMPlugin) String() string {
	if p.Version != "" {
		return fmt.Sprintf("%s@%s", p.Name, p.Version)
	}
	return p.Name
}

// IsNone returns true for "None" / "default" / empty.
func (p ORMPlugin) IsNone() bool {
	return p.Name == "" || p.Name == "None" || p.Name == "default"
}

// ══════════════════════════════════════════════════════════════════
//  Models generation options
// ══════════════════════════════════════════════════════════════════

// GenerateOptions controls the code generation process.
type GenerateOptions struct {
	// Language target (go, python, cpp, rust).
	Language string

	// ORM plugin to use.
	ORM ORMPlugin

	// Output directory for generated models.
	OutputDir string

	// Package / module name.
	Package string

	// BaseModelFields — extra fields injected into BaseModel.
	BaseModelFields []FieldDef

	// GenerateBaseModel — whether to emit a BaseModel class/struct.
	GenerateBaseModel bool

	// GenerateInit — generate __init__.py for Python.
	GenerateInit bool

	// FixImports — fix relative imports for Python.
	FixImports bool

	// PreserveProtoStructure — mirror proto directory structure.
	PreserveProtoStructure bool

	// Verbose — emit extra comments in generated code.
	Verbose bool
}
