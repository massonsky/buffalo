package utils

import (
	"path/filepath"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple path", "test/path", false},
		{"path with dots", "test/../path", false},
		{"empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestIsAbsolutePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"absolute unix", "/usr/bin", true},
		{"absolute windows", "C:\\Users\\test", true},
		{"relative", "test/path", false},
		{"current dir", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbsolutePath(tt.path)
			// Only test on matching OS
			if filepath.IsAbs(tt.path) == tt.expected {
				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	result := JoinPath("test", "path", "file.txt")
	expected := filepath.Join("test", "path", "file.txt")

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		target  string
		wantErr bool
	}{
		{"valid paths", "/home/user", "/home/user/docs", false},
		{"empty base", "", "/home/user/docs", true},
		{"empty target", "/home/user", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetRelativePath(tt.base, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetBaseName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.txt", "file.txt"},
		{"file.txt", "file.txt"},
		{"/path/to/dir/", "dir"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetBaseName(tt.path)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetDirName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.txt", "/path/to"},
		{"file.txt", "."},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetDirName(tt.path)
			// Normalize for cross-platform
			result = filepath.ToSlash(result)
			expected := filepath.ToSlash(tt.expected)
			if result != expected {
				t.Errorf("expected %s, got %s", expected, result)
			}
		})
	}
}

func TestChangeExtension(t *testing.T) {
	tests := []struct {
		path     string
		newExt   string
		expected string
	}{
		{"file.txt", ".md", "file.md"},
		{"file.txt", "md", "file.md"},
		{"path/to/file.proto", ".go", "path/to/file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ChangeExtension(tt.path, tt.newExt)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRemoveExtension(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"file.txt", "file"},
		{"path/to/file.proto", "path/to/file"},
		{"file", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := RemoveExtension(tt.path)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSplitExtension(t *testing.T) {
	tests := []struct {
		path         string
		expectedName string
		expectedExt  string
	}{
		{"file.txt", "file", ".txt"},
		{"path/to/file.proto", "path/to/file", ".proto"},
		{"file", "file", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			name, ext := SplitExtension(tt.path)
			if name != tt.expectedName || ext != tt.expectedExt {
				t.Errorf("expected (%s, %s), got (%s, %s)",
					tt.expectedName, tt.expectedExt, name, ext)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		name     string
		expected bool
		wantErr  bool
	}{
		{"*.proto", "file.proto", true, false},
		{"*.proto", "file.txt", false, false},
		{"test*.proto", "test_file.proto", true, false},
		{"[", "file", false, true}, // Invalid pattern
	}

	for _, tt := range tests {
		t.Run(tt.pattern+":"+tt.name, func(t *testing.T) {
			result, err := MatchPattern(tt.pattern, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestToSlash(t *testing.T) {
	path := "test\\path\\file.txt"
	result := ToSlash(path)
	expected := "test/path/file.txt"

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFromSlash(t *testing.T) {
	path := "test/path/file.txt"
	result := FromSlash(path)
	expected := filepath.FromSlash(path)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func BenchmarkNormalizePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NormalizePath("test/path/../file.txt")
	}
}

func BenchmarkJoinPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = JoinPath("test", "path", "to", "file.txt")
	}
}
