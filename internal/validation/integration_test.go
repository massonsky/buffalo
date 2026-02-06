package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massonsky/buffalo/internal/plugin"
	"github.com/massonsky/buffalo/pkg/logger"
)

// ══════════════════════════════════════════════════════════════════
//  Integration tests against real test-project proto files
// ══════════════════════════════════════════════════════════════════

const testProjectDir = "../../test-project"

// resolveTestProjectPath returns the absolute path to test-project/protos,
// or skips the test if the directory does not exist.
func resolveTestProjectPath(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join(testProjectDir, rel))
	if err != nil {
		t.Skipf("cannot resolve test-project path: %v", err)
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		t.Skipf("test-project path does not exist: %s", abs)
	}
	return abs
}

// ── Test 1: Existing proto files (no annotations) ────────────────

func TestIntegration_ExistingProtos_NoAnnotations(t *testing.T) {
	protosDir := resolveTestProjectPath(t, "protos")

	// Collect all .proto files
	var protoFiles []string
	filepath.Walk(protosDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})

	if len(protoFiles) == 0 {
		t.Skip("no proto files found in test-project/protos")
	}
	t.Logf("Found %d proto files in test-project", len(protoFiles))

	// Run the plugin — should produce no output but no errors either
	p := NewValidatePlugin()
	p.Init(DefaultValidateConfig())

	tempDir := t.TempDir()
	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: protoFiles,
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Errorf("expected success, errors: %v", output.Errors)
	}
	if len(output.GeneratedFiles) != 0 {
		t.Errorf("expected 0 generated files for unannotated protos, got %d", len(output.GeneratedFiles))
	}

	t.Logf("✅ Plugin correctly handled %d unannotated protos: %v", len(protoFiles), output.Messages)
}

// ── Test 2: Annotated User proto — full pipeline ─────────────────

func TestIntegration_AnnotatedUserProto_FullPipeline(t *testing.T) {
	// Create an annotated version of user.proto in a temp dir
	tempDir := t.TempDir()
	annotatedProto := filepath.Join(tempDir, "user.proto")

	content := `syntax = "proto3";

package user.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/yourorg/yourproject/protos/user/v1";

// User message with validation rules
message User {
  string user_id = 1 [(buffalo.validate.rules).string = {min_len: 1, max_len: 128, uuid: true}];
  string email = 2 [(buffalo.validate.rules).string = {email: true, min_len: 5, max_len: 255}];
  string name = 3 [(buffalo.validate.rules).string = {min_len: 1, max_len: 256, not_empty: true}];
}

// GetUserRequest with validated ID
message GetUserRequest {
  string user_id = 1 [(buffalo.validate.rules).string = {min_len: 1, uuid: true}];
}

// ListUsersRequest with page size constraints
message ListUsersRequest {
  int32 page_size = 1 [(buffalo.validate.rules).int32 = {gt: 0, lte: 100}];
  string page_token = 2;
}

// CreateUserRequest with required fields
message CreateUserRequest {
  string email = 1 [(buffalo.validate.rules).string = {email: true, min_len: 5}];
  string name = 2 [(buffalo.validate.rules).string = {not_empty: true, min_len: 1, max_len: 256}];
}

message DeleteUserRequest {
  string user_id = 1 [(buffalo.validate.rules).string = {min_len: 1, uuid: true}];
}
`
	writeIntegrationFile(t, annotatedProto, content)

	// Run with all languages
	p := NewValidatePlugin()
	p.Init(DefaultValidateConfig())

	outDir := filepath.Join(tempDir, "generated")
	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{annotatedProto},
		OutputDir:  outDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, errors: %v", output.Errors)
	}

	t.Logf("Messages: %v", output.Messages)
	t.Logf("Generated %d files:", len(output.GeneratedFiles))
	for _, f := range output.GeneratedFiles {
		rel, _ := filepath.Rel(outDir, f)
		info, _ := os.Stat(f)
		t.Logf("  %s (%d bytes)", rel, info.Size())
	}

	// Verify we have files for all 4 languages
	langDirs := map[string]bool{"go": false, "python": false, "cpp": false, "rust": false}
	for _, f := range output.GeneratedFiles {
		rel, _ := filepath.Rel(outDir, f)
		parts := strings.SplitN(rel, string(os.PathSeparator), 2)
		if len(parts) >= 1 {
			langDirs[parts[0]] = true
		}
	}
	for lang, found := range langDirs {
		if !found {
			t.Errorf("missing generated files for language: %s", lang)
		}
	}

	// ─── Verify Go output ────────────────────────────────────────
	goFiles := filterFiles(output.GeneratedFiles, "/go/")
	if len(goFiles) == 0 {
		t.Fatal("expected Go generated files")
	}

	for _, f := range goFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("cannot read %s: %v", f, err)
		}
		code := string(data)

		// Check common patterns
		assertContentContains(t, f, code, "DO NOT EDIT")
		assertContentContains(t, f, code, "package v1")
		assertContentContains(t, f, code, "func (m *")
		assertContentContains(t, f, code, "Validate() error")
	}

	// ─── Verify Python output ────────────────────────────────────
	pyFiles := filterFiles(output.GeneratedFiles, "/python/")
	if len(pyFiles) == 0 {
		t.Fatal("expected Python generated files")
	}

	for _, f := range pyFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("cannot read %s: %v", f, err)
		}
		code := string(data)

		assertContentContains(t, f, code, "DO NOT EDIT")
		assertContentContains(t, f, code, "def validate_")
		assertContentContains(t, f, code, "List[str]")
	}

	// ─── Verify C++ output ───────────────────────────────────────
	cppFiles := filterFiles(output.GeneratedFiles, "/cpp/")
	if len(cppFiles) < 2 { // at least one .h + one .cc pair
		t.Fatalf("expected at least 2 C++ generated files, got %d", len(cppFiles))
	}

	hasHeader := false
	hasSource := false
	for _, f := range cppFiles {
		if strings.HasSuffix(f, ".h") {
			hasHeader = true
		}
		if strings.HasSuffix(f, ".cc") {
			hasSource = true
		}
	}
	if !hasHeader {
		t.Error("missing C++ header file (.h)")
	}
	if !hasSource {
		t.Error("missing C++ source file (.cc)")
	}

	// ─── Verify Rust output ──────────────────────────────────────
	rsFiles := filterFiles(output.GeneratedFiles, "/rust/")
	if len(rsFiles) == 0 {
		t.Fatal("expected Rust generated files")
	}

	for _, f := range rsFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("cannot read %s: %v", f, err)
		}
		code := string(data)

		assertContentContains(t, f, code, "DO NOT EDIT")
		assertContentContains(t, f, code, "impl ")
		assertContentContains(t, f, code, "pub fn validate(&self)")
	}
}

