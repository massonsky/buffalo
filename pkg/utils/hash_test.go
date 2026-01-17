package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeHash(t *testing.T) {
	data := []byte("test data")

	tests := []struct {
		algo HashAlgorithm
		want string
	}{
		{MD5, "eb733a00c0c9d336e65691a37ab54293"},
		{SHA1, "f48dd853820860816c75d54d0f584dc863327a7c"},
		{SHA256, "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			hash, err := ComputeHash(data, tt.algo)
			if err != nil {
				t.Fatalf("ComputeHash failed: %v", err)
			}

			if hash != tt.want {
				t.Errorf("expected hash %s, got %s", tt.want, hash)
			}
		})
	}
}

func TestComputeStringHash(t *testing.T) {
	input := "test string"

	hash1, err := ComputeStringHash(input, SHA256)
	if err != nil {
		t.Fatalf("ComputeStringHash failed: %v", err)
	}

	hash2, err := ComputeStringHash(input, SHA256)
	if err != nil {
		t.Fatalf("ComputeStringHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("expected same hash for same input")
	}

	hash3, err := ComputeStringHash("different", SHA256)
	if err != nil {
		t.Fatal(err)
	}

	if hash1 == hash3 {
		t.Error("expected different hash for different input")
	}
}

func TestComputeFileHash(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")

	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := ComputeFileHash(file, SHA256)
	if err != nil {
		t.Fatalf("ComputeFileHash failed: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Verify hash matches content hash
	contentHash, err := ComputeHash(content, SHA256)
	if err != nil {
		t.Fatal(err)
	}

	if hash != contentHash {
		t.Error("file hash should match content hash")
	}
}

func TestVerifyFileHash(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")

	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Compute expected hash
	expectedHash, err := ComputeFileHash(file, SHA256)
	if err != nil {
		t.Fatal(err)
	}

	// Verify correct hash
	valid, err := VerifyFileHash(file, expectedHash, SHA256)
	if err != nil {
		t.Fatalf("VerifyFileHash failed: %v", err)
	}

	if !valid {
		t.Error("expected hash to be valid")
	}

	// Verify incorrect hash
	valid, err = VerifyFileHash(file, "wronghash", SHA256)
	if err != nil {
		t.Fatal(err)
	}

	if valid {
		t.Error("expected hash to be invalid")
	}
}

func TestHashConvenienceFunctions(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")

	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Test HashFile
	hash1, err := HashFile(file)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	// Test HashData
	hash2, err := HashData(content)
	if err != nil {
		t.Fatalf("HashData failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("HashFile and HashData should return same result")
	}

	// Test HashString
	hash3, err := HashString(string(content))
	if err != nil {
		t.Fatalf("HashString failed: %v", err)
	}

	if hash1 != hash3 {
		t.Error("HashString should match HashData")
	}
}

func TestInvalidHashAlgorithm(t *testing.T) {
	_, err := ComputeHash([]byte("test"), "invalid")
	if err == nil {
		t.Error("expected error for invalid hash algorithm")
	}
}

func BenchmarkComputeHash_MD5(b *testing.B) {
	data := []byte("benchmark data for hashing")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ComputeHash(data, MD5)
	}
}

func BenchmarkComputeHash_SHA256(b *testing.B) {
	data := []byte("benchmark data for hashing")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ComputeHash(data, SHA256)
	}
}

func BenchmarkComputeFileHash(b *testing.B) {
	tempDir := b.TempDir()
	file := filepath.Join(tempDir, "bench.txt")
	content := make([]byte, 1024*1024) // 1MB
	os.WriteFile(file, content, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ComputeFileHash(file, SHA256)
	}
}
