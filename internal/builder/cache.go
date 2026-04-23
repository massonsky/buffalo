package builder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/tracing"
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

	// ToolsHash captures the version pin of every code-generation tool the
	// entry was produced with. A change here invalidates the cache so users
	// never get stale output after upgrading protoc / protoc-gen-* binaries.
	ToolsHash string

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
	cacheDir  string
	toolsHash string
	log       Logger
}

// NewCacheManager creates a new CacheManager
func NewCacheManager(log Logger) CacheManager {
	return &cacheManager{
		cacheDir: ".buffalo-cache",
		log:      log,
	}
}

// NewCacheManagerWithTools creates a CacheManager that mixes the supplied
// tool versions into every cache entry so codegen output is invalidated when a
// pinned binary changes. Pass an empty map to disable the behavior.
func NewCacheManagerWithTools(log Logger, tools map[string]string) CacheManager {
	return &cacheManager{
		cacheDir:  ".buffalo-cache",
		toolsHash: hashTools(tools),
		log:       log,
	}
}

// hashTools produces a deterministic SHA256 of the (name, version) pairs.
func hashTools(tools map[string]string) string {
	if len(tools) == 0 {
		return ""
	}
	keys := make([]string, 0, len(tools))
	for k := range tools {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(tools[k])
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
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
	ctx, span := tracing.StartSpan(ctx, "cache.lookup", tracing.WithAttributes(map[string]any{
		"file": file.Path,
	}))
	_ = ctx
	defer span.End()

	cacheFile := c.getCacheFilePath(file.Path)

	if !utils.FileExists(cacheFile) {
		span.SetAttribute("hit", false)
		span.SetAttribute("miss_reason", "absent")
		return nil, nil
	}

	data, err := utils.ReadFile(cacheFile)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, errors.ErrCache, "failed to read cache file")
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, errors.ErrCache, "failed to unmarshal cache entry")
	}

	// Verify hash
	currentHash, err := c.computeFileHash(file.Path)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if entry.Hash != currentHash {
		c.log.Debug("Cache miss: hash mismatch", "file", file.Path)
		span.SetAttribute("hit", false)
		span.SetAttribute("miss_reason", "file_hash")
		return nil, nil
	}

	// Tool-version mismatch invalidates the entry: regenerated code must
	// reflect the currently pinned protoc / plugin binaries.
	if entry.ToolsHash != c.toolsHash {
		c.log.Debug("Cache miss: tool versions changed", "file", file.Path)
		span.SetAttribute("hit", false)
		span.SetAttribute("miss_reason", "tools_hash")
		return nil, nil
	}

	// Verify that generated files still exist
	for _, genFile := range entry.GeneratedFiles {
		if !utils.FileExists(genFile) {
			c.log.Debug("Cache miss: generated file missing", "file", file.Path, "missing", genFile)
			span.SetAttribute("hit", false)
			span.SetAttribute("miss_reason", "generated_missing")
			return nil, nil
		}
	}

	c.log.Debug("Cache hit", "file", file.Path)
	span.SetAttribute("hit", true)
	return &entry, nil
}

// Put stores cache entry
func (c *cacheManager) Put(ctx context.Context, entry *CacheEntry) error {
	if err := utils.EnsureDir(c.cacheDir); err != nil {
		return errors.Wrap(err, errors.ErrCache, "failed to create cache directory")
	}

	// Stamp the current tool fingerprint so future Get calls can detect a
	// drift after upgrades.
	if entry.ToolsHash == "" {
		entry.ToolsHash = c.toolsHash
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
