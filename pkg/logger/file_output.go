package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileOutput writes logs to a file with optional rotation.
type FileOutput struct {
	filename   string
	file       *os.File
	mutex      sync.Mutex
	rotation   *RotationConfig
	currentDay string
}

// RotationConfig configures log file rotation.
type RotationConfig struct {
	// MaxSize is the maximum size in megabytes before rotation.
	MaxSize int
	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int
	// Compress determines if rotated files should be compressed.
	Compress bool
	// Daily enables daily rotation.
	Daily bool
}

// NewFileOutput creates a new file output.
func NewFileOutput(filename string) (*FileOutput, error) {
	return NewFileOutputWithRotation(filename, nil)
}

// NewFileOutputWithRotation creates a new file output with rotation.
func NewFileOutputWithRotation(filename string, rotation *RotationConfig) (*FileOutput, error) {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	output := &FileOutput{
		filename:   filename,
		file:       file,
		rotation:   rotation,
		currentDay: time.Now().Format("2006-01-02"),
	}

	return output, nil
}

// Write writes the log entry to the file.
func (o *FileOutput) Write(p []byte) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Check if rotation is needed
	if o.rotation != nil {
		if err := o.checkRotation(); err != nil {
			return err
		}
	}

	_, err := o.file.Write(p)
	return err
}

// Close closes the file output.
func (o *FileOutput) Close() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.file != nil {
		return o.file.Close()
	}
	return nil
}

// checkRotation checks if the log file needs to be rotated.
func (o *FileOutput) checkRotation() error {
	if o.rotation == nil {
		return nil
	}

	// Check daily rotation
	if o.rotation.Daily {
		today := time.Now().Format("2006-01-02")
		if today != o.currentDay {
			if err := o.rotate(); err != nil {
				return err
			}
			o.currentDay = today
		}
	}

	// Check size-based rotation
	if o.rotation.MaxSize > 0 {
		info, err := o.file.Stat()
		if err != nil {
			return err
		}

		// Convert MaxSize from MB to bytes
		maxBytes := int64(o.rotation.MaxSize) * 1024 * 1024
		if info.Size() >= maxBytes {
			return o.rotate()
		}
	}

	return nil
}

// rotate rotates the log file.
func (o *FileOutput) rotate() error {
	// Close current file
	if err := o.file.Close(); err != nil {
		return err
	}

	// Rename current file
	timestamp := time.Now().Format("20060102-150405")
	rotatedName := fmt.Sprintf("%s.%s", o.filename, timestamp)

	if err := os.Rename(o.filename, rotatedName); err != nil {
		return err
	}

	// Open new file
	file, err := os.OpenFile(o.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	o.file = file

	// Clean up old files
	go o.cleanupOldFiles()

	return nil
}

// cleanupOldFiles removes old log files based on rotation config.
func (o *FileOutput) cleanupOldFiles() {
	if o.rotation == nil {
		return
	}

	dir := filepath.Dir(o.filename)
	base := filepath.Base(o.filename)
	pattern := base + ".*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Sort by modification time
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// Remove old files based on MaxAge
	if o.rotation.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -o.rotation.MaxAge)
		for _, file := range files {
			if file.modTime.Before(cutoff) {
				os.Remove(file.path)
			}
		}
	}

	// Remove excess files based on MaxBackups
	if o.rotation.MaxBackups > 0 && len(files) > o.rotation.MaxBackups {
		// Sort by modTime (oldest first)
		for i := 0; i < len(files)-1; i++ {
			for j := i + 1; j < len(files); j++ {
				if files[i].modTime.After(files[j].modTime) {
					files[i], files[j] = files[j], files[i]
				}
			}
		}

		// Remove oldest files
		for i := 0; i < len(files)-o.rotation.MaxBackups; i++ {
			os.Remove(files[i].path)
		}
	}
}
