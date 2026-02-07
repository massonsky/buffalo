package lsp

import (
	"strings"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

func TestProtoAnalyzer_Analyze(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name           string
		content        string
		wantDiagCount  int
		wantSeverities []DiagnosticSeverity
	}{
		{
			name: "valid proto3",
			content: `syntax = "proto3";

package test;

message User {
  string name = 1;
  int32 age = 2;
}`,
			wantDiagCount: 0,
		},
		{
			name: "missing syntax",
			content: `package test;

message User {
  string name = 1;
}`,
			wantDiagCount:  1,
			wantSeverities: []DiagnosticSeverity{SeverityWarning},
		},
		{
			name: "missing package",
			content: `syntax = "proto3";

message User {
  string name = 1;
}`,
			wantDiagCount:  1,
			wantSeverities: []DiagnosticSeverity{SeverityWarning},
		},
		{
			name: "invalid field number zero",
			content: `syntax = "proto3";
package test;

message User {
  string name = 0;
}`,
			wantDiagCount:  1,
			wantSeverities: []DiagnosticSeverity{SeverityError},
		},
		{
			name: "reserved field number range",
			content: `syntax = "proto3";
package test;

message User {
  string name = 19000;
}`,
			wantDiagCount:  1,
			wantSeverities: []DiagnosticSeverity{SeverityError},
		},
		{
			name: "snake_case hint",
			content: `syntax = "proto3";
package test;

message User {
  string userName = 1;
}`,
			wantDiagCount:  1,
			wantSeverities: []DiagnosticSeverity{SeverityHint},
		},
		{
			name: "mismatched braces",
			content: `syntax = "proto3";
package test;

message User {
  string name = 1;
`,
			wantDiagCount:  2,
			wantSeverities: []DiagnosticSeverity{SeverityError, SeverityError},
		},
		{
			name: "invalid syntax version",
			content: `syntax = "proto4";
package test;

message User {
  string name = 1;
}`,
			wantDiagCount:  1,
			wantSeverities: []DiagnosticSeverity{SeverityError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument("file:///test.proto", tt.content)
			diagnostics := analyzer.Analyze(doc)

			if len(diagnostics) != tt.wantDiagCount {
				t.Errorf("Analyze() got %d diagnostics, want %d", len(diagnostics), tt.wantDiagCount)
				for _, d := range diagnostics {
					t.Logf("  - %d: %s", d.Severity, d.Message)
				}
			}

			if len(tt.wantSeverities) > 0 && len(diagnostics) > 0 {
				for i, severity := range tt.wantSeverities {
					if i < len(diagnostics) && diagnostics[i].Severity != severity {
						t.Errorf("Diagnostic[%d] severity = %v, want %v", i, diagnostics[i].Severity, severity)
					}
				}
			}
		})
	}
}

func TestProtoAnalyzer_Complete(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name      string
		content   string
		line      int
		character int
		wantItems []string
	}{
		{
			name:      "top level",
			content:   "",
			line:      0,
			character: 0,
			wantItems: []string{"syntax", "package", "message", "service", "enum"},
		},
		{
			name: "inside message",
			content: `syntax = "proto3";
package test;

message User {
  
}`,
			line:      4,
			character: 2,
			wantItems: []string{"string", "int32", "repeated"},
		},
		{
			name: "inside service",
			content: `syntax = "proto3";
package test;

service UserService {
  
}`,
			line:      4,
			character: 2,
			wantItems: []string{"rpc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument("file:///test.proto", tt.content)
			pos := Position{Line: tt.line, Character: tt.character}
			items := analyzer.Complete(doc, pos, nil)

			// Check that expected items are present
			for _, want := range tt.wantItems {
				found := false
				for _, item := range items {
					if item.Label == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Complete() missing expected item %q", want)
				}
			}
		})
	}
}

