package builder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/utils"
)

// CacheEntry represents a cached build result
type CacheEntry struct {
	// File is the proto file path
	File string

	// Hash is the file content hash
	Hash string

	// DepsHash is the dependencies hash
	DepsHash string

	// Languages are the generated languages
	Languages []string

	// GeneratedFiles are the generated file paths
	GeneratedFiles []string
}

// CacheManager manages build cache
type CacheManager interface {
	// Check checks cache for files
	Check(ctx context.Context, files []*ProtoFile) (hits int, misses int)

	// Get retrieves cache entry for a file
	Get(ctx context.Context, file *ProtoFile) (*CacheEntry, error)

	// Put stores cache entry
	Put(ctx context.Context, entry *CacheEntry) error

	// Invalidate removes cache entry
	Invalidate(ctx context.Context, file string) error

	// Clear clears all cache
	Clear(ctx context.Context) error
}

// cacheManager implements CacheManager
type cacheManager struct {
	cacheDir string
	log      Logger
}

// NewCacheManager creates a new CacheManager
func NewCacheManager(log Logger) CacheManager {
	return &cacheManager{
		cacheDir: ".buffalo-cache",
		log:      log,
	}
}

// Check checks cache for files
func (c *cacheManager) Check(ctx context.Context, files []*ProtoFile) (hits int, misses int) {
	for _, file := range files {
		entry, err := c.Get(ctx, file)
		if err != nil || entry == nil {
			misses++
		} else {
			hits++
		}
	}
	return hits, misses
}

// Get retrieves cache entry
func (c *cacheManager) Get(ctx context.Context, file *ProtoFile) (*CacheEntry, error) {
	cacheFile := c.getCacheFilePath(file.Path)

	if !utils.FileExists(cacheFile) {
		return nil, nil
	}

	data, err := utils.ReadFile(cacheFile)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCache, "failed to read cache file")
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, errors.Wrap(err, errors.ErrCache, "failed to unmarshal cache entry")
	}

	// Verify hash
	currentHash, err := c.computeFileHash(file.Path)
	if err != nil {
		return nil, err
	}

	if entry.Hash != currentHash {
		c.log.Debug("Cache miss: hash mismatch", "file", file.Path)
		return nil, nil
	}

	// Verify that generated files still exist
	for _, genFile := range entry.GeneratedFiles {
		if !utils.FileExists(genFile) {
			c.log.Debug("Cache miss: generated file missing", "file", file.Path, "missing", genFile)
			return nil, nil
		}
	}

	c.log.Debug("Cache hit", "file", file.Path)
	return &entry, nil
}

// Put stores cache entry
func (c *cacheManager) Put(ctx context.Context, entry *CacheEntry) error {
	if err := utils.EnsureDir(c.cacheDir); err != nil {
		return errors.Wrap(err, errors.ErrCache, "failed to create cache directory")
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.ErrCache, "failed to marshal cache entry")
	}

	cacheFile := c.getCacheFilePath(entry.File)
	if err := utils.WriteFile(cacheFile, data); err != nil {
		return errors.Wrap(err, errors.ErrCache, "failed to write cache file")
	}

	c.log.Debug("Cache entry stored", "file", entry.File)
	return nil
}

// Invalidate removes cache entry
func (c *cacheManager) Invalidate(ctx context.Context, file string) error {
	cacheFile := c.getCacheFilePath(file)
	if utils.FileExists(cacheFile) {
		if err := os.Remove(cacheFile); err != nil {
			return errors.Wrap(err, errors.ErrCache, "failed to remove cache file")
		}
	}
	return nil
}

// Clear clears all cache
func (c *cacheManager) Clear(ctx context.Context) error {
	if utils.FileExists(c.cacheDir) {
		if err := utils.RemoveDir(c.cacheDir); err != nil {
			return errors.Wrap(err, errors.ErrCache, "failed to clear cache")
		}
	}
	return nil
}

// getCacheFilePath returns the cache file path for a proto file
func (c *cacheManager) getCacheFilePath(protoFile string) string {
	hash := sha256.Sum256([]byte(protoFile))
	filename := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(c.cacheDir, filename)
}

// computeFileHash computes SHA256 hash of a file
func (c *cacheManager) computeFileHash(path string) (string, error) {
	data, err := utils.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to read file for hashing")
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
