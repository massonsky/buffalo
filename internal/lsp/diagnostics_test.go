package lsp

import (
	"strings"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

func TestSyntaxDiagnostics_ValidProto(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";

package test.example;

import "google/protobuf/empty.proto";

message User {
  string name = 1;
  int32 age = 2;
  repeated string tags = 3;
}

enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}

service UserService {
  rpc GetUser(User) returns (User);
  rpc DeleteUser(User) returns (User) {
    option deprecated = true;
  }
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	// Filter out informational cross-ref messages (type not found in file)
	errors := filterBySeverity(diagnostics, SeverityError)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid proto, got %d:", len(errors))
		for _, d := range errors {
			t.Logf("  [%s] %s (line %d)", d.Code, d.Message, d.Range.Start.Line+1)
		}
	}
}

func TestSyntaxDiagnostics_MissingSyntax(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `package test;

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeMissingSyntax)
}

func TestSyntaxDiagnostics_InvalidSyntaxVersion(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto4";
package test;

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidSyntax)
}

func TestSyntaxDiagnostics_DuplicateSyntax(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
syntax = "proto3";
package test;

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeDuplicateSyntax)
}

func TestSyntaxDiagnostics_DuplicatePackage(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;
package test2;

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeDuplicatePackage)
}

func TestSyntaxDiagnostics_MissingPackage(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeMissingPackage)
}

func TestSyntaxDiagnostics_MissingSemicolon(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "syntax missing semicolon",
			content: `syntax = "proto3"
package test;

message User {
  string name = 1;
}`,
		},
		{
			name: "package missing semicolon",
			content: `syntax = "proto3";
package test

message User {
  string name = 1;
}`,
		},
		{
			name: "import missing semicolon",
			content: `syntax = "proto3";
package test;
import "google/protobuf/empty.proto"

message User {
  string name = 1;
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument("file:///test.proto", tt.content)
			diagnostics := analyzer.SyntaxDiagnostics(doc)

			assertHasDiagnostic(t, diagnostics, DiagCodeMissingSemicolon)
		})
	}
}

func TestSyntaxDiagnostics_DuplicateFieldNumber(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
  int32 age = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeDuplicateFieldNumber)
}

func TestSyntaxDiagnostics_DuplicateFieldName(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
  int32 name = 2;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeDuplicateFieldName)
}

func TestSyntaxDiagnostics_InvalidFieldNumber(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "field number zero",
			content: `syntax = "proto3";
package test;

message User {
  string name = 0;
}`,
		},
		{
			name: "reserved field number",
			content: `syntax = "proto3";
package test;

message User {
  string name = 19500;
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument("file:///test.proto", tt.content)
			diagnostics := analyzer.SyntaxDiagnostics(doc)

			hasError := false
			for _, d := range diagnostics {
				if d.Severity == SeverityError &&
					(d.Code == DiagCodeInvalidFieldNumber || d.Code == DiagCodeReservedFieldNumber) {
					hasError = true
					break
				}
			}
			if !hasError {
				t.Errorf("Expected field number error diagnostic")
				for _, d := range diagnostics {
					t.Logf("  [%v] %s", d.Code, d.Message)
				}
			}
		})
	}
}

func TestSyntaxDiagnostics_DuplicateImport(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

import "google/protobuf/empty.proto";
import "google/protobuf/empty.proto";

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeDuplicateImport)
}

func TestSyntaxDiagnostics_InvalidImport(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;
import "";

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidImport)
}

func TestSyntaxDiagnostics_MismatchedBraces(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "unclosed message",
			content: `syntax = "proto3";
package test;

message User {
  string name = 1;
`,
		},
		{
			name: "extra closing brace",
			content: `syntax = "proto3";
package test;

message User {
  string name = 1;
}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument("file:///test.proto", tt.content)
			diagnostics := analyzer.SyntaxDiagnostics(doc)

			assertHasDiagnostic(t, diagnostics, DiagCodeMismatchedBraces)
		})
	}
}

func TestSyntaxDiagnostics_InvalidMapKeyType(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  map<float, string> items = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidMapKey)
}

func TestSyntaxDiagnostics_FieldAtTopLevel(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

string name = 1;
`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeTopLevelStatement)
}

func TestSyntaxDiagnostics_RPCAtTopLevel(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

rpc GetUser(UserRequest) returns (User);
`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeTopLevelStatement)
}

func TestSyntaxDiagnostics_InvalidRPC(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message Req {}
message Resp {}

service UserService {
  rpc GetUser(Req);
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidRPC)
}

func TestSyntaxDiagnostics_DuplicateEnumValue(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
  STATUS_DUPLICATE = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeDuplicateFieldNumber)
}

func TestSyntaxDiagnostics_EnumFirstValueNotZero(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

enum Status {
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidEnumDefault)
}

func TestSyntaxDiagnostics_InvalidStatementInEnum(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

enum Status {
  STATUS_UNKNOWN = 0;
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidSyntaxGeneral)
}

func TestSyntaxDiagnostics_FieldInService(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

service UserService {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidSyntaxGeneral)
}

func TestSyntaxDiagnostics_UnterminatedString(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3;
package test;

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeUnterminatedString)
}