func TestProtoAnalyzer_Hover(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name      string
		content   string
		line      int
		character int
		wantHover bool
		wantKind  MarkupKind
	}{
		{
			name: "hover on string type",
			content: `syntax = "proto3";
package test;

message User {
  string name = 1;
}`,
			line:      4,
			character: 4,
			wantHover: true,
			wantKind:  MarkupKindMarkdown,
		},
		{
			name: "hover on message keyword",
			content: `syntax = "proto3";
package test;

message User {
  string name = 1;
}`,
			line:      3,
			character: 2,
			wantHover: true,
			wantKind:  MarkupKindMarkdown,
		},
		{
			name: "hover on message name",
			content: `syntax = "proto3";
package test;

message User {
  string name = 1;
}`,
			line:      3,
			character: 10,
			wantHover: true,
			wantKind:  MarkupKindMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocument("file:///test.proto", tt.content)
			pos := Position{Line: tt.line, Character: tt.character}
			hover := analyzer.Hover(doc, pos)

			if tt.wantHover {
				if hover == nil {
					t.Error("Hover() returned nil, want hover info")
				} else if hover.Contents.Kind != tt.wantKind {
					t.Errorf("Hover().Contents.Kind = %v, want %v", hover.Contents.Kind, tt.wantKind)
				}
			} else if hover != nil {
				t.Errorf("Hover() returned info, want nil")
			}
		})
	}
}

func TestProtoAnalyzer_Definition(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
}

message Request {
  User user = 1;
}`
	doc := NewDocument("file:///test.proto", content)

	// Find definition of User reference
	pos := Position{Line: 8, Character: 4} // "User" in Request
	loc := analyzer.Definition(doc, pos)

	if loc == nil {
		t.Error("Definition() returned nil, want location")
	} else {
		if loc.Range.Start.Line != 3 {
			t.Errorf("Definition location line = %d, want 3", loc.Range.Start.Line)
		}
	}
}

func TestProtoAnalyzer_DocumentSymbols(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
  int32 age = 2;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (User);
}`

	doc := NewDocument("file:///test.proto", content)
	symbols := analyzer.DocumentSymbols(doc)

	// Should have User message and UserService
	if len(symbols) < 2 {
		t.Errorf("DocumentSymbols() returned %d symbols, want at least 2", len(symbols))
	}

	// Check User message
	foundUser := false
	for _, sym := range symbols {
		if sym.Name == "User" {
			foundUser = true
			if sym.Kind != SymbolKindStruct {
				t.Errorf("User symbol kind = %v, want Struct", sym.Kind)
			}
		}
	}
	if !foundUser {
		t.Error("DocumentSymbols() missing User symbol")
	}
}

func TestProtoAnalyzer_Format(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;
message User {
string name = 1;
  int32 age = 2;
}`

	doc := NewDocument("file:///test.proto", content)
	edits := analyzer.Format(doc, FormattingOptions{
		TabSize:      2,
		InsertSpaces: true,
	})

	// Should have edits to fix indentation
	if len(edits) == 0 {
		t.Error("Format() returned no edits, expected indentation fixes")
	}
}

func TestProtoAnalyzer_FoldingRanges(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
  int32 age = 2;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (User);
}`

	doc := NewDocument("file:///test.proto", content)
	ranges := analyzer.FoldingRanges(doc)

	// Should have folding ranges for User and UserService
	if len(ranges) < 2 {
		t.Errorf("FoldingRanges() returned %d ranges, want at least 2", len(ranges))
	}
}

func TestDocument_GetWordAtPosition(t *testing.T) {
	content := `syntax = "proto3";
package test;

message User {
  string name = 1;
}`

	doc := NewDocument("file:///test.proto", content)

	tests := []struct {
		line      int
		character int
		want      string
	}{
		{0, 2, "syntax"},
		{3, 10, "User"},
		{4, 4, "string"},
		{4, 12, "name"},
	}

	for _, tt := range tests {
		pos := Position{Line: tt.line, Character: tt.character}
		got := doc.getWordAtPosition(pos)
		if got != tt.want {
			t.Errorf("getWordAtPosition(%d, %d) = %q, want %q", tt.line, tt.character, got, tt.want)
		}
	}
}

