package dependency

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/tracing"
)

// Downloader handles downloading dependencies from various sources.
type Downloader struct {
	targetDir string
	log       *logger.Logger
}

// NewDownloader creates a new downloader.
func NewDownloader(targetDir string, log *logger.Logger) *Downloader {
	return &Downloader{
		targetDir: targetDir,
		log:       log,
	}
}

// Download downloads a dependency based on its source.
func (d *Downloader) Download(ctx context.Context, dep Dependency) (result *DownloadResult, err error) {
	ctx, span := tracing.StartSpan(ctx, "dep.fetch", tracing.WithAttributes(map[string]any{
		"dep.name":   dep.Name,
		"dep.source": dep.Source.Type,
	}))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(tracing.StatusError, err.Error())
		} else {
			span.SetStatus(tracing.StatusOK, "")
		}
		span.End()
	}()

	switch dep.Source.Type {
	case "git":
		return d.downloadGit(ctx, dep)
	case "url":
		return d.downloadURL(ctx, dep)
	case "local":
		return d.copyLocal(dep)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", dep.Source.Type)
	}
}

// downloadGit clones a git repository.
func (d *Downloader) downloadGit(ctx context.Context, dep Dependency) (*DownloadResult, error) {
	d.log.Info("Cloning git repository", logger.String("url", dep.Source.URL))

	destPath := filepath.Join(d.targetDir, dep.Name)

	// Remove existing directory if exists
	if err := os.RemoveAll(destPath); err != nil {
		return nil, fmt.Errorf("failed to remove existing directory: %w", err)
	}

	// Clone repository
	args := []string{"clone", "--depth", "1"}

	// Add branch/tag if specified
	if dep.Source.Ref != "" {
		args = append(args, "--branch", dep.Source.Ref)
	} else if dep.Version != "" && dep.Version != "latest" {
		args = append(args, "--branch", dep.Version)
	}

	args = append(args, dep.Source.URL, destPath)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w", err)
	}

	// Get commit hash
	version, err := d.getGitCommitHash(destPath)
	if err != nil {
		d.log.Warn("Failed to get commit hash", logger.Error(err))
		version = dep.Version
		if version == "" {
			version = "unknown"
		}
	}

	// Calculate hash of content
	hash, err := d.calculateDirHash(destPath)
	if err != nil {
		d.log.Warn("Failed to calculate hash", logger.Error(err))
		hash = ""
	}

	// Determine proto path
	protoPath := destPath
	if dep.SubPath != "" {
		protoPath = filepath.Join(destPath, dep.SubPath)
	}

	d.log.Info("Successfully cloned",
		logger.String("name", dep.Name),
		logger.String("version", version))

	return &DownloadResult{
		Name:      dep.Name,
		Version:   version,
		LocalPath: destPath,
		ProtoPath: protoPath,
		Hash:      hash,
	}, nil
}

// downloadURL downloads from HTTP/HTTPS URL (archive).
func (d *Downloader) downloadURL(ctx context.Context, dep Dependency) (*DownloadResult, error) {
	return nil, fmt.Errorf("URL downloads not yet implemented")
}

// copyLocal copies from local filesystem.
func (d *Downloader) copyLocal(dep Dependency) (*DownloadResult, error) {
	d.log.Info("Copying local dependency", logger.String("path", dep.Source.Path))

	sourcePath := dep.Source.Path
	destPath := filepath.Join(d.targetDir, dep.Name)

	// Remove existing directory if exists
	if err := os.RemoveAll(destPath); err != nil {
		return nil, fmt.Errorf("failed to remove existing directory: %w", err)
	}

	// Copy directory
	if err := d.copyDir(sourcePath, destPath); err != nil {
		return nil, fmt.Errorf("failed to copy directory: %w", err)
	}

	// Calculate hash
	hash, err := d.calculateDirHash(destPath)
	if err != nil {
		d.log.Warn("Failed to calculate hash", logger.Error(err))
		hash = ""
	}

	protoPath := destPath
	if dep.SubPath != "" {
		protoPath = filepath.Join(destPath, dep.SubPath)
	}

	return &DownloadResult{
		Name:      dep.Name,
		Version:   "local",
		LocalPath: destPath,
		ProtoPath: protoPath,
		Hash:      hash,
	}, nil
}

// getGitCommitHash gets the current commit hash.
func (d *Downloader) getGitCommitHash(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// calculateDirHash calculates SHA256 hash of directory contents.
func (d *Downloader) calculateDirHash(dir string) (string, error) {
	hash := sha256.New()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Only hash proto files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(hash, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// copyDir recursively copies a directory.
func (d *Downloader) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file
		return d.copyFile(path, targetPath)
	})
}

// copyFile copies a single file.
func (d *Downloader) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
