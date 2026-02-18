package models

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════
//  Factory tests
// ══════════════════════════════════════════════════════════════════

func TestNewModelCodeGenerator_AllLanguages(t *testing.T) {
	cases := []struct {
		lang string
		orm  string
		want string
	}{
		// Python (always pydantic)
		{"python", "None", "pydantic"},
		{"python", "pydantic", "pydantic"},
		{"python", "pydantic@2.0", "pydantic"},
		{"python", "sqlalchemy", "pydantic"},
		// Go
		{"go", "None", "None"},
		{"go", "gorm", "gorm"},
		{"go", "sqlx", "sqlx"},
		// Rust
		{"rust", "None", "None"},
		{"rust", "diesel", "diesel"},
		// C++
		{"cpp", "None", "None"},
	}

	for _, tc := range cases {
		orm := ParseORMPlugin(tc.orm)
		gen, err := NewModelCodeGenerator(tc.lang, orm)
		if err != nil {
			t.Errorf("NewModelCodeGenerator(%q, %q) err: %v", tc.lang, tc.orm, err)
			continue
		}
		if gen.Language() != tc.lang {
			t.Errorf("Language() = %q, want %q", gen.Language(), tc.lang)
		}
		if gen.ORMName() != tc.want {
			t.Errorf("ORMName() = %q, want %q", gen.ORMName(), tc.want)
		}
	}
}

