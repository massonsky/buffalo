package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BuildMetrics contains metrics collected during a build
type BuildMetrics struct {
	// Build identification
	BuildID   string    `json:"build_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration_seconds"`

	// File metrics
	TotalProtoFiles  int   `json:"total_proto_files"`
	ProcessedFiles   int   `json:"processed_files"`
	GeneratedFiles   int   `json:"generated_files"`
	SkippedFiles     int   `json:"skipped_files"`
	FailedFiles      int   `json:"failed_files"`
	TotalInputBytes  int64 `json:"total_input_bytes"`
	TotalOutputBytes int64 `json:"total_output_bytes"`

	// Performance metrics
	FilesPerSecond  float64 `json:"files_per_second"`
	BytesPerSecond  float64 `json:"bytes_per_second"`
	AverageFileTime float64 `json:"average_file_time_ms"`

	// Cache metrics
	CacheHits    int     `json:"cache_hits"`
	CacheMisses  int     `json:"cache_misses"`
	CacheHitRate float64 `json:"cache_hit_rate"`

	// Language breakdown
	LanguageMetrics map[string]*LanguageMetrics `json:"language_metrics"`

	// Error metrics
	ErrorCount   int      `json:"error_count"`
	WarningCount int      `json:"warning_count"`
	Errors       []string `json:"errors,omitempty"`

	// Memory metrics (estimated)
	PeakMemoryMB float64 `json:"peak_memory_mb"`

	// Build options
	Workers      int  `json:"workers"`
	Incremental  bool `json:"incremental"`
	CacheEnabled bool `json:"cache_enabled"`
}

// LanguageMetrics contains metrics for a specific language
type LanguageMetrics struct {
	Language       string  `json:"language"`
	FilesGenerated int     `json:"files_generated"`
	BytesGenerated int64   `json:"bytes_generated"`
	Duration       float64 `json:"duration_seconds"`
	Errors         int     `json:"errors"`
}

// Collector collects build metrics
type Collector struct {
	mu          sync.Mutex
	metrics     *BuildMetrics
	fileTimings map[string]time.Duration
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		metrics: &BuildMetrics{
			BuildID:         fmt.Sprintf("build-%d", time.Now().Unix()),
			StartTime:       time.Now(),
			LanguageMetrics: make(map[string]*LanguageMetrics),
		},
		fileTimings: make(map[string]time.Duration),
	}
}

// SetBuildOptions sets build configuration options
func (c *Collector) SetBuildOptions(workers int, incremental, cacheEnabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.Workers = workers
	c.metrics.Incremental = incremental
	c.metrics.CacheEnabled = cacheEnabled
}

// SetTotalFiles sets the total number of proto files
func (c *Collector) SetTotalFiles(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.TotalProtoFiles = count
}

// RecordFileProcessed records a processed file
func (c *Collector) RecordFileProcessed(file string, inputBytes int64, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.ProcessedFiles++
	c.metrics.TotalInputBytes += inputBytes
	c.fileTimings[file] = duration
}

// RecordFileGenerated records a generated file
func (c *Collector) RecordFileGenerated(language string, file string, bytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.GeneratedFiles++
	c.metrics.TotalOutputBytes += bytes

	if c.metrics.LanguageMetrics[language] == nil {
		c.metrics.LanguageMetrics[language] = &LanguageMetrics{
			Language: language,
		}
	}
	c.metrics.LanguageMetrics[language].FilesGenerated++
	c.metrics.LanguageMetrics[language].BytesGenerated += bytes
}

// RecordFileSkipped records a skipped file (cache hit)
func (c *Collector) RecordFileSkipped(file string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.SkippedFiles++
	c.metrics.CacheHits++
}

// RecordFileFailed records a failed file
func (c *Collector) RecordFileFailed(file string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.FailedFiles++
	c.metrics.ErrorCount++
	c.metrics.Errors = append(c.metrics.Errors, fmt.Sprintf("%s: %v", file, err))
}

// RecordCacheMiss records a cache miss
func (c *Collector) RecordCacheMiss() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.CacheMisses++
}

// RecordWarning records a warning
func (c *Collector) RecordWarning(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.WarningCount++
}

// RecordError records an error
func (c *Collector) RecordError(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics.ErrorCount++
	c.metrics.Errors = append(c.metrics.Errors, msg)
}

