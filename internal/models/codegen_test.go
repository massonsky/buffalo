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
		// Python
		{"python", "None", "None"},
		{"python", "pydantic", "pydantic"},
		{"python", "pydantic@2.0", "pydantic"},
		{"python", "sqlalchemy", "sqlalchemy"},
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
		{"python", "gorm"},
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

func TestPythonNoneGenerator_BaseModel(t *testing.T) {
	gen := &PythonNoneGenerator{}
	f, err := gen.GenerateBaseModel(testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "class BaseModel")
	assertContains(t, f.Content, "from dataclasses import")
	assertContains(t, f.Content, "def to_dict")
}

func TestPythonNoneGenerator_Model(t *testing.T) {
	gen := &PythonNoneGenerator{}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}
	content := files[0].Content
	assertContains(t, content, "class UserProfile")
	assertContains(t, content, "email: str")
	assertContains(t, content, "tags: List[str]")
}

func TestPythonPydanticGenerator_Model(t *testing.T) {
	gen := &PythonPydanticGenerator{version: "2.0"}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "class UserProfile")
	assertContains(t, content, "Field(")
}

func TestPythonSQLAlchemyGenerator_Model(t *testing.T) {
	gen := &PythonSQLAlchemyGenerator{version: "2.0"}
	files, err := gen.GenerateModel(testModel(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	content := files[0].Content
	assertContains(t, content, "class UserProfile")
	assertContains(t, content, "__tablename__")
	assertContains(t, content, "user_profiles")
}

func TestPythonGenerator_Init(t *testing.T) {
	gen := &PythonNoneGenerator{}
	models := []ModelDef{testModel()}
	f, err := gen.GenerateInit(models, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, f.Content, "from .base_model import BaseModel")
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
		"UserProfile": "user_profile",
		"HTTPHandler": "h_t_t_p_handler",
		"simple":      "simple",
		"CamelCase":   "camel_case",
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