func TestNewModelCodeGenerator_UnsupportedLang(t *testing.T) {
	orm := ORMPlugin{Name: "None"}
	_, err := NewModelCodeGenerator("lua", orm)
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestNewModelCodeGenerator_UnsupportedORM(t *testing.T) {
	cases := []struct {
		lang string
		orm  string
	}{
		{"go", "django"},
		{"rust", "gorm"},
		{"cpp", "gorm"},
	}
	for _, tc := range cases {
		orm := ParseORMPlugin(tc.orm)
		_, err := NewModelCodeGenerator(tc.lang, orm)
		if err == nil {
			t.Errorf("expected error for %s/%s", tc.lang, tc.orm)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  Shared test model
// ══════════════════════════════════════════════════════════════════

func testModel() ModelDef {
	return ModelDef{
		MessageName: "UserProfile",
		Package:     "myapp.models",
		Name:        "UserProfile",
		TableName:   "user_profiles",
		Description: "User profile model",
		Generate:    []string{"model"},
		Fields: []FieldDef{
			{
				Name:      "email",
				ProtoType: "string",
				Unique:    true,
				MaxLength: 255,
				JSONName:  "email",
			},
			{
				Name:       "display_name",
				ProtoType:  "string",
				MaxLength:  100,
				Nullable:   true,
				Visibility: VisibilityPublic,
			},
			{
				Name:       "password_hash",
				ProtoType:  "string",
				Visibility: VisibilityPrivate,
				Behavior:   BehaviorWriteOnly,
				Sensitive:  true,
			},
			{
				Name:      "age",
				ProtoType: "int32",
				Nullable:  true,
			},
			{
				Name:         "role",
				ProtoType:    "string",
				Visibility:   VisibilityInternal,
				DefaultValue: "user",
			},
			{
				Name:      "is_active",
				ProtoType: "bool",
			},
			{
				Name:      "tags",
				ProtoType: "string",
				Repeated:  true,
			},
			{
				Name:      "score",
				ProtoType: "double",
				Behavior:  BehaviorComputed,
				DBIgnore:  true,
			},
		},
	}
}

func testOpts() GenerateOptions {
	return GenerateOptions{
		Package: "myapp.models",
	}
}

// ══════════════════════════════════════════════════════════════════
//  Python generator tests
// ══════════════════════════════════════════════════════════════════

func TestPythonPydanticGenerator_Model(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	m := testModel()
	m.FilePath = "araviec/common/v1/resolution.proto"
	files, err := gen.GenerateModel(m, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "class UserProfile")
	assertContains(t, content, "ProtoBaseModel")
	assertContains(t, content, "Field(")
	assertContains(t, content, "def from_proto")
	assertContains(t, content, "def to_proto")
	assertContains(t, content, "from araviec.common.v1.resolution_pb2 import UserProfile as _ProtoClass")
}

func TestPythonPydanticGenerator_Model_Pb2Import_UsesBaseDirPrefix(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	m := testModel()
	m.FilePath = "araviec/common/v1/resolution.proto"

	opts := testOpts()
	opts.Pb2ImportPrefix = "araviec_apis.generated.python"

	files, err := gen.GenerateModel(m, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "from araviec_apis.generated.python.araviec.common.v1.resolution_pb2 import UserProfile as _ProtoClass")
}

func TestPythonPydanticGenerator_Model_Pb2Import_NoPrefixWhenEmpty(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	m := testModel()
	m.FilePath = "araviec/common/v1/resolution.proto"

	opts := testOpts()
	// Pb2ImportPrefix is empty — no prefix should be added
	opts.Pb2ImportPrefix = ""

	files, err := gen.GenerateModel(m, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "from araviec.common.v1.resolution_pb2 import UserProfile as _ProtoClass")
}

func TestPythonPydanticGenerator_Model_EscapesQuotedDescription(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	m := testModel()
	m.Fields = []FieldDef{
		{
			Name:        "label",
			ProtoType:   "string",
			Description: "Например: \"Нижнее\", \"Троллинг\", \"Переходный режим\"",
		},
	}

	files, err := gen.GenerateModel(m, testOpts())
	if err != nil {
		t.Fatal(err)
	}

	content := files[0].Content
	assertContains(t, content, "description=\"Например: \\\"Нижнее\\\", \\\"Троллинг\\\", \\\"Переходный режим\\\"\"")
}

func TestPythonPydanticGenerator_BaseModel_ProtoConversion(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	f, err := gen.GenerateBaseModel(testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "class ProtoBaseModel(BaseModel)")
	assertContains(t, f.Content, "def from_proto")
	assertContains(t, f.Content, "def to_proto")
	assertContains(t, f.Content, "MessageToDict")
	assertContains(t, f.Content, "ParseDict")
}

func TestPythonPydanticGenerator_BaseModel_ExcludesBaseFieldsInProtoConversion(t *testing.T) {
	// Verify that ProtoBaseModel excludes base class fields (id, timestamps)
	// from to_proto / from_proto so that ParseDict does not try to set them
	// on the proto message where they don't exist.
	for _, ver := range []string{"2.0", "1.0"} {
		t.Run("v"+ver, func(t *testing.T) {
			gen := &PythonPydanticGenerator{version: ver}
			f, err := gen.GenerateBaseModel(testOpts())
			if err != nil {
				t.Fatal(err)
			}
			assertContains(t, f.Content, "_base_model_fields")
			assertContains(t, f.Content, `"id"`)
			assertContains(t, f.Content, `"created_at"`)
			assertContains(t, f.Content, `"updated_at"`)
			assertContains(t, f.Content, `"deleted_at"`)
			assertContains(t, f.Content, "exclude=self._base_model_fields")
			// from_proto should filter out base fields too
			assertContains(t, f.Content, "if k not in cls._base_model_fields")
		})
	}
}

func TestPythonPydanticGenerator_Model_WithExtendsImportFallback(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	m := testModel()
	m.Extends = "AuditableEntity"

	files, err := gen.GenerateModel(m, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "from .auditable_entity import AuditableEntity")
	assertContains(t, content, "AuditableEntity = ProtoBaseModel")
	assertContains(t, content, "class UserProfile(AuditableEntity)")
}

func TestPythonPydanticGenerator_Init(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	models := []ModelDef{testModel()}
	f, err := gen.GenerateInit(models, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "from .base_model import BaseModel")
	assertContains(t, f.Content, "ProtoBaseModel")
	assertContains(t, f.Content, "__all__")

	if !strings.Contains(f.Path, "__init__.py") {
		t.Errorf("expected __init__.py path, got %s", f.Path)
	}
}

// ══════════════════════════════════════════════════════════════════
//  Go generator tests
// ══════════════════════════════════════════════════════════════════

func TestGoNoneGenerator_BaseModel(t *testing.T) {
	gen := &GoNoneGenerator{}
	f, err := gen.GenerateBaseModel(testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "type BaseModel struct")
	assertContains(t, f.Content, "uuid.UUID")
	assertContains(t, f.Content, "func (m *BaseModel) FromProto")
	assertContains(t, f.Content, "func (m *BaseModel) ToProto")
}

func TestGoNoneGenerator_Model(t *testing.T) {
	gen := &GoNoneGenerator{}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "type UserProfile struct")
	assertContains(t, content, "Email string")
	assertContains(t, content, "`json:\"email\"`")
	assertContains(t, content, "Tags []string")
	assertContains(t, content, "func (m *UserProfile) FromProto")
	assertContains(t, content, "func (m *UserProfile) ToProto")
}

func TestGoGORMGenerator_Model(t *testing.T) {
	gen := &GoGORMGenerator{}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "type UserProfile struct")
	assertContains(t, content, "gorm:")
	assertContains(t, content, "TableName()")
	assertContains(t, content, "user_profiles")
	assertContains(t, content, "func (m *UserProfile) FromProto")
	assertContains(t, content, "func (m *UserProfile) ToProto")
}

func TestGoSQLXGenerator_Model(t *testing.T) {
	gen := &GoSQLXGenerator{}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "type UserProfile struct")
	assertContains(t, content, "`db:")
	assertContains(t, content, "func (m *UserProfile) FromProto")
	assertContains(t, content, "func (m *UserProfile) ToProto")
}

// ══════════════════════════════════════════════════════════════════
//  Rust generator tests
// ══════════════════════════════════════════════════════════════════

func TestRustNoneGenerator_BaseModel(t *testing.T) {
	gen := &RustNoneGenerator{}
	f, err := gen.GenerateBaseModel(testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "pub struct BaseModel")
	assertContains(t, f.Content, "Serialize")
	assertContains(t, f.Content, "Deserialize")
	assertContains(t, f.Content, "trait ProtoConvertible")
}

func TestRustNoneGenerator_Model(t *testing.T) {
	gen := &RustNoneGenerator{}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "pub struct UserProfile")
	assertContains(t, content, "pub email: String")
	assertContains(t, content, "impl ProtoConvertible for UserProfile")
}

