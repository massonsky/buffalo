package utils

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher represents a file system watcher
type FileWatcher struct {
	watcher *fsnotify.Watcher
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &FileWatcher{
		watcher: watcher,
	}, nil
}

// Add adds a path to watch
func (fw *FileWatcher) Add(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := fw.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to add path to watcher: %w", err)
	}

	return nil
}

// Remove removes a path from watching
func (fw *FileWatcher) Remove(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := fw.watcher.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove path from watcher: %w", err)
	}

	return nil
}

// Events returns the channel for file system events
func (fw *FileWatcher) Events() <-chan fsnotify.Event {
	return fw.watcher.Events
}

// Errors returns the channel for watcher errors
func (fw *FileWatcher) Errors() <-chan error {
	return fw.watcher.Errors
}

// Close closes the watcher
func (fw *FileWatcher) Close() error {
	return fw.watcher.Close()
}

// FindExecutable finds an executable in the system PATH
func FindExecutable(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("executable '%s' not found in PATH", name)
	}
	return path, nil
}

// ExecCommand executes a command and returns its output
func ExecCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}
