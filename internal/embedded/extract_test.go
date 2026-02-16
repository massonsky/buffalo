package embedded

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProtoFS_Contains_ValidateProto(t *testing.T) {
	data, err := ProtoFS.ReadFile("proto/buffalo/validate/validate.proto")
	if err != nil {
		t.Fatalf("failed to read embedded validate.proto: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded validate.proto is empty")
	}
	content := string(data)
	if !strings.Contains(content, "package buffalo.validate") {
		t.Error("expected embedded proto to contain 'package buffalo.validate'")
	}
	if !strings.Contains(content, "FieldRules") {
		t.Error("expected embedded proto to contain 'FieldRules'")
	}
}

func TestProtoFS_Contains_ModelsProto(t *testing.T) {
	data, err := ProtoFS.ReadFile("proto/buffalo/models/models.proto")
	if err != nil {
		t.Fatalf("failed to read embedded models.proto: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded models.proto is empty")
	}
	content := string(data)
	if !strings.Contains(content, "package buffalo.models") {
		t.Error("expected embedded proto to contain 'package buffalo.models'")
	}
	if !strings.Contains(content, "ModelOptions") {
		t.Error("expected embedded proto to contain 'ModelOptions'")
	}
}

func TestListEmbeddedProtos(t *testing.T) {
	files, err := ListEmbeddedProtos()
	if err != nil {
		t.Fatalf("ListEmbeddedProtos failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one embedded proto file")
	}

	found := false
	for _, f := range files {
		if strings.Contains(f, "validate.proto") {
			found = true
		}
	}
	if !found {
		t.Errorf("validate.proto not found in embedded protos: %v", files)
	}
}

func TestExtractAllProtos(t *testing.T) {
	tmpDir := t.TempDir()

	protoPath, err := ExtractAllProtos(tmpDir)
	if err != nil {
		t.Fatalf("ExtractAllProtos failed: %v", err)
	}

	expectedProtoPath := filepath.Join(tmpDir, "proto")
	if protoPath != expectedProtoPath {
		t.Errorf("expected protoPath %q, got %q", expectedProtoPath, protoPath)
	}

	// Verify the file was extracted
	validatePath := filepath.Join(tmpDir, "proto", "buffalo", "validate", "validate.proto")
	info, err := os.Stat(validatePath)
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("extracted file is empty")
	}

	// Verify content
	data, err := os.ReadFile(validatePath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if !strings.Contains(string(data), "package buffalo.validate") {
		t.Error("extracted file content mismatch")
	}
}

func TestExtractAllProtos_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Extract twice — should not error
	_, err := ExtractAllProtos(tmpDir)
	if err != nil {
		t.Fatalf("first extract failed: %v", err)
	}

	_, err = ExtractAllProtos(tmpDir)
	if err != nil {
		t.Fatalf("second extract (overwrite) failed: %v", err)
	}
}

func TestValidateProtoImportPath_AutoExtract(t *testing.T) {
	tmpDir := t.TempDir()

	// File doesn't exist yet → auto-extract
	importPath, err := ValidateProtoImportPath(tmpDir)
	if err != nil {
		t.Fatalf("ValidateProtoImportPath failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "proto")
	if importPath != expectedPath {
		t.Errorf("expected import path %q, got %q", expectedPath, importPath)
	}

	// File now exists → no re-extract needed
	importPath2, err := ValidateProtoImportPath(tmpDir)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if importPath2 != expectedPath {
		t.Errorf("expected import path %q on second call, got %q", expectedPath, importPath2)
	}
}

func TestExtractValidateProto(t *testing.T) {
	tmpDir := t.TempDir()

	protoPath, err := ExtractValidateProto(tmpDir)
	if err != nil {
		t.Fatalf("ExtractValidateProto failed: %v", err)
	}

	// Verify directory structure
	expectedFile := filepath.Join(tmpDir, "proto", "buffalo", "validate", "validate.proto")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("validate.proto not extracted to expected path: %s", expectedFile)
	}

	// protoPath should be the directory to pass to protoc --proto_path
	if !strings.HasSuffix(protoPath, "proto") {
		t.Errorf("protoPath should end with 'proto', got %q", protoPath)
	}
}