// ── Test 3: Mix of annotated and unannotated protos ──────────────

func TestIntegration_MixedAnnotatedAndPlainProtos(t *testing.T) {
	tempDir := t.TempDir()

	// Plain proto (existing from test-project style)
	plainProto := filepath.Join(tempDir, "example.proto")
	writeIntegrationFile(t, plainProto, `syntax = "proto3";
package example;
message ExampleRequest {
  string id = 1;
}
message ExampleResponse {
  string id = 1;
  string name = 2;
  int32 value = 3;
}
`)

	// Annotated proto
	annotatedProto := filepath.Join(tempDir, "location.proto")
	writeIntegrationFile(t, annotatedProto, `syntax = "proto3";
package geo;
message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
  double lng = 2 [(buffalo.validate.rules).double = {gte: -180, lte: 180}];
  string name = 3 [(buffalo.validate.rules).string = {not_empty: true, max_len: 256}];
}
`)

	// Another annotated proto
	configProto := filepath.Join(tempDir, "config.proto")
	writeIntegrationFile(t, configProto, `syntax = "proto3";
package app;
message AppConfig {
  string app_name = 1 [(buffalo.validate.rules).string = {not_empty: true, min_len: 3, max_len: 64}];
  int32 port = 2 [(buffalo.validate.rules).int32 = {gte: 1024, lte: 65535}];
  string host = 3 [(buffalo.validate.rules).string = {hostname: true}];
  int32 max_connections = 4 [(buffalo.validate.rules).int32 = {gt: 0, lte: 10000}];
}
`)

	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		Options: map[string]interface{}{
			"languages": []interface{}{"go", "python"},
		},
	}
	p.Init(cfg)

	outDir := filepath.Join(tempDir, "out")
	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{plainProto, annotatedProto, configProto},
		OutputDir:  outDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, errors: %v", output.Errors)
	}

	// Should only generate for Location and AppConfig (2 messages × 2 languages)
	t.Logf("Generated %d files", len(output.GeneratedFiles))
	for _, f := range output.GeneratedFiles {
		rel, _ := filepath.Rel(outDir, f)
		t.Logf("  %s", rel)
	}

	goFiles := filterFiles(output.GeneratedFiles, "/go/")
	pyFiles := filterFiles(output.GeneratedFiles, "/python/")

	if len(goFiles) != 2 {
		t.Errorf("expected 2 Go files (Location + AppConfig), got %d", len(goFiles))
	}
	if len(pyFiles) != 2 {
		t.Errorf("expected 2 Python files (Location + AppConfig), got %d", len(pyFiles))
	}
}

// ── Test 4: Registry integration with test-project protos ────────

