# Buffalo Build Metrics Guide

## Overview

Buffalo collects detailed metrics during builds to help you optimize performance, track cache efficiency, and identify bottlenecks.

## Quick Start

```bash
# Show last build metrics
buffalo metrics show

# Show build history
buffalo metrics history

# Build with metrics enabled (metrics are always collected)
buffalo build
```

## Metrics Commands

### metrics show

Display metrics from the most recent build:

```bash
buffalo metrics show
```

Output example:
```
📊 Build Metrics
═══════════════════════════════════════════════════════════

🆔 Build ID:     build-1737117647
⏱️  Duration:     0.34 seconds
📅 Start:        2026-01-17T13:30:47+03:00
📅 End:          2026-01-17T13:30:47+03:00

📁 Files
───────────────────────────────────────────────────────────
  Total Proto:     4
  Processed:       4
  Generated:       8
  Skipped:         0
  Failed:          0

📈 Performance
───────────────────────────────────────────────────────────
  Files/sec:       11.76
  Bytes/sec:       45.23 KB/s
  Avg File Time:   42.31 ms
  Input Size:      12.45 KB
  Output Size:     15.67 KB

🔄 Cache
───────────────────────────────────────────────────────────
  Hits:            0
  Misses:          4
  Hit Rate:        0.0%

⚙️  Build Options
───────────────────────────────────────────────────────────
  Workers:         4
  Incremental:     true
  Cache Enabled:   true

🌐 Languages
───────────────────────────────────────────────────────────
  python: 4 files, 7.83 KB
  go: 4 files, 7.84 KB
```

### metrics history

Show metrics from recent builds:

```bash
# Show last 10 builds
buffalo metrics history

# Show last 5 builds
buffalo metrics history --limit 5
```

Output example:
```
📊 Build History (last 3 builds)

✅ 1. build-1737117647
   Duration: 0.34s | Files: 4 processed, 8 generated
   Cache: 0.0% hit rate | Errors: 0, Warnings: 0

✅ 2. build-1737117520
   Duration: 0.28s | Files: 4 processed, 8 generated
   Cache: 75.0% hit rate | Errors: 0, Warnings: 0

⚠️ 3. build-1737117412
   Duration: 0.45s | Files: 4 processed, 6 generated
   Cache: 50.0% hit rate | Errors: 0, Warnings: 2
```

## Collected Metrics

### Build Identification

| Metric | Description |
|--------|-------------|
| `build_id` | Unique build identifier |
| `start_time` | Build start timestamp |
| `end_time` | Build end timestamp |
| `duration_seconds` | Total build duration |

### File Metrics

| Metric | Description |
|--------|-------------|
| `total_proto_files` | Total proto files found |
| `processed_files` | Files that were processed |
| `generated_files` | Output files generated |
| `skipped_files` | Files skipped (cache hit) |
| `failed_files` | Files that failed to compile |
| `total_input_bytes` | Total size of input files |
| `total_output_bytes` | Total size of generated files |

### Performance Metrics

| Metric | Description |
|--------|-------------|
| `files_per_second` | Compilation throughput |
| `bytes_per_second` | Output generation rate |
| `average_file_time_ms` | Average time per file |

### Cache Metrics

| Metric | Description |
|--------|-------------|
| `cache_hits` | Files served from cache |
| `cache_misses` | Files that needed recompilation |
| `cache_hit_rate` | Percentage of cache hits |

### Language Breakdown

Per-language metrics:
- `files_generated` - Files generated for this language
- `bytes_generated` - Output size for this language
- `duration_seconds` - Time spent on this language
- `errors` - Errors for this language

### Error Metrics

| Metric | Description |
|--------|-------------|
| `error_count` | Total errors during build |
| `warning_count` | Total warnings during build |
| `errors` | List of error messages |

## Metrics Storage

Metrics are stored in JSON format:

```
.buffalo/
├── cache/
│   └── metrics/
│       ├── latest.json        # Most recent build
│       ├── metrics-build-1737117647.json
│       ├── metrics-build-1737117520.json
│       └── metrics-build-1737117412.json
```

### JSON Format

```json
{
  "build_id": "build-1737117647",
  "start_time": "2026-01-17T13:30:47+03:00",
  "end_time": "2026-01-17T13:30:47+03:00",
  "duration_seconds": 0.34,
  "total_proto_files": 4,
  "processed_files": 4,
  "generated_files": 8,
  "skipped_files": 0,
  "failed_files": 0,
  "total_input_bytes": 12450,
  "total_output_bytes": 15670,
  "files_per_second": 11.76,
  "bytes_per_second": 46117.65,
  "average_file_time_ms": 42.31,
  "cache_hits": 0,
  "cache_misses": 4,
  "cache_hit_rate": 0.0,
  "language_metrics": {
    "python": {
      "language": "python",
      "files_generated": 4,
      "bytes_generated": 7834,
      "duration_seconds": 0.15,
      "errors": 0
    },
    "go": {
      "language": "go",
      "files_generated": 4,
      "bytes_generated": 7836,
      "duration_seconds": 0.17,
      "errors": 0
    }
  },
  "error_count": 0,
  "warning_count": 0,
  "workers": 4,
  "incremental": true,
  "cache_enabled": true
}
```

## Performance Optimization

### Improving Build Speed

1. **Increase workers**
   ```yaml
   build:
     workers: 8  # Match CPU cores
   ```

2. **Enable caching**
   ```yaml
   build:
     cache:
       enabled: true
   ```

3. **Use incremental builds**
   ```yaml
   build:
     incremental: true
   ```

### Improving Cache Hit Rate

- Avoid touching proto files unnecessarily
- Use `buffalo build --incremental` (default)
- Check cache statistics after builds

### Reducing Generated File Size

- Use `optimize_for = LITE_RUNTIME` in proto files
- Exclude unused protos with `proto.exclude`

## Integration

### CI/CD Metrics

Export metrics for CI/CD dashboards:

```bash
# Get metrics as JSON
buffalo metrics show --format json > build-metrics.json

# Upload to metrics service
curl -X POST https://metrics.example.com/api/builds \
  -H "Content-Type: application/json" \
  -d @build-metrics.json
```

### Monitoring

Track metrics over time:
- Build duration trends
- Cache efficiency
- Error rates
- File generation patterns

## Troubleshooting

### No metrics found

```
No build metrics found
Run 'buffalo build --metrics' to collect build metrics
```

Metrics are collected automatically during `buffalo build`. If no metrics exist:
1. Run a build: `buffalo build`
2. Check metrics directory exists: `.buffalo/cache/metrics/`

### Low cache hit rate

If cache hit rate is consistently 0%:
1. Ensure caching is enabled: `build.cache.enabled: true`
2. Check cache directory permissions
3. Verify files aren't being modified between builds

## See Also

- [Configuration Guide](CONFIG_GUIDE.md)
- [CI/CD Guide](CI_CD_GUIDE.md)
- [Plugin Guide](PLUGIN_GUIDE.md)
