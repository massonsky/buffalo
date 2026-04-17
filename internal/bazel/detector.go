package bazel

import (
	"os"
	"path/filepath"
)

// workspaceFiles lists the files that indicate a Bazel workspace root,
// ordered by preference (bzlmod first).
var workspaceFiles = []string{
	"MODULE.bazel",
	"WORKSPACE.bazel",
	"WORKSPACE",
}

// DetectWorkspace searches upward from dir looking for a Bazel workspace root.
// Returns nil if no workspace is found.
func DetectWorkspace(dir string) (*Workspace, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	root, wsType, err := findWorkspaceRoot(absDir)
	if err != nil {
		return nil, err
	}
	if root == "" {
		return nil, nil // no workspace found
	}

	ws := &Workspace{
		Root:       root,
		Type:       wsType,
		BuildFiles: make(map[string]string),
	}

	// Read module name for bzlmod
	if wsType == "bzlmod" {
		ws.ModuleName = readModuleName(filepath.Join(root, "MODULE.bazel"))
	}

	return ws, nil
}

// findWorkspaceRoot walks up from dir until it finds a workspace marker file.
func findWorkspaceRoot(dir string) (string, string, error) { //nolint:unparam // error return kept for interface compatibility
	current := dir
	for {
		for _, name := range workspaceFiles {
			path := filepath.Join(current, name)
			if _, err := os.Stat(path); err == nil {
				wsType := "workspace"
				if name == "MODULE.bazel" {
					wsType = "bzlmod"
				}
				return current, wsType, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", "", nil // reached filesystem root
		}
		current = parent
	}
}

// readModuleName extracts the module name from a MODULE.bazel file.
// Returns empty string on any failure — non-critical.
func readModuleName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return parseModuleName(string(data))
}

// FindBuildFiles discovers all BUILD and BUILD.bazel files under root.
func FindBuildFiles(root string) (map[string]string, error) {
	result := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible directories
		}

		// Skip common non-Bazel directories
		if info.IsDir() {
			base := info.Name()
			if base == ".git" || base == "node_modules" || base == "bazel-bin" ||
				base == "bazel-out" || base == "bazel-testlogs" || base == ".buffalo-cache" {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		if base == "BUILD" || base == "BUILD.bazel" {
			rel, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return nil
			}
			pkg := "//" + filepath.ToSlash(rel)
			if rel == "." {
				pkg = "//"
			}
			result[pkg] = path
		}

		return nil
	})

	return result, err
}

// IsBazelWorkspace returns true if the given directory (or a parent) is a Bazel workspace.
func IsBazelWorkspace(dir string) bool {
	ws, err := DetectWorkspace(dir)
	return err == nil && ws != nil
}

// HasBazelFile returns true if the specific directory contains a BUILD file.
func HasBazelFile(dir string) bool {
	for _, name := range []string{"BUILD.bazel", "BUILD"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
