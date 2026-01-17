# Buffalo Configuration Guide

## Overview

Buffalo uses YAML configuration files (`buffalo.yaml`) for project settings. This guide covers configuration structure, validation, and best practices.

## Quick Start

```bash
# Initialize new configuration
buffalo config init

# Validate existing configuration
buffalo config validate

# Show configuration summary
buffalo config show
```

## Configuration File Structure

### Complete Example

```yaml
# buffalo.yaml - Complete Configuration Reference

# Project Information
project:
  name: "my-api"
  version: "1.0.0"

# Proto File Settings
proto:
  paths:
    - "./protos"
    - "./vendor/protos"
  exclude:
    - "**/*_test.proto"
    - "**/internal/*.proto"
  import_paths:
    - "./protos"
    - ".buffalo/depends"

# Output Settings
output:
  base_dir: "./generated"
  preserve_proto_structure: true
  directories:
    python: "python"
    go: "go"
    rust: "rust"
    cpp: "cpp"

# Language-Specific Settings
languages:
  python:
    enabled: true
    package: "my_api"
    generator: "grpcio-tools"
  go:
    enabled: true
    module: "github.com/myorg/my-api"
    generator: "protoc-gen-go"
  rust:
    enabled: false
    generator: "tonic"
  cpp:
    enabled: false
    namespace: "my_api"

# Build Settings
build:
  workers: 4
  incremental: true
  cache:
    enabled: true
    directory: ".buffalo/cache"

# Versioning
versioning:
  enabled: false
  strategy: "hash"
  output_format: "directory"
  keep_versions: 5

# Logging
logging:
  level: "info"
  format: "text"
  output: "stdout"

# External Dependencies
dependencies:
  - name: "googleapis"
    url: "https://github.com/googleapis/googleapis.git"
    paths:
      - "google/api"
      - "google/rpc"

# Plugins
plugins:
  - name: "naming-validator"
    enabled: true
    hooks:
      - "post-parse"
    priority: 100
    config:
      naming_convention: "snake_case"

# Templates
templates:
  - name: "custom-go"
    language: "go"
    path: "./templates/go"
    patterns:
      - "**/*.tmpl"
    vars:
      packagePrefix: "github.com/myorg"
```

## Configuration Sections

### project

```yaml
project:
  name: "my-project"     # Project name (used in logs, reports)
  version: "1.0.0"       # Project version
```

### proto

```yaml
proto:
  paths:                 # Directories containing .proto files
    - "./protos"
  exclude:               # Glob patterns for files to exclude
    - "**/*_test.proto"
  import_paths:          # Additional import paths for protoc
    - "./protos"
    - "./vendor"
```

### output

```yaml
output:
  base_dir: "./generated"              # Root output directory
  preserve_proto_structure: true       # Keep proto directory structure
  directories:                         # Language-specific subdirectories
    python: "python"
    go: "go"
    rust: "rust"
    cpp: "cpp"
```

### languages

Each language has specific settings:

#### Python
```yaml
languages:
  python:
    enabled: true
    package: "my_package"      # Python package name
    generator: "grpcio-tools"  # grpcio-tools | betterproto
```

#### Go
```yaml
languages:
  go:
    enabled: true
    module: "github.com/org/project"  # Go module path
    generator: "protoc-gen-go"        # protoc-gen-go | protoc-gen-go-grpc
```

#### Rust
```yaml
languages:
  rust:
    enabled: false
    generator: "tonic"         # tonic | prost
```

#### C++
```yaml
languages:
  cpp:
    enabled: false
    namespace: "myproject"     # C++ namespace
```

### build

```yaml
build:
  workers: 4                   # Parallel workers (0 = auto)
  incremental: true            # Only rebuild changed files
  cache:
    enabled: true              # Enable compilation cache
    directory: ".buffalo/cache"
```

### versioning

```yaml
versioning:
  enabled: false               # Enable version management
  strategy: "hash"             # hash | timestamp | semantic | git
  output_format: "directory"   # directory | suffix
  keep_versions: 5             # Number of versions to keep (0 = all)
```

### plugins

```yaml
plugins:
  - name: "my-plugin"
    enabled: true
    hooks:
      - "pre-build"
      - "post-build"
    priority: 100              # Lower = earlier execution
    config:
      option1: "value1"
      option2: true
```

### templates

```yaml
templates:
  - name: "custom-template"
    language: "go"
    path: "./templates"
    patterns:
      - "*.tmpl"
      - "**/*.tpl"
    vars:
      customVar: "value"
```

## Config Commands

### config validate

Validate configuration file:

```bash
# Validate buffalo.yaml
buffalo config validate

# Validate specific file
buffalo config validate --config my-config.yaml
```

Output includes:
- ✅ Valid settings
- ⚠️ Warnings (non-critical issues)
- ❌ Errors (must be fixed)
- 💡 Suggestions for improvements

### config show

Display configuration summary:

```bash
# Text format (default)
buffalo config show

# YAML format
buffalo config show --format yaml
```

### config init

Create new configuration:

```bash
# Create buffalo.yaml with defaults
buffalo config init

# Overwrite existing
buffalo config init --force
```

## Validation Checks

Buffalo validates:

| Check | Severity | Description |
|-------|----------|-------------|
| YAML Syntax | Error | Valid YAML format |
| output.base_dir | Error | Required field |
| proto.paths | Error | At least one path |
| Path exists | Error | Proto paths exist |
| Languages | Warning | At least one enabled |
| go.module | Warning | Required for Go |
| build.workers | Info | Recommended setting |
| Plugin names | Error | Must be non-empty |
| Template paths | Warning | Should exist |

## Environment Variables

Override config values with environment variables:

```bash
export BUFFALO_OUTPUT_BASE_DIR="./out"
export BUFFALO_BUILD_WORKERS=8
export BUFFALO_LOGGING_LEVEL="debug"
```

## Best Practices

1. **Use snake_case** for field names (not camelCase)
2. **Set workers** to number of CPU cores
3. **Enable caching** for faster builds
4. **Use relative paths** for portability
5. **Exclude test files** to speed up builds
6. **Define import_paths** for external proto dependencies

## Troubleshooting

### "output.base_dir is required"

```yaml
# ❌ Wrong
output:
  basedir: "./gen"

# ✅ Correct
output:
  base_dir: "./gen"
```

### "No proto paths defined"

```yaml
# Add proto.paths
proto:
  paths:
    - "./protos"
```

### "Path does not exist"

Create the directory or update the path:

```bash
mkdir -p protos
```

## See Also

- [Templates Guide](TEMPLATES.md)
- [Metrics Guide](METRICS.md)
- [Plugin Guide](PLUGIN_GUIDE.md)
