package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindFiles(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create test files
	files := []string{
		"test1.proto",
		"test2.proto",
		"test.txt",
		"subdir/test3.proto",
		"subdir/test4.txt",
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if err := EnsureDir(filepath.Dir(path)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name          string
		opts          FindFilesOptions
		expectedCount int
	}{
		{
			name: "find all proto files recursively",
			opts: FindFilesOptions{
				Pattern:   "*.proto",
				Recursive: true,
			},
			expectedCount: 3,
		},
		{
			name: "find proto files non-recursively",
			opts: FindFilesOptions{
				Pattern:   "*.proto",
				Recursive: false,
			},
			expectedCount: 2,
		},
		{
			name: "find all files",
			opts: FindFilesOptions{
				Recursive: true,
			},
			expectedCount: 6, // 5 files + 1 subdir
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := FindFiles(tempDir, tt.opts)
			if err != nil {
				t.Fatalf("FindFiles failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("expected %d files, got %d", tt.expectedCount, len(results))
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "exists.txt")

	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", existingFile, true},
		{"non-existing file", filepath.Join(tempDir, "notexist.txt"), false},
		{"directory", tempDir, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if !IsDir(tempDir) {
		t.Error("expected directory to be recognized")
	}

	if IsDir(file) {
		t.Error("expected file to not be recognized as directory")
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new", "nested", "dir")

	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if !IsDir(newDir) {
		t.Error("directory was not created")
	}

	// Test creating existing directory (should not fail)
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir should not fail for existing directory: %v", err)
	}
}

func TestCleanDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create some files
	for i := 0; i < 3; i++ {
		file := filepath.Join(tempDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Clean directory
	if err := CleanDir(tempDir); err != nil {
		t.Fatalf("CleanDir failed: %v", err)
	}

	// Check directory is empty
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Errorf("expected empty directory, got %d entries", len(entries))
	}

	// Check directory still exists
	if !IsDir(tempDir) {
		t.Error("directory should still exist after cleaning")
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()

	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "dest.txt")
	content := []byte("test content")

	// Create source file
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Copy file
	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file
	if !FileExists(dstFile) {
		t.Fatal("destination file does not exist")
	}

	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("expected content '%s', got '%s'", content, dstContent)
	}
}

func TestReadWriteFile(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")

	// Write file
	if err := WriteFile(file, content); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file
	readContent, err := ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("expected content '%s', got '%s'", content, readContent)
	}
}

func TestGetFileSize(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")

	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	size, err := GetFileSize(file)
	if err != nil {
		t.Fatalf("GetFileSize failed: %v", err)
	}

	expectedSize := int64(len(content))
	if size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, size)
	}
}

func TestHasExtension(t *testing.T) {
	tests := []struct {
		path       string
		extensions []string
		expected   bool
	}{
		{"test.proto", []string{".proto"}, true},
		{"test.proto", []string{".txt", ".proto"}, true},
		{"test.txt", []string{".proto"}, false},
		{"test", []string{".proto"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := HasExtension(tt.path, tt.extensions...)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func BenchmarkFindFiles(b *testing.B) {
	tempDir := b.TempDir()

	// Create test files
	for i := 0; i < 100; i++ {
		file := filepath.Join(tempDir, "test"+string(rune('0'+i%10))+".proto")
		os.WriteFile(file, []byte("test"), 0644)
	}

	opts := FindFilesOptions{
		Pattern:   "*.proto",
		Recursive: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FindFiles(tempDir, opts)
	}
}

func BenchmarkCopyFile(b *testing.B) {
	tempDir := b.TempDir()
	srcFile := filepath.Join(tempDir, "source.txt")
	content := make([]byte, 1024*1024) // 1MB
	os.WriteFile(srcFile, content, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dstFile := filepath.Join(tempDir, "dest"+string(rune('0'+i%10))+".txt")
		_ = CopyFile(srcFile, dstFile)
	}
}
