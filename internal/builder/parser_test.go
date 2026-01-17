package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

func TestProtoParser_ParseFile(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	parser := NewProtoParser(logAdapter)
	ctx := context.Background()

	// Create a test proto file
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	protoContent := `syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 value = 2;
}

service TestService {
  rpc GetTest(TestMessage) returns (TestMessage);
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	// Test successful parse
	t.Run("Success", func(t *testing.T) {
		result, err := parser.ParseFile(ctx, protoFile, []string{tempDir})
		if err != nil {
			t.Fatalf("ParseFile failed: %v", err)
		}

		if result.Path != protoFile {
			t.Errorf("Expected path %s, got %s", protoFile, result.Path)
		}

		if result.Package != "test" {
			t.Errorf("Expected package 'test', got '%s'", result.Package)
		}

		if result.Syntax != "proto3" {
			t.Errorf("Expected syntax 'proto3', got '%s'", result.Syntax)
		}

		if len(result.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(result.Messages))
		}

		if len(result.Services) != 1 {
			t.Errorf("Expected 1 service, got %d", len(result.Services))
		}
	})

	// Test non-existent file
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := parser.ParseFile(ctx, "nonexistent.proto", nil)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

func TestProtoParser_ParseFiles(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	parser := NewProtoParser(logAdapter)
	ctx := context.Background()

	// Create multiple test proto files
	tempDir := t.TempDir()

	files := []string{
		filepath.Join(tempDir, "file1.proto"),
		filepath.Join(tempDir, "file2.proto"),
	}

	for i, file := range files {
		content := `syntax = "proto3";
package test` + string(rune('1'+i)) + `;
message TestMessage` + string(rune('1'+i)) + ` {
  string name = 1;
}
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test successful parse
	t.Run("Success", func(t *testing.T) {
		results, err := parser.ParseFiles(ctx, files, []string{tempDir})
		if err != nil {
			t.Fatalf("ParseFiles failed: %v", err)
		}

		if len(results) != len(files) {
			t.Errorf("Expected %d results, got %d", len(files), len(results))
		}

		for i, result := range results {
			if result.Path != files[i] {
				t.Errorf("Expected path %s, got %s", files[i], result.Path)
			}
		}
	})

	// Test with invalid file
	t.Run("InvalidFile", func(t *testing.T) {
		invalidFiles := append(files, "nonexistent.proto")
		_, err := parser.ParseFiles(ctx, invalidFiles, []string{tempDir})
		if err == nil {
			t.Error("Expected error for invalid file, got nil")
		}
	})

	// Test with empty list
	t.Run("EmptyList", func(t *testing.T) {
		results, err := parser.ParseFiles(ctx, []string{}, nil)
		if err != nil {
			t.Errorf("Expected no error for empty list, got %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty list, got %d", len(results))
		}
	})
}

func TestProtoParser_ParseFileStructures(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	parser := NewProtoParser(logAdapter)
	ctx := context.Background()

	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "complex.proto")

	// More complex proto file for testing
	protoContent := `syntax = "proto3";

package complex;

import "google/protobuf/timestamp.proto";

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

message User {
  string id = 1;
  string name = 2;
  repeated string emails = 3;
  Status status = 4;
  google.protobuf.Timestamp created_at = 5;
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc ListUsers(google.protobuf.Empty) returns (stream User);
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	result, err := parser.ParseFile(ctx, protoFile, []string{tempDir})
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Verify imports
	if len(result.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(result.Imports))
	}

	// Verify enums
	if len(result.Enums) != 1 {
		t.Errorf("Expected 1 enum, got %d", len(result.Enums))
	} else {
		enum := result.Enums[0]
		if enum.Name != "Status" {
			t.Errorf("Expected enum name 'Status', got '%s'", enum.Name)
		}
		if len(enum.Values) != 3 {
			t.Errorf("Expected 3 enum values, got %d", len(enum.Values))
		}
	}

	// Verify messages
	if len(result.Messages) < 2 {
		t.Errorf("Expected at least 2 messages, got %d", len(result.Messages))
	}

	// Find User message
	var userMsg *Message
	for i := range result.Messages {
		if result.Messages[i].Name == "User" {
			userMsg = &result.Messages[i]
			break
		}
	}

	if userMsg == nil {
		t.Fatal("User message not found")
	}

	if len(userMsg.Fields) != 5 {
		t.Errorf("Expected 5 fields in User message, got %d", len(userMsg.Fields))
	}

	// Verify repeated field
	var emailsField *Field
	for i := range userMsg.Fields {
		if userMsg.Fields[i].Name == "emails" {
			emailsField = &userMsg.Fields[i]
			break
		}
	}

	if emailsField == nil {
		t.Fatal("emails field not found")
	}

	if !emailsField.Repeated {
		t.Error("Expected emails field to be repeated")
	}

	// Verify services
	if len(result.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(result.Services))
	} else {
		service := result.Services[0]
		if service.Name != "UserService" {
			t.Errorf("Expected service name 'UserService', got '%s'", service.Name)
		}
		if len(service.Methods) != 2 {
			t.Errorf("Expected 2 methods, got %d", len(service.Methods))
		}
	}
}