func TestSyntaxDiagnostics_ReservedKeywordAsFieldName(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string import = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeReservedKeyword)
}

func TestSyntaxDiagnostics_OnlyOptionsInRPCBody(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message Req {}
message Resp {}

service UserService {
  rpc GetUser(Req) returns (Resp) {
    string invalid = 1;
  }
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	assertHasDiagnostic(t, diagnostics, DiagCodeInvalidSyntaxGeneral)
}

func TestSyntaxDiagnostics_NestedMessages(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message Outer {
  string name = 1;

  message Inner {
    int32 id = 1;
    string value = 2;
  }

  Inner inner = 2;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	errors := filterBySeverity(diagnostics, SeverityError)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid nested messages, got %d:", len(errors))
		for _, d := range errors {
			t.Logf("  [%v] %s (line %d)", d.Code, d.Message, d.Range.Start.Line+1)
		}
	}
}

func TestSyntaxDiagnostics_ValidMapField(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message Config {
  map<string, string> metadata = 1;
  map<int32, string> indices = 2;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	errors := filterBySeverity(diagnostics, SeverityError)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid map fields, got %d:", len(errors))
		for _, d := range errors {
			t.Logf("  [%v] %s (line %d)", d.Code, d.Message, d.Range.Start.Line+1)
		}
	}
}

func TestSyntaxDiagnostics_SyntaxOrderWarning(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `package test;
syntax = "proto3";

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	found := false
	for _, d := range diagnostics {
		if strings.Contains(d.Message, "should appear before") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning about syntax declaration order")
	}
}

func TestSyntaxDiagnostics_NonProtoFile(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	doc := NewDocument("file:///test.go", "package main")
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	if len(diagnostics) != 0 {
		t.Errorf("Expected 0 diagnostics for non-proto file, got %d", len(diagnostics))
	}
}

func TestSyntaxDiagnostics_ComplexService(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

import "buffalo/permissions/permissions.proto";

message GetUserRequest {
  string user_id = 1;
}

message User {
  string user_id = 1;
  string username = 2;
  string email = 3;
}

service UserService {
  option (buffalo.permissions.resource) = "users";

  rpc GetUser(GetUserRequest) returns (User) {
    option (buffalo.permissions.action) = "read";
    option (buffalo.permissions.roles) = ["admin", "user"];
  }
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.SyntaxDiagnostics(doc)

	errors := filterBySeverity(diagnostics, SeverityError)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid complex service, got %d:", len(errors))
		for _, d := range errors {
			t.Logf("  [%v] %s (line %d)", d.Code, d.Message, d.Range.Start.Line+1)
		}
	}
}

func TestSyntaxDiagnostics_IntegrationWithAnalyze(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
  int32 name = 2;
}`

	doc := NewDocument("file:///test.proto", content)
	diagnostics := analyzer.Analyze(doc)

	// Should include syntax diagnostics (duplicate field name) alongside existing checks
	found := false
	for _, d := range diagnostics {
		if d.Code == DiagCodeDuplicateFieldName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected Analyze() to include syntax diagnostics for duplicate field names")
		for _, d := range diagnostics {
			t.Logf("  [%v] source=%s %s", d.Code, d.Source, d.Message)
		}
	}
}

func TestStripInlineComment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`string name = 1; // comment`, `string name = 1;`},
		{`string name = 1;`, `string name = 1;`},
		{`import "path/to/file.proto"; // import`, `import "path/to/file.proto";`},
		{`string url = 1; // contains "quotes"`, `string url = 1;`},
		{`option foo = "bar // not a comment";`, `option foo = "bar // not a comment";`},
	}

	for _, tt := range tests {
		got := stripInlineComment(tt.input)
		if got != tt.want {
			t.Errorf("stripInlineComment(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCountBracesOutsideStrings(t *testing.T) {
	tests := []struct {
		input      string
		wantOpens  int
		wantCloses int
	}{
		{`message User {`, 1, 0},
		{`}`, 0, 1},
		{`option foo = "{}";`, 0, 0},
		{`message Foo { }`, 1, 1},
		{`rpc Get(Req) returns (Resp) {`, 1, 0},
	}

	for _, tt := range tests {
		opens, closes := countBracesOutsideStrings(tt.input)
		if opens != tt.wantOpens || closes != tt.wantCloses {
			t.Errorf("countBracesOutsideStrings(%q) = (%d, %d), want (%d, %d)",
				tt.input, opens, closes, tt.wantOpens, tt.wantCloses)
		}
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world foo bar", 10, "hello w..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		got := truncateStr(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// Test helpers

func filterBySeverity(diagnostics []Diagnostic, severity DiagnosticSeverity) []Diagnostic {
	var result []Diagnostic
	for _, d := range diagnostics {
		if d.Severity == severity {
			result = append(result, d)
		}
	}
	return result
}

func assertHasDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, d := range diagnostics {
		if d.Code == code {
			return
		}
	}
	t.Errorf("Expected diagnostic with code %s, but not found. Got %d diagnostics:", code, len(diagnostics))
	for _, d := range diagnostics {
		t.Logf("  [%v] %s: %s (line %d)", d.Code, d.Source, d.Message, d.Range.Start.Line+1)
	}
}
