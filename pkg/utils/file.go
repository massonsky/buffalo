package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/pkg/errors"
)

// FileInfo contains information about a file.
type FileInfo struct {
	Path     string      // Absolute path to the file
	RelPath  string      // Relative path to the file
	Size     int64       // File size in bytes
	IsDir    bool        // Whether the path is a directory
	ModTime  int64       // Modification time (Unix timestamp)
	FileInfo os.FileInfo // Original FileInfo from os
}

// FindFilesOptions contains options for finding files.
type FindFilesOptions struct {
	Pattern     string   // File pattern to match (e.g., "*.proto")
	Recursive   bool     // Search recursively in subdirectories
	IncludeDirs bool     // Include directories in results
	Exclude     []string // Patterns to exclude
}

// FindFiles finds files in a directory matching the given options.
func FindFiles(root string, opts FindFilesOptions) ([]FileInfo, error) {
	if root == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "root directory cannot be empty")
	}

	// Check if root exists
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err, errors.ErrNotFound, "directory not found: %s", root)
		}
		return nil, errors.Wrap(err, errors.ErrIO, "failed to stat directory: %s", root)
	}

	var results []FileInfo

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, errors.ErrIO, "error walking path: %s", path)
		}

		// Skip directories if not included
		if info.IsDir() && !opts.IncludeDirs && path != root {
			if !opts.Recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Check exclusion patterns
		for _, pattern := range opts.Exclude {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check file pattern
		if opts.Pattern != "" {
			matched, err := filepath.Match(opts.Pattern, filepath.Base(path))
			if err != nil {
				return errors.Wrap(err, errors.ErrInvalidArgument, "invalid pattern: %s", opts.Pattern)
			}
			if !matched {
				return nil
			}
		}

		relPath, _ := filepath.Rel(root, path)
		results = append(results, FileInfo{
			Path:     path,
			RelPath:  relPath,
			Size:     info.Size(),
			IsDir:    info.IsDir(),
			ModTime:  info.ModTime().Unix(),
			FileInfo: info,
		})

		return nil
	}

	if err := filepath.Walk(root, walkFn); err != nil {
		return nil, err
	}

	return results, nil
}

// FileExists checks if a file or directory exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// EnsureDir creates a directory and all necessary parent directories.
func EnsureDir(path string) error {
	if path == "" {
		return errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to create directory: %s", path)
	}

	return nil
}

// CleanDir removes all contents of a directory but keeps the directory itself.
func CleanDir(path string) error {
	if path == "" {
		return errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Check if directory exists
	if !IsDir(path) {
		return errors.New(errors.ErrNotFound, "directory not found: %s", path)
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to read directory: %s", path)
	}

	// Remove each entry
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return errors.Wrap(err, errors.ErrIO, "failed to remove: %s", entryPath)
		}
	}

	return nil
}

// RemoveDir removes a directory and all its contents.
func RemoveDir(path string) error {
	if path == "" {
		return errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	if err := os.RemoveAll(path); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to remove directory: %s", path)
	}

	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	if src == "" || dst == "" {
		return errors.New(errors.ErrInvalidArgument, "source and destination cannot be empty")
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Wrap(err, errors.ErrNotFound, "source file not found: %s", src)
		}
		return errors.Wrap(err, errors.ErrIO, "failed to open source file: %s", src)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to stat source file: %s", src)
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir); err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to create destination file: %s", dst)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to copy file contents")
	}

	// Copy permissions
	if err := dstFile.Chmod(srcInfo.Mode()); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to set file permissions")
	}

	return nil
}

// ReadFile reads the entire contents of a file.
func ReadFile(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(err, errors.ErrNotFound, "file not found: %s", path)
		}
		return nil, errors.Wrap(err, errors.ErrIO, "failed to read file: %s", path)
	}

	return data, nil
}

// WriteFile writes data to a file, creating it if necessary.
func WriteFile(path string, data []byte) error {
	if path == "" {
		return errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return errors.Wrap(err, errors.ErrIO, "failed to write file: %s", path)
	}

	return nil
}

// GetFileSize returns the size of a file in bytes.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, errors.Wrap(err, errors.ErrNotFound, "file not found: %s", path)
		}
		return 0, errors.Wrap(err, errors.ErrIO, "failed to stat file: %s", path)
	}

	return info.Size(), nil
}

// GetFileExtension returns the extension of a file (including the dot).
func GetFileExtension(path string) string {
	return filepath.Ext(path)
}

// HasExtension checks if a file has one of the specified extensions.
func HasExtension(path string, extensions ...string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range extensions {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}
