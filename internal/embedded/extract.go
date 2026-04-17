package embedded

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ExtractValidateProto extracts the embedded buffalo/validate/validate.proto
// into the given workspace directory.
//
// The file is placed at:
//
//	<workspaceDir>/proto/buffalo/validate/validate.proto
//
// It returns the absolute path to the directory that should be added to
// protoc's --proto_path (i.e. <workspaceDir>/proto).
func ExtractValidateProto(workspaceDir string) (protoPath string, err error) {
	return ExtractAllProtos(workspaceDir)
}

// ExtractAllProtos extracts every embedded proto file into workspaceDir
// and returns the root proto_path that should be passed to protoc.
//
// Existing files are overwritten to ensure the version matches the binary.
func ExtractAllProtos(workspaceDir string) (protoPath string, err error) {
	protoPath = filepath.Join(workspaceDir, "proto")

	err = fs.WalkDir(ProtoFS, "proto", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		target := filepath.Join(workspaceDir, path) // workspaceDir + "proto/buffalo/validate/..."

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := ProtoFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
		}

		if err := os.WriteFile(target, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}

		return nil
	})

	return protoPath, err
}

// ValidateProtoImportPath returns the proto_path that should be added to
// protoc / Buffalo import paths so that `import "buffalo/validate/validate.proto";`
// and `import "buffalo/permissions/permissions.proto";` resolve correctly.
// If any embedded files haven't been extracted yet, it re-extracts them all.
//
// workspaceDir is typically ".buffalo".
func ValidateProtoImportPath(workspaceDir string) (string, error) {
	// Check all known embedded proto files, not just validate.
	requiredFiles := []string{
		filepath.Join(workspaceDir, "proto", "buffalo", "validate", "validate.proto"),
		filepath.Join(workspaceDir, "proto", "buffalo", "permissions", "permissions.proto"),
		filepath.Join(workspaceDir, "proto", "buffalo", "models", "models.proto"),
	}

	for _, target := range requiredFiles {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			// At least one file is missing — re-extract everything.
			return ExtractAllProtos(workspaceDir)
		}
	}

	return filepath.Join(workspaceDir, "proto"), nil
}

// ListEmbeddedProtos returns a list of all embedded proto file paths.
func ListEmbeddedProtos() ([]string, error) {
	var files []string
	err := fs.WalkDir(ProtoFS, "proto", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ExtractBazelRules extracts the embedded rules_buffalo Bazel module
// into <workspaceDir>/bazel/rules_buffalo/.
//
// The extracted files provide Bazel rules for Buffalo proto compilation:
//   - buffalo_proto_compile — hermetic rule for bazel build
//   - buffalo_proto_gen — source-tree generation via bazel run
//
// Returns the absolute path to the extracted rules_buffalo directory.
func ExtractBazelRules(workspaceDir string) (rulesPath string, err error) {
	rulesPath = filepath.Join(workspaceDir, "bazel", "rules_buffalo")

	// Walk embedded bazel/rules_buffalo/ and extract
	err = fs.WalkDir(BazelFS, "bazel/rules_buffalo", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Strip "bazel/rules_buffalo" prefix, rebuild under workspaceDir/bazel/rules_buffalo
		rel := strings.TrimPrefix(path, "bazel/rules_buffalo")
		if rel == "" {
			rel = "."
		} else {
			rel = strings.TrimPrefix(rel, "/")
		}

		target := filepath.Join(rulesPath, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := BazelFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
		}

		if err := os.WriteFile(target, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}

		return nil
	})

	return rulesPath, err
}

// ListEmbeddedBazelFiles returns a list of all embedded Bazel rule file paths.
func ListEmbeddedBazelFiles() ([]string, error) {
	var files []string
	err := fs.WalkDir(BazelFS, "bazel", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