func TestRustNoneGenerator_VisibilityMapping(t *testing.T) {
	gen := &RustNoneGenerator{}
	m := testModel()
	files, err := gen.GenerateModel(m, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	// password_hash is VisibilityPrivate → no 'pub' prefix
	if strings.Contains(content, "pub password_hash") {
		t.Error("private field should not have pub prefix")
	}
	// role is VisibilityInternal → pub(super)
	assertContains(t, content, "pub(super) role")
}

func TestRustDieselGenerator_Model(t *testing.T) {
	gen := &RustDieselGenerator{version: "2.0"}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "diesel(table_name")
	assertContains(t, content, "Queryable")
	assertContains(t, content, "pub struct NewUserProfile")
	assertContains(t, content, "impl ProtoConvertible for UserProfile")
}

func TestRustGenerator_Init(t *testing.T) {
	gen := &RustNoneGenerator{}
	models := []ModelDef{testModel()}
	f, err := gen.GenerateInit(models, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "pub mod base_model;")
	assertContains(t, f.Content, "pub mod user_profile;")
}

// ══════════════════════════════════════════════════════════════════
//  C++ generator tests
// ══════════════════════════════════════════════════════════════════

func TestCppNoneGenerator_BaseModel(t *testing.T) {
	gen := &CppNoneGenerator{}
	f, err := gen.GenerateBaseModel(testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "struct BaseModel")
	assertContains(t, f.Content, "#pragma once")
	assertContains(t, f.Content, "namespace buffalo::models")
	assertContains(t, f.Content, "proto_to_json")
	assertContains(t, f.Content, "json_to_proto")
}

func TestCppNoneGenerator_Model(t *testing.T) {
	gen := &CppNoneGenerator{}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "struct UserProfile")
	assertContains(t, content, "#pragma once")
	assertContains(t, content, "std::string email")
	assertContains(t, content, "to_json_obj() const override")
	assertContains(t, content, "from_json_obj(const nlohmann::json& j) override")
}

