package embedded

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
// resolves correctly. If the files haven't been extracted yet, it extracts them.
//
// workspaceDir is typically ".buffalo".
func ValidateProtoImportPath(workspaceDir string) (string, error) {
	protoDir := filepath.Join(workspaceDir, "proto", "buffalo", "validate")
	target := filepath.Join(protoDir, "validate.proto")

	if _, err := os.Stat(target); os.IsNotExist(err) {
		// Auto-extract
		return ExtractAllProtos(workspaceDir)
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
