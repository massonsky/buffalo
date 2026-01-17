package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheManager_PutAndGet(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	entry := &CacheEntry{
		File:      "test.proto",
		Hash:      "abc123",
		DepsHash:  "def456",
		Languages: []string{"go", "python"},
		GeneratedFiles: map[string][]string{
			"go":     {"test.pb.go", "test_grpc.pb.go"},
			"python": {"test_pb2.py", "test_pb2_grpc.py"},
		},
	}

	// Test Put
	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Test Get
	retrieved, err := cache.Get("test.proto")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.File != entry.File {
		t.Errorf("Expected file %s, got %s", entry.File, retrieved.File)
	}

	if retrieved.Hash != entry.Hash {
		t.Errorf("Expected hash %s, got %s", entry.Hash, retrieved.Hash)
	}

	if len(retrieved.Languages) != len(entry.Languages) {
		t.Errorf("Expected %d languages, got %d", len(entry.Languages), len(retrieved.Languages))
	}
}

func TestCacheManager_Check(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	// Create a test proto file
	protoFile := filepath.Join(tempDir, "test.proto")
	content := "syntax = \"proto3\";\npackage test;\n"
	if err := os.WriteFile(protoFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// File not in cache yet
	valid, err := cache.Check(protoFile, nil)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if valid {
		t.Error("Expected cache miss for new file")
	}

	// Add to cache
	hash, err := cache.computeFileHash(protoFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	entry := &CacheEntry{
		File:      protoFile,
		Hash:      hash,
		DepsHash:  "no-deps",
		Languages: []string{"go"},
	}

	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Now check should pass
	valid, err = cache.Check(protoFile, nil)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !valid {
		t.Error("Expected cache hit for unchanged file")
	}

	// Modify the file
	newContent := "syntax = \"proto3\";\npackage test;\nmessage Test {}\n"
	if err := os.WriteFile(protoFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Check should fail now
	valid, err = cache.Check(protoFile, nil)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if valid {
		t.Error("Expected cache miss for modified file")
	}
}

func TestCacheManager_Invalidate(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	// Add entries
	entries := []*CacheEntry{
		{File: "a.proto", Hash: "hash1", DepsHash: "deps1", Languages: []string{"go"}},
		{File: "b.proto", Hash: "hash2", DepsHash: "deps2", Languages: []string{"go"}},
		{File: "c.proto", Hash: "hash3", DepsHash: "deps3", Languages: []string{"go"}},
	}

	for _, entry := range entries {
		if err := cache.Put(entry); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Invalidate b.proto
	if err := cache.Invalidate("b.proto"); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// a.proto and c.proto should still exist
	if _, err := cache.Get("a.proto"); err != nil {
		t.Error("a.proto should still be in cache")
	}

	if _, err := cache.Get("c.proto"); err != nil {
		t.Error("c.proto should still be in cache")
	}

	// b.proto should be gone
	if _, err := cache.Get("b.proto"); err == nil {
		t.Error("b.proto should not be in cache")
	}
}

func TestCacheManager_Clear(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	// Add entries
	entries := []*CacheEntry{
		{File: "a.proto", Hash: "hash1", DepsHash: "deps1", Languages: []string{"go"}},
		{File: "b.proto", Hash: "hash2", DepsHash: "deps2", Languages: []string{"go"}},
	}

	for _, entry := range entries {
		if err := cache.Put(entry); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Verify cache directory exists and has files
	cacheDir := filepath.Join(tempDir, ".buffalo-cache")
	entries1, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("Failed to read cache dir: %v", err)
	}
	if len(entries1) == 0 {
		t.Error("Cache directory should have files")
	}

	// Clear cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Cache directory should be empty
	entries2, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("Failed to read cache dir after clear: %v", err)
	}
	if len(entries2) != 0 {
		t.Errorf("Cache directory should be empty, got %d files", len(entries2))
	}

	// Getting entries should fail
	if _, err := cache.Get("a.proto"); err == nil {
		t.Error("a.proto should not be in cache after clear")
	}
}

func TestCacheManager_ComputeFileHash(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.proto")
	content := "syntax = \"proto3\";\npackage test;\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := cache.computeFileHash(testFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	// Same file should produce same hash
	hash2, err := cache.computeFileHash(testFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Same file should produce same hash")
	}

	// Different content should produce different hash
	newContent := "syntax = \"proto3\";\npackage different;\n"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash3, err := cache.computeFileHash(testFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}
}

func TestCacheManager_ComputeDepsHash(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	// Create dependency files
	deps := make(map[string]string)
	for _, name := range []string{"dep1.proto", "dep2.proto", "dep3.proto"} {
		depFile := filepath.Join(tempDir, name)
		content := "syntax = \"proto3\";\npackage " + name + ";\n"
		if err := os.WriteFile(depFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create dep file: %v", err)
		}
		deps[name] = depFile
	}

	// Convert to []*ProtoFile
	var depFiles []*ProtoFile
	for name, path := range deps {
		depFiles = append(depFiles, &ProtoFile{
			Path:    path,
			Package: name,
		})
	}

	hash1, err := cache.computeDepsHash(depFiles)
	if err != nil {
		t.Fatalf("computeDepsHash failed: %v", err)
	}

	if hash1 == "" {
		t.Error("Deps hash should not be empty")
	}

	// Same deps should produce same hash
	hash2, err := cache.computeDepsHash(depFiles)
	if err != nil {
		t.Fatalf("computeDepsHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Same deps should produce same hash")
	}

	// Empty deps
	hash3, err := cache.computeDepsHash(nil)
	if err != nil {
		t.Fatalf("computeDepsHash failed for nil deps: %v", err)
	}

	if hash3 == "" {
		t.Error("Empty deps should still produce a hash")
	}

	if hash1 == hash3 {
		t.Error("Different deps should produce different hash")
	}
}

func TestCacheManager_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	cache := NewCacheManager(tempDir)

	// Get non-existent file
	_, err := cache.Get("nonexistent.proto")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Check non-existent file
	_, err = cache.Check("nonexistent.proto", nil)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Invalidate non-existent file (should not error)
	err = cache.Invalidate("nonexistent.proto")
	if err != nil {
		t.Errorf("Invalidate should not error for non-existent file: %v", err)
	}
}