// ══════════════════════════════════════════════════════════════════
//  Helper type mapping tests
// ══════════════════════════════════════════════════════════════════

func TestProtoTypeToGo(t *testing.T) {
	cases := []struct {
		proto    string
		nullable bool
		want     string
	}{
		{"string", false, "string"},
		{"string", true, "*string"},
		{"int32", false, "int32"},
		{"int64", true, "*int64"},
		{"bool", false, "bool"},
		{"bytes", false, "[]byte"},
		{"bytes", true, "[]byte"}, // bytes is always pointer-like
		{"float", false, "float32"},
		{"double", true, "*float64"},
	}
	for _, tc := range cases {
		got := protoTypeToGo(tc.proto, tc.nullable)
		if got != tc.want {
			t.Errorf("protoTypeToGo(%q, %v) = %q, want %q", tc.proto, tc.nullable, got, tc.want)
		}
	}
}

func TestProtoTypeToPython(t *testing.T) {
	cases := []struct {
		proto    string
		nullable bool
		want     string
	}{
		{"string", false, "str"},
		{"string", true, "Optional[str]"},
		{"int32", false, "int"},
		{"bool", false, "bool"},
		{"float", false, "float"},
		{"bytes", false, "bytes"},
	}
	for _, tc := range cases {
		got := protoTypeToPython(tc.proto, tc.nullable)
		if got != tc.want {
			t.Errorf("protoTypeToPython(%q, %v) = %q, want %q", tc.proto, tc.nullable, got, tc.want)
		}
	}
}

func TestProtoTypeToRust(t *testing.T) {
	cases := []struct {
		proto    string
		nullable bool
		want     string
	}{
		{"string", false, "String"},
		{"string", true, "Option<String>"},
		{"int32", false, "i32"},
		{"uint64", false, "u64"},
		{"bytes", false, "Vec<u8>"},
	}
	for _, tc := range cases {
		got := protoTypeToRust(tc.proto, tc.nullable)
		if got != tc.want {
			t.Errorf("protoTypeToRust(%q, %v) = %q, want %q", tc.proto, tc.nullable, got, tc.want)
		}
	}
}