// Finalize calculates final metrics
func (c *Collector) Finalize() *BuildMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.EndTime = time.Now()
	c.metrics.Duration = c.metrics.EndTime.Sub(c.metrics.StartTime).Seconds()

	// Calculate performance metrics
	if c.metrics.Duration > 0 {
		c.metrics.FilesPerSecond = float64(c.metrics.ProcessedFiles) / c.metrics.Duration
		c.metrics.BytesPerSecond = float64(c.metrics.TotalOutputBytes) / c.metrics.Duration
	}

	// Calculate average file time
	if len(c.fileTimings) > 0 {
		var totalTime time.Duration
		for _, d := range c.fileTimings {
			totalTime += d
		}
		c.metrics.AverageFileTime = float64(totalTime.Milliseconds()) / float64(len(c.fileTimings))
	}

	// Calculate cache hit rate
	totalCacheOperations := c.metrics.CacheHits + c.metrics.CacheMisses
	if totalCacheOperations > 0 {
		c.metrics.CacheHitRate = float64(c.metrics.CacheHits) / float64(totalCacheOperations) * 100
	}

	return c.metrics
}

// GetMetrics returns the current metrics
func (c *Collector) GetMetrics() *BuildMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}

// Store stores metrics to file
type Store struct {
	dir string
}

// NewStore creates a new metrics store
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metrics directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Save saves metrics to file
func (s *Store) Save(metrics *BuildMetrics) error {
	filename := fmt.Sprintf("metrics-%s.json", metrics.BuildID)
	path := filepath.Join(s.dir, filename)

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	// Also update latest.json
	latestPath := filepath.Join(s.dir, "latest.json")
	if err := os.WriteFile(latestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write latest metrics: %w", err)
	}

	return nil
}

// LoadLatest loads the most recent metrics
func (s *Store) LoadLatest() (*BuildMetrics, error) {
	path := filepath.Join(s.dir, "latest.json")
	return s.loadFromFile(path)
}

// LoadByID loads metrics by build ID
func (s *Store) LoadByID(buildID string) (*BuildMetrics, error) {
	filename := fmt.Sprintf("metrics-%s.json", buildID)
	path := filepath.Join(s.dir, filename)
	return s.loadFromFile(path)
}

// List returns all stored metrics
func (s *Store) List(limit int) ([]*BuildMetrics, error) {
	pattern := filepath.Join(s.dir, "metrics-*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics files: %w", err)
	}

	// Sort by modification time (newest first)
	// TODO: Implement proper sorting

	metrics := make([]*BuildMetrics, 0)
	for i, file := range files {
		if limit > 0 && i >= limit {
			break
		}
		m, err := s.loadFromFile(file)
		if err != nil {
			continue
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (s *Store) loadFromFile(path string) (*BuildMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics file: %w", err)
	}

	var metrics BuildMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &metrics, nil
}

// FormatMetrics formats metrics for display
func FormatMetrics(m *BuildMetrics) string {
	output := fmt.Sprintf(`
📊 Build Metrics
═══════════════════════════════════════════════════════════

🆔 Build ID:     %s
⏱️  Duration:     %.2f seconds
📅 Start:        %s
📅 End:          %s

📁 Files
───────────────────────────────────────────────────────────
  Total Proto:     %d
  Processed:       %d
  Generated:       %d
  Skipped:         %d
  Failed:          %d

📈 Performance
───────────────────────────────────────────────────────────
  Files/sec:       %.2f
  Bytes/sec:       %.2f KB/s
  Avg File Time:   %.2f ms
  Input Size:      %.2f KB
  Output Size:     %.2f KB

🔄 Cache
───────────────────────────────────────────────────────────
  Hits:            %d
  Misses:          %d
  Hit Rate:        %.1f%%

⚙️  Build Options
───────────────────────────────────────────────────────────
  Workers:         %d
  Incremental:     %v
  Cache Enabled:   %v
`,
		m.BuildID,
		m.Duration,
		m.StartTime.Format(time.RFC3339),
		m.EndTime.Format(time.RFC3339),
		m.TotalProtoFiles,
		m.ProcessedFiles,
		m.GeneratedFiles,
		m.SkippedFiles,
		m.FailedFiles,
		m.FilesPerSecond,
		m.BytesPerSecond/1024,
		m.AverageFileTime,
		float64(m.TotalInputBytes)/1024,
		float64(m.TotalOutputBytes)/1024,
		m.CacheHits,
		m.CacheMisses,
		m.CacheHitRate,
		m.Workers,
		m.Incremental,
		m.CacheEnabled,
	)

	// Language breakdown
	if len(m.LanguageMetrics) > 0 {
		output += "\n🌐 Languages\n───────────────────────────────────────────────────────────\n"
		for lang, lm := range m.LanguageMetrics {
			output += fmt.Sprintf("  %s: %d files, %.2f KB\n", lang, lm.FilesGenerated, float64(lm.BytesGenerated)/1024)
		}
	}

	// Errors
	if m.ErrorCount > 0 {
		output += fmt.Sprintf("\n❌ Errors: %d\n", m.ErrorCount)
	}
	if m.WarningCount > 0 {
		output += fmt.Sprintf("⚠️  Warnings: %d\n", m.WarningCount)
	}

	return output
}