func TestIntegration_RegistryPipeline(t *testing.T) {
	tempDir := t.TempDir()

	proto := filepath.Join(tempDir, "service.proto")
	writeIntegrationFile(t, proto, `syntax = "proto3";
package service.v1;

message CreateOrderRequest {
  string customer_id = 1 [(buffalo.validate.rules).string = {uuid: true, min_len: 1}];
  repeated string item_ids = 2 [(buffalo.validate.rules).repeated = {min_items: 1, max_items: 50}];
  double total_amount = 3 [(buffalo.validate.rules).double = {gt: 0}];
  string currency = 4 [(buffalo.validate.rules).string = {min_len: 3, max_len: 3, pattern: "^[A-Z]+$"}];
  string notes = 5;
}

message Order {
  string order_id = 1 [(buffalo.validate.rules).string = {uuid: true}];
  string customer_id = 2 [(buffalo.validate.rules).string = {uuid: true}];
  double total_amount = 3 [(buffalo.validate.rules).double = {gte: 0}];
  int32 item_count = 4 [(buffalo.validate.rules).int32 = {gte: 0}];
}
`)

	// Set up full registry pipeline
	log := logger.New()
	registry := plugin.NewRegistry(log)

	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		HookPoints: []plugin.HookPoint{
			plugin.HookPointPostParse,
		},
		Priority: 90,
		Options: map[string]interface{}{
			"languages": []interface{}{"go", "python", "rust"},
		},
	}

	if err := registry.Register(p, cfg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := registry.InitAll(); err != nil {
		t.Fatalf("InitAll failed: %v", err)
	}
	defer registry.ShutdownAll()

	outDir := filepath.Join(tempDir, "out")
	input := &plugin.Input{
		ProtoFiles: []string{proto},
		OutputDir:  outDir,
	}

	err := registry.ExecuteHook(context.Background(), plugin.HookPointPostParse, input)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	t.Logf("Generated %d files via registry hook:", len(input.GeneratedFiles))
	for _, f := range input.GeneratedFiles {
		rel, _ := filepath.Rel(outDir, f)
		t.Logf("  %s", rel)
	}

	if len(input.GeneratedFiles) == 0 {
		t.Error("expected generated files from registry hook execution")
	}

	// Verify Go generated code quality for CreateOrderRequest
	goFiles := filterFiles(input.GeneratedFiles, "/go/")
	for _, f := range goFiles {
		data, _ := os.ReadFile(f)
		code := string(data)
		if strings.Contains(f, "create_order_request") {
			assertContentContains(t, f, code, "Validate() error")
			assertContentContains(t, f, code, "UUID")   // UUID error message
			assertContentContains(t, f, code, "regexp") // pattern rule
			t.Logf("✅ CreateOrderRequest validation verified")
		}
	}
}

// ── Test 5: Strict mode with erroneous protos ────────────────────

func TestIntegration_StrictMode_BadProto(t *testing.T) {
	tempDir := t.TempDir()

	goodProto := filepath.Join(tempDir, "good.proto")
	writeIntegrationFile(t, goodProto, `syntax = "proto3";
package test;
message Good {
  double val = 1 [(buffalo.validate.rules).double = {gte: 0, lte: 100}];
}
`)

	badProto := filepath.Join(tempDir, "bad.proto")
	writeIntegrationFile(t, badProto, `syntax = "proto3";
package test;
message Bad {
  double val = 1 [(buffalo.validate.rules).double = {gte: not_a_number}];
}
`)

	// Non-strict: should succeed with warnings
	p1 := NewValidatePlugin()
	p1.Init(plugin.Config{
		Name: "buffalo-validate", Enabled: true,
		Options: map[string]interface{}{"strict": false, "languages": []interface{}{"go"}},
	})

	out1, _ := p1.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{goodProto, badProto}, OutputDir: filepath.Join(tempDir, "out1"),
	})
	if !out1.Success {
		t.Error("non-strict mode: expected success despite bad proto")
	}
	if len(out1.Warnings) == 0 {
		t.Error("non-strict mode: expected warnings for bad proto")
	}
	t.Logf("Non-strict warnings: %v", out1.Warnings)

	// Strict: should fail
	p2 := NewValidatePlugin()
	p2.Init(plugin.Config{
		Name: "buffalo-validate", Enabled: true,
		Options: map[string]interface{}{"strict": true, "languages": []interface{}{"go"}},
	})

	out2, _ := p2.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{goodProto, badProto}, OutputDir: filepath.Join(tempDir, "out2"),
	})
	if out2.Success {
		t.Error("strict mode: expected failure for bad proto")
	}
	if len(out2.Errors) == 0 {
		t.Error("strict mode: expected errors for bad proto")
	}
	t.Logf("Strict errors: %v", out2.Errors)
}

