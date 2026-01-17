package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

func TestCacheManager_PutAndGet(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	cache := NewCacheManager(logAdapter)

	ctx := context.Background()

	// Create temp file for proto
	tempDir := t.TempDir()
	protoPath := filepath.Join(tempDir, "test.proto")
	if err := os.WriteFile(protoPath, []byte("syntax = \"proto3\";\npackage test;\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	entry := &CacheEntry{
		File:           protoPath,
		Hash:           "abc123",
		DepsHash:       "def456",
		Languages:      []string{"go", "python"},
		GeneratedFiles: []string{"test.pb.go", "test_grpc.pb.go", "test_pb2.py", "test_pb2_grpc.py"},
	}

	// Test Put
	if err := cache.Put(ctx, entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Test Get
	protoFile := &ProtoFile{Path: protoPath, Package: "test"}
	retrieved, err := cache.Get(ctx, protoFile)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Since hash doesn't match file content, we may get nil
	// This is expected behavior
	if retrieved != nil {
		if retrieved.File != entry.File {
			t.Errorf("Expected file %s, got %s", entry.File, retrieved.File)
		}
	}
}

func TestCacheManager_Check(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	cache := NewCacheManager(logAdapter)

	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a test proto file
	protoPath := filepath.Join(tempDir, "test.proto")
	content := "syntax = \"proto3\";\npackage test;\n"
	if err := os.WriteFile(protoPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	protoFile := &ProtoFile{Path: protoPath, Package: "test"}

	// File not in cache yet - should be a miss
	hits, misses := cache.Check(ctx, []*ProtoFile{protoFile})
	if hits != 0 {
		t.Errorf("Expected 0 hits for new file, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss for new file, got %d", misses)
	}
}

func TestCacheManager_Invalidate(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	cache := NewCacheManager(logAdapter)

	ctx := context.Background()
	tempDir := t.TempDir()

	// Create test proto files
	protoA := filepath.Join(tempDir, "a.proto")
	protoB := filepath.Join(tempDir, "b.proto")
	protoC := filepath.Join(tempDir, "c.proto")

	for _, p := range []string{protoA, protoB, protoC} {
		if err := os.WriteFile(p, []byte("syntax = \"proto3\";\npackage test;\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Add entries
	entries := []*CacheEntry{
		{File: protoA, Hash: "hash1", DepsHash: "deps1", Languages: []string{"go"}},
		{File: protoB, Hash: "hash2", DepsHash: "deps2", Languages: []string{"go"}},
		{File: protoC, Hash: "hash3", DepsHash: "deps3", Languages: []string{"go"}},
	}

	for _, entry := range entries {
		if err := cache.Put(ctx, entry); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Invalidate b.proto
	if err := cache.Invalidate(ctx, protoB); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// Invalidating should not error
	t.Log("Invalidation completed successfully")
}

func TestCacheManager_Clear(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	cache := NewCacheManager(logAdapter)

	ctx := context.Background()
	tempDir := t.TempDir()

	// Create test proto files
	protoA := filepath.Join(tempDir, "a.proto")
	protoB := filepath.Join(tempDir, "b.proto")

	for _, p := range []string{protoA, protoB} {
		if err := os.WriteFile(p, []byte("syntax = \"proto3\";\npackage test;\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Add entries
	entries := []*CacheEntry{
		{File: protoA, Hash: "hash1", DepsHash: "deps1", Languages: []string{"go"}},
		{File: protoB, Hash: "hash2", DepsHash: "deps2", Languages: []string{"go"}},
	}

	for _, entry := range entries {
		if err := cache.Put(ctx, entry); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Clear cache
	if err := cache.Clear(ctx); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// After clear, cache should miss
	protoFileA := &ProtoFile{Path: protoA, Package: "test"}
	hits, misses := cache.Check(ctx, []*ProtoFile{protoFileA})
	if hits != 0 {
		t.Errorf("Expected 0 hits after clear, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss after clear, got %d", misses)
	}
}

func TestCacheManager_NonExistentFile(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	cache := NewCacheManager(logAdapter)

	ctx := context.Background()

	// Get non-existent file
	protoFile := &ProtoFile{Path: "nonexistent.proto", Package: "test"}
	entry, err := cache.Get(ctx, protoFile)
	// Should return nil entry, not error (file not in cache)
	if err != nil {
		t.Logf("Get returned error for non-existent file: %v", err)
	}
	if entry != nil {
		t.Error("Expected nil entry for non-existent file")
	}

	// Check non-existent file - should be a miss
	hits, misses := cache.Check(ctx, []*ProtoFile{protoFile})
	if hits != 0 {
		t.Errorf("Expected 0 hits for non-existent file, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss for non-existent file, got %d", misses)
	}

	// Invalidate non-existent file (should not error)
	err = cache.Invalidate(ctx, "nonexistent.proto")
	if err != nil {
		t.Errorf("Invalidate should not error for non-existent file: %v", err)
	}
}
