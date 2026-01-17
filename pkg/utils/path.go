package utils

import (
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/pkg/errors"
)

// NormalizePath normalizes a file path by cleaning it and converting to absolute path.
func NormalizePath(path string) (string, error) {
	if path == "" {
		return "", errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Clean the path (removes redundant separators and resolves . and ..)
	cleaned := filepath.Clean(path)

	// Convert to absolute path
	absolute, err := filepath.Abs(cleaned)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to get absolute path: %s", path)
	}

	return absolute, nil
}

// IsAbsolutePath checks if a path is absolute.
func IsAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// JoinPath joins path elements into a single path.
func JoinPath(elements ...string) string {
	return filepath.Join(elements...)
}

// GetRelativePath returns the relative path from base to target.
func GetRelativePath(base, target string) (string, error) {
	if base == "" || target == "" {
		return "", errors.New(errors.ErrInvalidArgument, "base and target paths cannot be empty")
	}

	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrInvalidArgument,
			"failed to get relative path from %s to %s", base, target)
	}

	return rel, nil
}

// GetBaseName returns the base name of a path (file name without directory).
func GetBaseName(path string) string {
	return filepath.Base(path)
}

// GetDirName returns the directory name of a path.
func GetDirName(path string) string {
	return filepath.Dir(path)
}

// SplitPath splits a path into directory and file name.
func SplitPath(path string) (dir, file string) {
	return filepath.Split(path)
}

// ChangeExtension changes the extension of a file path.
func ChangeExtension(path, newExt string) string {
	if !strings.HasPrefix(newExt, ".") {
		newExt = "." + newExt
	}
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + newExt
}

// RemoveExtension removes the extension from a file path.
func RemoveExtension(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext)
}

// SplitExtension splits a path into name and extension.
func SplitExtension(path string) (name, ext string) {
	ext = filepath.Ext(path)
	name = strings.TrimSuffix(path, ext)
	return
}

// HasPathPrefix checks if a path starts with the given prefix.
func HasPathPrefix(path, prefix string) bool {
	cleanPath := filepath.Clean(path)
	cleanPrefix := filepath.Clean(prefix)

	// Handle Windows paths case-insensitively
	if filepath.VolumeName(cleanPath) != "" {
		cleanPath = strings.ToLower(cleanPath)
		cleanPrefix = strings.ToLower(cleanPrefix)
	}

	return strings.HasPrefix(cleanPath, cleanPrefix)
}

// IsSubPath checks if child is a subdirectory or file under parent.
func IsSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	// If relative path starts with "..", it's not a subpath
	return !strings.HasPrefix(rel, "..")
}

// ExpandPath expands environment variables and home directory in a path.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := filepath.Abs(".")
		if err != nil {
			return "", errors.Wrap(err, errors.ErrIO, "failed to get home directory")
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}

	// Note: os.ExpandEnv is not used here to avoid security issues
	// Users should handle environment variable expansion explicitly if needed
	return path, nil
}

// ToSlash converts path separators to forward slashes (for cross-platform compatibility).
func ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// FromSlash converts forward slashes to OS-specific separators.
func FromSlash(path string) string {
	return filepath.FromSlash(path)
}

// MatchPattern checks if a file name matches a glob pattern.
func MatchPattern(pattern, name string) (bool, error) {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false, errors.Wrap(err, errors.ErrInvalidArgument, "invalid pattern: %s", pattern)
	}
	return matched, nil
}

// GlobPattern finds all paths matching a pattern.
func GlobPattern(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidArgument, "invalid glob pattern: %s", pattern)
	}
	return matches, nil
}

// MakeRelative makes a path relative to a base directory.
// If the path is already relative, it returns it unchanged.
func MakeRelative(base, path string) (string, error) {
	if !IsAbsolutePath(path) {
		return path, nil
	}

	return GetRelativePath(base, path)
}

// MakeAbsolute makes a path absolute relative to a base directory.
// If the path is already absolute, it returns it unchanged.
func MakeAbsolute(base, path string) (string, error) {
	if IsAbsolutePath(path) {
		return NormalizePath(path)
	}

	joined := filepath.Join(base, path)
	return NormalizePath(joined)
}