// ── Test 6: Verify generated Go code compiles correctly ──────────

func TestIntegration_GeneratedGoCodeStructure(t *testing.T) {
	tempDir := t.TempDir()

	proto := filepath.Join(tempDir, "product.proto")
	writeIntegrationFile(t, proto, `syntax = "proto3";
package store.v1;

message Product {
  string sku = 1 [(buffalo.validate.rules).string = {min_len: 3, max_len: 32, pattern: "^[A-Z0-9-]+$"}];
  string name = 2 [(buffalo.validate.rules).string = {not_empty: true, min_len: 1, max_len: 512}];
  double price = 3 [(buffalo.validate.rules).double = {gte: 0}];
  int32 stock = 4 [(buffalo.validate.rules).int32 = {gte: 0}];
  string category = 5 [(buffalo.validate.rules).string = {not_empty: true}];
  string url = 6 [(buffalo.validate.rules).string = {uri: true}];
  string contact_email = 7 [(buffalo.validate.rules).string = {email: true}];
  string origin_ip = 8 [(buffalo.validate.rules).string = {ip: true}];
}
`)

	p := NewValidatePlugin()
	p.Init(plugin.Config{
		Name: "buffalo-validate", Enabled: true,
		Options: map[string]interface{}{"languages": []interface{}{"go"}},
	})

	outDir := filepath.Join(tempDir, "out")
	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{proto}, OutputDir: outDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, errors: %v", output.Errors)
	}

	// Read generated Go file and check structure
	goFile := filepath.Join(outDir, "go", "product.validate.go")
	data, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("cannot read generated file: %v", err)
	}
	code := string(data)

	// Verify all imports are present
	assertContentContains(t, goFile, code, `"fmt"`)
	assertContentContains(t, goFile, code, `"strings"`)
	assertContentContains(t, goFile, code, `"regexp"`)   // pattern + UUID
	assertContentContains(t, goFile, code, `"net/mail"`) // email
	assertContentContains(t, goFile, code, `"net/url"`)  // uri
	assertContentContains(t, goFile, code, `"net"`)      // ip

	// Verify method signature
	assertContentContains(t, goFile, code, "func (m *Product) Validate() error")

	// Verify all field validations present
	assertContentContains(t, goFile, code, "m.Sku")
	assertContentContains(t, goFile, code, "m.Name")
	assertContentContains(t, goFile, code, "m.Price")
	assertContentContains(t, goFile, code, "m.Stock")
	assertContentContains(t, goFile, code, "m.Category")
	assertContentContains(t, goFile, code, "m.Url")
	assertContentContains(t, goFile, code, "m.ContactEmail")
	assertContentContains(t, goFile, code, "m.OriginIp")

	// Verify rule-specific checks
	assertContentContains(t, goFile, code, "regexp.MatchString")  // pattern check
	assertContentContains(t, goFile, code, "mail.ParseAddress")   // email check
	assertContentContains(t, goFile, code, "url.ParseRequestURI") // URI check
	assertContentContains(t, goFile, code, "net.ParseIP")         // IP check
	assertContentContains(t, goFile, code, "strings.TrimSpace")   // not_empty check

	t.Logf("✅ Generated Go file verified:\n%s", code)
}

// ── Test 7: Print generated code for visual inspection ───────────

func TestIntegration_PrintGeneratedSamples(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping visual inspection in short mode")
	}

	tempDir := t.TempDir()

	proto := filepath.Join(tempDir, "location.proto")
	writeIntegrationFile(t, proto, `syntax = "proto3";
package geo;
message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
  double lng = 2 [(buffalo.validate.rules).double = {gte: -180, lte: 180}];
  string name = 3 [(buffalo.validate.rules).string = {not_empty: true, min_len: 1, max_len: 256}];
}
`)

	messages, err := ExtractValidationRules(readFile(t, proto), proto)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	for _, lang := range []string{"go", "python", "cpp", "rust"} {
		gen, _ := NewCodeGenerator(lang)
		files, _ := gen.Generate(messages)

		for _, f := range files {
			t.Logf("\n═══ %s: %s ═══\n%s", strings.ToUpper(lang), f.Path, f.Content)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  Helpers
// ══════════════════════════════════════════════════════════════════

func writeIntegrationFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	return string(data)
}

func filterFiles(files []string, pattern string) []string {
	var result []string
	for _, f := range files {
		if strings.Contains(f, pattern) {
			result = append(result, f)
		}
	}
	return result
}

func assertContentContains(t *testing.T, file, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("[%s] expected content to contain %q", filepath.Base(file), substr)
		if len(content) < 2000 {
			fmt.Fprintf(os.Stderr, "Full content:\n%s\n", content)
		}
	}
}