func TestProtoTypeToCpp(t *testing.T) {
	cases := []struct {
		proto    string
		nullable bool
		want     string
	}{
		{"string", false, "std::string"},
		{"string", true, "std::optional<std::string>"},
		{"int32", false, "int32_t"},
		{"bool", false, "bool"},
	}
	for _, tc := range cases {
		got := protoTypeToCpp(tc.proto, tc.nullable)
		if got != tc.want {
			t.Errorf("protoTypeToCpp(%q, %v) = %q, want %q", tc.proto, tc.nullable, got, tc.want)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  Helper string conversion tests
// ══════════════════════════════════════════════════════════════════

func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"UserProfile":  "user_profile",
		"HTTPHandler":  "http_handler",
		"GPSData":      "gps_data",
		"HTTPSHandler": "https_handler",
		"UserID":       "user_id",
		"simple":       "simple",
		"CamelCase":    "camel_case",
	}
	for input, want := range cases {
		got := toSnakeCase(input)
		if got != want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	cases := map[string]string{
		"user_profile": "UserProfile",
		"simple":       "Simple",
		"hello_world":  "HelloWorld",
	}
	for input, want := range cases {
		got := toPascalCase(input)
		if got != want {
			t.Errorf("toPascalCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	cases := map[string]string{
		"user_profile": "userProfile",
		"simple":       "simple",
	}
	for input, want := range cases {
		got := toCamelCase(input)
		if got != want {
			t.Errorf("toCamelCase(%q) = %q, want %q", input, got, want)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  CheckORMDependencies tests
// ══════════════════════════════════════════════════════════════════

func TestCheckORMDependencies(t *testing.T) {
	// None → no warnings
	w := CheckORMDependencies("python", "None")
	if len(w) != 0 {
		t.Errorf("expected no warnings for None, got %v", w)
	}

	// pydantic → hint
	w = CheckORMDependencies("python", "pydantic")
	if len(w) == 0 {
		t.Error("expected warning for pydantic")
	}

	// gorm → hint
	w = CheckORMDependencies("go", "gorm")
	if len(w) == 0 {
		t.Error("expected warning for gorm")
	}

	// diesel → hint
	w = CheckORMDependencies("rust", "diesel")
	if len(w) == 0 {
		t.Error("expected warning for diesel")
	}
}

// ══════════════════════════════════════════════════════════════════
//  Map / from-proto codegen tests
// ══════════════════════════════════════════════════════════════════

func TestCodegenMapField_AllLanguages(t *testing.T) {
	model := ModelDef{
		MessageName: "Settings",
		Package:     "test",
		Fields: []FieldDef{
			{Name: "source_id", ProtoType: "string", Number: 1},
			{Name: "settings", ProtoType: "map", Number: 2, IsMap: true, MapKeyType: "string", MapValueType: "string"},
			{Name: "tags", ProtoType: "string", Number: 3, Repeated: true},
		},
	}

	opts := GenerateOptions{
		Package: "test",
	}

	cases := []struct {
		lang     string
		orm      string
		contains []string
	}{
		{"python", "None", []string{"Dict[str, str]", "List[str]"}},
		{"python", "pydantic", []string{"Dict[str, str]", "List[str]"}},
		{"go", "None", []string{"map[string]string", "[]string"}},
		{"go", "gorm", []string{"map[string]string", "[]string"}},
		{"rust", "None", []string{"std::collections::HashMap<String, String>", "Vec<String>"}},
		{"cpp", "None", []string{"std::map<std::string, std::string>", "std::vector<std::string>"}},
	}

	for _, tc := range cases {
		t.Run(tc.lang+"/"+tc.orm, func(t *testing.T) {
			orm := ParseORMPlugin(tc.orm)
			gen, err := NewModelCodeGenerator(tc.lang, orm)
			if err != nil {
				t.Fatalf("NewModelCodeGenerator: %v", err)
			}

			opts.Language = tc.lang
			opts.ORM = orm
			files, err := gen.GenerateModel(model, opts)
			if err != nil {
				t.Fatalf("GenerateModel: %v", err)
			}
			if len(files) == 0 {
				t.Fatal("expected generated files")
			}

			content := files[0].Content
			for _, substr := range tc.contains {
				assertContains(t, content, substr)
			}
		})
	}
}

func TestCodegenFromProto_PlainMessage(t *testing.T) {
	// Simulate a plain proto message (no buffalo annotations)
	model := ModelDef{
		MessageName: "VideoOptions",
		Package:     "multimedia",
		Description: "Video configuration",
		Fields: []FieldDef{
			{Name: "resolution", ProtoType: "Resolution", Number: 1},
			{Name: "fps", ProtoType: "int32", Number: 2},
			{Name: "bitrate", ProtoType: "int32", Number: 3},
			{Name: "codec", ProtoType: "string", Number: 4},
		},
	}

	opts := GenerateOptions{
		Package: "multimedia",
	}

	// Test in all languages
	for _, lang := range []string{"python", "go", "rust", "cpp"} {
		t.Run(lang, func(t *testing.T) {
			gen, err := NewModelCodeGenerator(lang, ORMPlugin{Name: "None"})
			if err != nil {
				t.Fatalf("NewModelCodeGenerator: %v", err)
			}
			opts.Language = lang
			files, err := gen.GenerateModel(model, opts)
			if err != nil {
				t.Fatalf("GenerateModel: %v", err)
			}
			if len(files) == 0 {
				t.Fatal("expected generated files")
			}

			content := files[0].Content
			// All languages should contain the model name
			assertContains(t, content, "VideoOptions")
			// All should have fps and codec fields
			if lang == "python" {
				assertContains(t, content, "fps")
				assertContains(t, content, "codec")
			} else if lang == "go" {
				assertContains(t, content, "Fps")
				assertContains(t, content, "Codec")
			}
		})
	}
}

// ══════════════════════════════════════════════════════════════════
//  Helpers
// ══════════════════════════════════════════════════════════════════

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, truncate(content, 500))
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ══════════════════════════════════════════════════════════════════
//  Enum codegen tests
// ══════════════════════════════════════════════════════════════════

func testEnum() EnumDef {
	return EnumDef{
		Name:    "DeviceType",
		Comment: "Тип устройства",
		Values: []EnumValue{
			{Name: "DEVICE_TYPE_UNSPECIFIED", Number: 0, Comment: "Не указан"},
			{Name: "DEVICE_TYPE_CAMERA", Number: 1, Comment: "Камера"},
			{Name: "DEVICE_TYPE_SENSOR", Number: 2},
			{Name: "DEVICE_TYPE_GATEWAY", Number: 3},
		},
	}
}

func TestGenerateEnum_AllLanguages(t *testing.T) {
	enum := testEnum()
	opts := testOpts()

	cases := []struct {
		lang     string
		orm      string
		contains []string
	}{
		{"python", "pydantic", []string{"class DeviceType(int, Enum)", "DEVICE_TYPE_CAMERA = 1"}},
		{"python", "None", []string{"class DeviceType(int, Enum)", "DEVICE_TYPE_UNSPECIFIED = 0"}},
		{"go", "None", []string{"type DeviceType int32", "DeviceType_DEVICE_TYPE_CAMERA DeviceType = 1"}},
		{"go", "gorm", []string{"type DeviceType int32"}},
		{"rust", "None", []string{"pub enum DeviceType", "#[repr(i32)]"}},
		{"rust", "diesel", []string{"pub enum DeviceType"}},
		{"cpp", "None", []string{"enum class DeviceType : int32_t", "DEVICE_TYPE_CAMERA = 1"}},
	}

	for _, tc := range cases {
		t.Run(tc.lang+"/"+tc.orm, func(t *testing.T) {
			orm := ParseORMPlugin(tc.orm)
			gen, err := NewModelCodeGenerator(tc.lang, orm)
			if err != nil {
				t.Fatalf("NewModelCodeGenerator: %v", err)
			}
			opts.Language = tc.lang
			f, err := gen.GenerateEnum(enum, opts)
			if err != nil {
				t.Fatalf("GenerateEnum: %v", err)
			}
			for _, substr := range tc.contains {
				assertContains(t, f.Content, substr)
			}
		})
	}
}

func TestModelWithNestedEnums_Python(t *testing.T) {
	model := ModelDef{
		MessageName: "MultimediaSource",
		Package:     "test",
		Fields: []FieldDef{
			{Name: "source_id", ProtoType: "string", Number: 1},
			{Name: "status", ProtoType: "SourceStatus", Number: 2, IsEnum: true, EnumTypeName: "SourceStatus"},
		},
		Enums: []EnumDef{
			{
				Name: "SourceStatus",
				Values: []EnumValue{
					{Name: "SOURCE_STATUS_UNSPECIFIED", Number: 0},
					{Name: "SOURCE_STATUS_ACTIVE", Number: 1},
					{Name: "SOURCE_STATUS_INACTIVE", Number: 2},
				},
			},
		},
	}
	opts := testOpts()

	gen := &PythonPydanticGenerator{version: "2.0"}
	files, err := gen.GenerateModel(model, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "class SourceStatus(int, Enum)")
	assertContains(t, content, "SOURCE_STATUS_ACTIVE = 1")
	assertContains(t, content, "class MultimediaSource")
}

func TestModelWithOneofs_Python(t *testing.T) {
	model := ModelDef{
		MessageName: "Event",
		Package:     "test",
		Fields: []FieldDef{
			{Name: "id", ProtoType: "string", Number: 1},
			{Name: "text_data", ProtoType: "string", Number: 10, OneofGroup: "payload", Nullable: true},
			{Name: "numeric_data", ProtoType: "int64", Number: 11, OneofGroup: "payload", Nullable: true},
		},
		Oneofs: []OneofDef{
			{
				Name: "payload",
				Fields: []FieldDef{
					{Name: "text_data", ProtoType: "string", Number: 10},
					{Name: "numeric_data", ProtoType: "int64", Number: 11},
				},
			},
		},
	}
	opts := testOpts()

	gen := &PythonPydanticGenerator{version: "2.0"}
	files, err := gen.GenerateModel(model, opts)
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "PayloadType = Union[")
	assertContains(t, content, "class Event")
}