func TestNamingConventions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isSnake  bool
		isPascal bool
		isUpper  bool
	}{
		{"snake_case", "user_name", true, false, false},
		{"camelCase", "userName", false, false, false},
		{"PascalCase", "UserName", false, true, false},
		{"UPPER_SNAKE", "USER_NAME", false, true, true},
		{"lowercase", "username", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSnakeCase(tt.input); got != tt.isSnake {
				t.Errorf("isSnakeCase(%q) = %v, want %v", tt.input, got, tt.isSnake)
			}
			if got := isPascalCase(tt.input); got != tt.isPascal {
				t.Errorf("isPascalCase(%q) = %v, want %v", tt.input, got, tt.isPascal)
			}
			if got := isUpperSnakeCase(tt.input); got != tt.isUpper {
				t.Errorf("isUpperSnakeCase(%q) = %v, want %v", tt.input, got, tt.isUpper)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"userName", "user_name"},
		{"UserName", "user_name"},
		{"getUserByID", "get_user_by_i_d"},
		{"name", "name"},
	}

	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCompletions(t *testing.T) {
	// Test that completion functions return non-empty lists
	if items := topLevelCompletions(); len(items) == 0 {
		t.Error("topLevelCompletions() returned empty list")
	}

	if items := fieldTypeCompletions(); len(items) == 0 {
		t.Error("fieldTypeCompletions() returned empty list")
	}

	if items := serviceCompletions(); len(items) == 0 {
		t.Error("serviceCompletions() returned empty list")
	}

	if items := buffaloAnnotationCompletions(); len(items) == 0 {
		t.Error("buffaloAnnotationCompletions() returned empty list")
	}

	if items := commonImportCompletions(); len(items) == 0 {
		t.Error("commonImportCompletions() returned empty list")
	}
}

func TestScalarTypeDoc(t *testing.T) {
	scalarTypes := []string{"double", "float", "int32", "int64", "uint32", "uint64", "sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64", "bool", "string", "bytes"}

	for _, typeName := range scalarTypes {
		doc := getScalarTypeDoc(typeName)
		if doc == "" {
			t.Errorf("getScalarTypeDoc(%q) returned empty string", typeName)
		}
	}
}

func TestKeywordDoc(t *testing.T) {
	keywords := []string{"syntax", "package", "import", "message", "service", "enum", "rpc", "repeated", "optional", "oneof", "map"}

	for _, kw := range keywords {
		doc := getKeywordDoc(kw)
		if doc == "" {
			t.Errorf("getKeywordDoc(%q) returned empty string", kw)
		}
	}
}

func TestAnalyze_PermissionImportVariants(t *testing.T) {
	log := logger.New()
	analyzer := NewProtoAnalyzer(log)

	tests := []struct {
		name       string
		importLine string
		wantWarn   bool // expect "without import" warning
	}{
		{
			name:       "buffalo/permissions/permissions.proto",
			importLine: `import "buffalo/permissions/permissions.proto";`,
			wantWarn:   false,
		},
		{
			name:       "buffalo/permissions.proto",
			importLine: `import "buffalo/permissions.proto";`,
			wantWarn:   false,
		},
		{
			name:       "no import",
			importLine: ``,
			wantWarn:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `syntax = "proto3";
package test;
` + tt.importLine + `

service UserService {
  option (buffalo.permissions.resource) = "users";
  rpc GetUser(GetUserRequest) returns (User) {
    option (buffalo.permissions.method_perms) = {
      action: "read"
    };
  }
}

message GetUserRequest { string id = 1; }
message User { string id = 1; }
`
			doc := NewDocument("file:///test.proto", content)
			diagnostics := analyzer.Analyze(doc)

			hasImportWarn := false
			for _, d := range diagnostics {
				if d.Severity == SeverityWarning &&
					(strings.Contains(d.Message, "without import") || strings.Contains(d.Message, "not imported")) {
					hasImportWarn = true
					break
				}
			}

			if tt.wantWarn && !hasImportWarn {
				t.Errorf("expected 'without import' warning, but none found")
				for _, d := range diagnostics {
					t.Logf("  %d: %s", d.Severity, d.Message)
				}
			}
			if !tt.wantWarn && hasImportWarn {
				t.Errorf("did NOT expect 'without import' warning, but found one")
				for _, d := range diagnostics {
					if strings.Contains(d.Message, "without import") || strings.Contains(d.Message, "not imported") {
						t.Logf("  %d: %s", d.Severity, d.Message)
					}
				}
			}
		})
	}
}
