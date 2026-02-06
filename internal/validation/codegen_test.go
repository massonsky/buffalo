package validation

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════
//  NewCodeGenerator tests
// ══════════════════════════════════════════════════════════════════

func TestNewCodeGenerator_SupportedLanguages(t *testing.T) {
	for _, lang := range []string{"go", "python", "cpp", "rust"} {
		gen, err := NewCodeGenerator(lang)
		if err != nil {
			t.Errorf("NewCodeGenerator(%q) returned error: %v", lang, err)
		}
		if gen == nil {
			t.Errorf("NewCodeGenerator(%q) returned nil", lang)
		}
		if gen.Language() != lang {
			t.Errorf("expected language %q, got %q", lang, gen.Language())
		}
	}
}

func TestNewCodeGenerator_UnsupportedLanguage(t *testing.T) {
	_, err := NewCodeGenerator("java")
	if err == nil {
		t.Error("expected error for unsupported language, got nil")
	}
}

// ══════════════════════════════════════════════════════════════════
//  Location message — canonical test fixture
// ══════════════════════════════════════════════════════════════════

func locationMessageRules() []MessageRules {
	return []MessageRules{
		{
			MessageName: "Location",
			Package:     "geo",
			Fields: map[string][]FieldRule{
				"lat": {
					{Type: RuleGte, Value: float64(-90), FieldName: "lat", FieldType: "double"},
					{Type: RuleLte, Value: float64(90), FieldName: "lat", FieldType: "double"},
				},
				"lng": {
					{Type: RuleGte, Value: float64(-180), FieldName: "lng", FieldType: "double"},
					{Type: RuleLte, Value: float64(180), FieldName: "lng", FieldType: "double"},
				},
			},
		},
	}
}

// ══════════════════════════════════════════════════════════════════
//  Go code generator
// ══════════════════════════════════════════════════════════════════

func TestGoCodeGenerator_Location(t *testing.T) {
	gen := &GoCodeGenerator{}
	files, err := gen.Generate(locationMessageRules())
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "location.validate.go" {
		t.Errorf("expected path 'location.validate.go', got '%s'", files[0].Path)
	}

	content := files[0].Content
	assertContains(t, content, "func (m *Location) Validate() error")
	assertContains(t, content, "m.Lat < -90")
	assertContains(t, content, "m.Lat > 90")
	assertContains(t, content, "m.Lng < -180")
	assertContains(t, content, "m.Lng > 180")
	assertContains(t, content, "package geo")
	assertContains(t, content, "DO NOT EDIT")
}

func TestGoCodeGenerator_StringRules(t *testing.T) {
	gen := &GoCodeGenerator{}
	messages := []MessageRules{
		{
			MessageName: "User",
			Package:     "api",
			Fields: map[string][]FieldRule{
				"email": {
					{Type: RuleEmail, Value: true, FieldName: "email", FieldType: "string"},
					{Type: RuleMinLen, Value: uint64(5), FieldName: "email", FieldType: "string"},
				},
				"name": {
					{Type: RuleNotEmpty, Value: true, FieldName: "name", FieldType: "string"},
				},
			},
		},
	}

	files, err := gen.Generate(messages)
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := files[0].Content
	assertContains(t, content, "mail.ParseAddress")
	assertContains(t, content, "len([]rune(m.Email)) < 5")
	assertContains(t, content, "strings.TrimSpace(m.Name)")
}

func TestGoCodeGenerator_DisabledMessage(t *testing.T) {
	gen := &GoCodeGenerator{}
	messages := []MessageRules{
		{
			MessageName: "Foo",
			Package:     "test",
			Disabled:    true,
			Fields:      map[string][]FieldRule{"x": {{Type: RuleGte, Value: float64(0)}}},
		},
	}
	files, err := gen.Generate(messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files for disabled message, got %d", len(files))
	}
}

func TestGoCodeGenerator_Imports(t *testing.T) {
	gen := &GoCodeGenerator{}
	messages := []MessageRules{
		{
			MessageName: "Mix",
			Package:     "test",
			Fields: map[string][]FieldRule{
				"url":   {{Type: RuleURI, Value: true, FieldName: "url", FieldType: "string"}},
				"email": {{Type: RuleEmail, Value: true, FieldName: "email", FieldType: "string"}},
				"id":    {{Type: RuleUUID, Value: true, FieldName: "id", FieldType: "string"}},
				"ip":    {{Type: RuleIP, Value: true, FieldName: "ip", FieldType: "string"}},
			},
		},
	}
	files, err := gen.Generate(messages)
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}

	content := files[0].Content
	assertContains(t, content, `"net/url"`)
	assertContains(t, content, `"net/mail"`)
	assertContains(t, content, `"regexp"`)
	assertContains(t, content, `"net"`)
}

// ══════════════════════════════════════════════════════════════════
//  Python code generator
// ══════════════════════════════════════════════════════════════════

func TestPythonCodeGenerator_Location(t *testing.T) {
	gen := &PythonCodeGenerator{}
	files, err := gen.Generate(locationMessageRules())
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "location_validate.py" {
		t.Errorf("expected path 'location_validate.py', got '%s'", files[0].Path)
	}

	content := files[0].Content
	assertContains(t, content, "def validate_location(msg)")
	assertContains(t, content, "msg.lat < -90")
	assertContains(t, content, "msg.lat > 90")
	assertContains(t, content, "msg.lng < -180")
	assertContains(t, content, "msg.lng > 180")
	assertContains(t, content, "DO NOT EDIT")
}

func TestPythonCodeGenerator_StringRules(t *testing.T) {
	gen := &PythonCodeGenerator{}
	messages := []MessageRules{
		{
			MessageName: "User",
			Package:     "api",
			Fields: map[string][]FieldRule{
				"email": {
					{Type: RuleEmail, Value: true, FieldName: "email", FieldType: "string"},
				},
				"name": {
					{Type: RuleNotEmpty, Value: true, FieldName: "name", FieldType: "string"},
					{Type: RuleMinLen, Value: uint64(1), FieldName: "name", FieldType: "string"},
				},
			},
		},
	}
	files, err := gen.Generate(messages)
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}

	content := files[0].Content
	assertContains(t, content, "def validate_user(msg)")
	assertContains(t, content, "re.match")
	assertContains(t, content, "msg.name.strip()")
}

// ══════════════════════════════════════════════════════════════════
//  C++ code generator
// ══════════════════════════════════════════════════════════════════

func TestCppCodeGenerator_Location(t *testing.T) {
	gen := &CppCodeGenerator{}
	files, err := gen.Generate(locationMessageRules())
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}
	// Header + source
	if len(files) != 2 {
		t.Fatalf("expected 2 files (header + source), got %d", len(files))
	}

	header := files[0]
	source := files[1]

	if header.Path != "location.validate.h" {
		t.Errorf("expected header path 'location.validate.h', got '%s'", header.Path)
	}
	if source.Path != "location.validate.cc" {
		t.Errorf("expected source path 'location.validate.cc', got '%s'", source.Path)
	}

	assertContains(t, header.Content, "LOCATION_VALIDATE_H_")
	assertContains(t, header.Content, "ValidateLocation")
	assertContains(t, source.Content, "msg.lat() < -90")
	assertContains(t, source.Content, "msg.lat() > 90")
	assertContains(t, source.Content, "msg.lng() < -180")
	assertContains(t, source.Content, "msg.lng() > 180")
}

// ══════════════════════════════════════════════════════════════════
//  Rust code generator
// ══════════════════════════════════════════════════════════════════

func TestRustCodeGenerator_Location(t *testing.T) {
	gen := &RustCodeGenerator{}
	files, err := gen.Generate(locationMessageRules())
	if err != nil {
		t.Fatalf("codegen failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "location_validate.rs" {
		t.Errorf("expected path 'location_validate.rs', got '%s'", files[0].Path)
	}

	content := files[0].Content
	assertContains(t, content, "impl Location")
	assertContains(t, content, "pub fn validate(&self)")
	assertContains(t, content, "self.lat < -90")
	assertContains(t, content, "self.lat > 90")
	assertContains(t, content, "self.lng < -180")
	assertContains(t, content, "self.lng > 180")
}

// ══════════════════════════════════════════════════════════════════
//  Helper utilities
// ══════════════════════════════════════════════════════════════════

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"Location", "location"},
		{"UserService", "user_service"},
		{"HTTPClient", "h_t_t_p_client"},
		{"simple", "simple"},
		{"A", "a"},
		{"", ""},
	}
	for _, tc := range tests {
		got := toSnakeCase(tc.input)
		if got != tc.expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"lat", "Lat"},
		{"user_name", "UserName"},
		{"id", "Id"},
		{"", ""},
		{"a_b_c", "ABC"},
	}
	for _, tc := range tests {
		got := toPascalCase(tc.input)
		if got != tc.expected {
			t.Errorf("toPascalCase(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestGoPackageName(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"geo", "geo"},
		{"my.company.api.v1", "v1"},
		{"a.b", "b"},
	}
	for _, tc := range tests {
		got := goPackageName(tc.input)
		if got != tc.expected {
			t.Errorf("goPackageName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  End-to-end: parse proto → generate all languages
// ══════════════════════════════════════════════════════════════════

func TestEndToEnd_ParseAndGenerateAllLanguages(t *testing.T) {
	proto := `syntax = "proto3";

package geo;

import "buffalo/validate/validate.proto";

message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
  double lng = 2 [(buffalo.validate.rules).double = {gte: -180, lte: 180}];
}
`
	messages, err := ExtractValidationRules(proto, "location.proto")
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected at least 1 message")
	}

	for _, lang := range []string{"go", "python", "cpp", "rust"} {
		gen, err := NewCodeGenerator(lang)
		if err != nil {
			t.Fatalf("NewCodeGenerator(%q) failed: %v", lang, err)
		}

		files, err := gen.Generate(messages)
		if err != nil {
			t.Fatalf("generate(%q) failed: %v", lang, err)
		}
		if len(files) == 0 {
			t.Errorf("expected files for %s, got 0", lang)
		}

		for _, f := range files {
			if f.Content == "" {
				t.Errorf("generated file %s (%s) has empty content", f.Path, lang)
			}
			if !strings.Contains(f.Content, "DO NOT EDIT") {
				t.Errorf("generated file %s (%s) missing 'DO NOT EDIT' header", f.Path, lang)
			}
		}
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected string to contain %q, but it didn't.\nFull string:\n%s", substr, s)
	}
}
