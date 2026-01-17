# 🎮 Buffalo CLI Commands Reference

Complete guide to all Buffalo CLI commands and options.

## 📋 Table of Contents

- [Core Commands](#core-commands)
  - [build](#build)
  - [rebuild](#rebuild)
  - [watch](#watch)
- [Code Quality](#code-quality)
  - [lint](#lint)
  - [format](#format)
  - [validate](#validate)
- [Project Management](#project-management)
  - [init](#init)
  - [check](#check)
  - [list](#list)
  - [deps](#deps)
  - [stats](#stats)
  - [clear](#clear)
- [Utility Commands](#utility-commands)
  - [version](#version)
  - [completion](#completion)
- [Global Flags](#global-flags)

---

## Core Commands

### build

Build protobuf files and generate code for specified languages.

**Usage:**
```bash
buffalo build [flags]
```

**Flags:**
- `-o, --output <dir>` - Output directory (default: "./generated")
- `-l, --lang <langs>` - Target languages: python,go,rust,cpp
- `-p, --proto-path <paths>` - Paths to search for proto files
- `--dry-run` - Show what would be built without building

**Examples:**
```bash
# Build with default config
buffalo build

# Build for specific languages
buffalo build --lang python,go

# Build with custom output
buffalo build --output ./gen --lang python

# Dry run
buffalo build --dry-run
```

---

### rebuild

Force a complete rebuild, ignoring cache.

**Usage:**
```bash
buffalo rebuild [flags]
```

**Flags:**
- `-o, --output <dir>` - Output directory (default: "./generated")
- `-l, --lang <langs>` - Target languages
- `-f, --force` - Force rebuild (always true)

**Examples:**
```bash
# Full rebuild
buffalo rebuild

# Rebuild specific languages
buffalo rebuild --lang go,rust
```

**What it does:**
1. Clears build cache
2. Deletes generated output
3. Performs fresh build
4. Disables incremental compilation

---

### watch

Watch proto files and automatically rebuild on changes.

**Usage:**
```bash
buffalo watch [flags]
```

**Flags:**
- `-p, --proto-path <paths>` - Paths to watch (default: ".")
- `-l, --lang <langs>` - Target languages
- `-o, --output <dir>` - Output directory (default: "./generated")
- `--debounce <ms>` - Debounce delay in milliseconds (default: 500)

**Examples:**
```bash
# Watch default paths
buffalo watch

# Watch specific paths
buffalo watch --proto-path ./protos --proto-path ./api

# Custom debounce (wait 1 second before rebuilding)
buffalo watch --debounce 1000

# Watch and build for Python and Go
buffalo watch --lang python,go
```

**How it works:**
1. Performs initial build
2. Monitors specified paths for .proto file changes
3. Debounces rapid changes
4. Automatically triggers rebuild on change
5. Press Ctrl+C to stop

---

## Code Quality

### lint

Check proto files for style violations and best practices.

**Usage:**
```bash
buffalo lint [flags]
```

**Flags:**
- `-p, --proto-path <paths>` - Paths to lint (default from config)
- `--fix` - Automatically fix issues when possible
- `--strict` - Treat warnings as errors
- `--rules <rules>` - Specific rules: naming,imports,docs,syntax
- `--ignore <patterns>` - Patterns to ignore

**Examples:**
```bash
# Lint all proto files
buffalo lint

# Lint with auto-fix
buffalo lint --fix

# Strict mode
buffalo lint --strict

# Specific rules only
buffalo lint --rules naming,imports
```

**Checks:**
- ✓ Syntax declaration (proto3 recommended)
- ✓ Package naming (lowercase)
- ✓ Message/service naming (PascalCase)
- ✓ Field naming (snake_case)
- ✓ Trailing whitespace

**Output:**
```
📄 protos/user.proto (3 issue(s))
  ⚠️ Line 5:1 [naming] Package name should be lowercase
  ℹ️ Line 12:1 [naming] Field name should be snake_case: 'userName'
  ℹ️ Line 25:42 [whitespace] Trailing whitespace [fixable]

╔════════════════════════════════════════════════════════╗
║  Lint Summary                                           ║
╚════════════════════════════════════════════════════════╝
   Files checked: 15
   Files with issues: 3
   Total issues: 8
   ├─ Errors: 1
   ├─ Warnings: 4
   └─ Info: 3
   Fixable: 2 (run with --fix)
```

---

### format

Format proto files according to style guidelines.

**Usage:**
```bash
buffalo format [flags]
buffalo fmt [flags]  # alias
```

**Flags:**
- `-w, --write` - Write formatted output to files
- `--check` - Check if files are formatted (exit with error if not)
- `-p, --proto-path <paths>` - Paths to format
- `--indent <n>` - Indentation spaces (default: 2)

**Examples:**
```bash
# Check formatting (dry run)
buffalo format

# Format and write changes
buffalo format --write

# Check in CI (fails if unformatted)
buffalo format --check

# Custom indentation
buffalo format --write --indent 4
```

**What it does:**
- Consistent indentation
- Proper brace placement
- Normalized whitespace
- Organized imports

---

### validate

Validate proto files using protoc for syntax and semantic errors.

**Usage:**
```bash
buffalo validate [flags]
```

**Flags:**
- `-p, --proto-path <paths>` - Paths to validate
- `--strict` - Treat warnings as errors

**Examples:**
```bash
# Validate all proto files
buffalo validate

# Validate specific paths
buffalo validate --proto-path ./protos

# Strict mode
buffalo validate --strict
```

**Checks:**
- Syntax errors
- Type mismatches
- Missing imports
- Duplicate definitions
- Invalid field numbers

**Output:**
```
✓ Validating proto files...
✅ protos/user.proto
✅ protos/auth.proto
❌ protos/broken.proto
   protos/broken.proto:15:3: "UnknownType" is not defined.

╔════════════════════════════════════════════════════════╗
║  Validation Summary                                     ║
╚════════════════════════════════════════════════════════╝
   Files checked: 3
   Valid: 2
   Errors: 1
```

---

## Project Management

### init

Initialize a new Buffalo project with default configuration.

**Usage:**
```bash
buffalo init [flags]
```

**Flags:**
- `-f, --force` - Overwrite existing config file

**Examples:**
```bash
# Create buffalo.yaml in current directory
buffalo init

# Force overwrite existing config
buffalo init --force
```

**Creates:**
- `buffalo.yaml` - Default configuration file with:
  - Project settings
  - Proto paths
  - Output directories
  - Language configurations

---

### check

Check project configuration and dependencies.

**Usage:**
```bash
buffalo check [flags]
```

**Flags:**
- `-v, --verbose` - Verbose output with details

**Examples:**
```bash
# Basic check
buffalo check

# Detailed check
buffalo check --verbose
```

**Validates:**
1. ✅ Configuration file (buffalo.yaml)
2. ✅ Proto files existence and readability
3. ✅ Output directory configuration
4. ✅ Enabled languages
5. ✅ protoc installation
6. ✅ Language-specific tools:
   - Python: python/python3
   - Go: protoc-gen-go, protoc-gen-go-grpc
   - Rust: cargo
   - C++: protoc with C++ support
7. ✅ Cache configuration

**Output:**
- Issues (❌) - Critical problems that prevent building
- Warnings (⚠️) - Non-critical issues
- Success (✅) - Everything OK

---

### list

List all proto files in the project.

**Usage:**
```bash
buffalo list [flags]
```

**Flags:**
- `-p, --proto-path <paths>` - Paths to search (default from config)
- `-r, --recursive` - Search recursively (default: true)
- `-f, --full` - Show full paths
- `-g, --grouped` - Group by directory

**Examples:**
```bash
# List all proto files
buffalo list

# Show full paths
buffalo list --full

# Group by directory
buffalo list --grouped

# List from specific paths
buffalo list --proto-path ./protos --proto-path ./api
```

**Output:**
```
📋 Listing proto files...
✅ Found 15 proto file(s)

• protos/user.proto
• protos/auth.proto
• api/v1/service.proto
...

📊 Summary:
   Files: 15
   Total size: 45.2 KB
   Directories: 3
```

---

### stats

Show project statistics.

**Usage:**
```bash
buffalo stats [flags]
```

**Flags:**
- `-d, --detailed` - Show detailed statistics
- `--json` - Output as JSON (not yet implemented)

**Examples:**
```bash
# Basic statistics
buffalo stats

# Detailed statistics
buffalo stats --detailed
```

**Shows:**
1. **Configuration**
   - Status (found/not found)
   - Enabled languages

2. **Proto Files**
   - Count and total size
   - Number of directories
   - Largest file (detailed mode)

3. **Generated Code**
   - Total files and size
   - Per-language breakdown (detailed mode)

4. **Build Cache**
   - Status (active/disabled)
   - Size and file count

5. **Quick Summary**
   - Proto → Generated files ratio
   - Compression ratio
   - Cache overhead

**Example Output:**
```
╔════════════════════════════════════════════════════════╗
║         Buffalo Project Statistics                     ║
╚════════════════════════════════════════════════════════╝

📋 Configuration
   Status: ✅ Found
   Enabled Languages: [python go]

📦 Proto Files
   Count: 15 file(s)
   Total Size: 45.2 KB
   Directories: 3

🔨 Generated Code
   Total Files: 120
   Total Size: 456.8 KB
   Python: 60 files (228.4 KB)
   Go: 60 files (228.4 KB)

💾 Build Cache
   Status: ✅ Active
   Files: 45
   Size: 123.5 KB

╔════════════════════════════════════════════════════════╗
║  Quick Summary                                          ║
╚════════════════════════════════════════════════════════╝
   Proto → Generated: 15 → 120 files
   Compression Ratio: 10.1x
   Cache Overhead: 123.5 KB
```

---

### clear

Clear cache and generated files.

**Usage:**
```bash
buffalo clear [flags]
```

**Flags:**
- `--cache` - Clear build cache only
- `--output` - Clear generated output only
- `--all` - Clear everything (default if no flags)
- `-y, --confirm` - Skip confirmation prompt

**Examples:**
```bash
# Clear everything (with confirmation)
buffalo clear

# Clear only cache
buffalo clear --cache

# Clear only output
buffalo clear --output

# Clear all without confirmation
buffalo clear --all --confirm
```

**Safety:**
- Asks for confirmation by default
- Shows what will be deleted
- Prevents deletion of root/home directories

---

## Utility Commands

### version

Print version information.

**Usage:**
```bash
buffalo version [flags]
```

**Flags:**
- `-s, --short` - Print only version number

**Examples:**
```bash
# Full version info
buffalo version

# Short version
buffalo version --short
```

**Output:**
```
🦬 Buffalo - Protobuf/gRPC Multi-Language Builder

Version:    v0.5.0
Commit:     a1b2c3d
Build Date: 2024-01-17
Go Version: go1.21.0
Platform:   windows/amd64

✅ Available Infrastructure:
  • Logger:  Structured logging system
  • Errors:  Enhanced error handling
  • Utils:   File operations & validation
  • Metrics: Performance monitoring

✅ Core Builder (v0.3.0):
  • Proto Parser:       Parse .proto files
  • Dependency Resolver: Topological sort
  • Executor:           Parallel compilation
  • Cache Manager:      Incremental builds

✅ Language Compilers (v0.5.0):
  • Python Compiler:    protoc + grpcio-tools
  • Go Compiler:        protoc-gen-go + grpc
  • Rust Compiler:      prost + tonic
  • C++ Compiler:       protoc + grpc++
```

---

### completion

Generate shell completion scripts.

**Usage:**
```bash
buffalo completion [bash|zsh|fish|powershell]
```

**Examples:**
```bash
# Bash
buffalo completion bash > /etc/bash_completion.d/buffalo

# Zsh
buffalo completion zsh > "${fpath[1]}/_buffalo"

# Fish
buffalo completion fish > ~/.config/fish/completions/buffalo.fish

# PowerShell
buffalo completion powershell > buffalo.ps1
```

---

## Global Flags

Available for all commands:

- `--config <file>` - Config file path (default: "./buffalo.yaml")
- `-v, --verbose` - Verbose output (debug level logging)
- `--help` - Show help for command
- `--version` - Show version (root command only)

**Examples:**
```bash
# Use custom config
buffalo build --config ./config/production.yaml

# Verbose output
buffalo check --verbose

# Both combined
buffalo build --config ./dev.yaml --verbose --lang python
```

---

## Command Aliases

Some commands have short aliases:

| Command | Alias |
|---------|-------|
| None yet | - |

---

## Exit Codes

Buffalo uses standard exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Build error |
| 4 | Dependency missing |

---

## Environment Variables

Buffalo respects the following environment variables:

- `BUFFALO_CONFIG` - Default config file path
- `BUFFALO_LOG_LEVEL` - Log level (debug, info, warn, error)
- `BUFFALO_NO_COLOR` - Disable colored output (set to any value)

**Examples:**
```bash
# Set log level
export BUFFALO_LOG_LEVEL=debug
buffalo build

# Use specific config
export BUFFALO_CONFIG=./production.yaml
buffalo build

# Disable colors
export BUFFALO_NO_COLOR=1
buffalo check
```

---

## Configuration Precedence

Buffalo loads configuration in this order (later overrides earlier):

1. Default values
2. Config file (`buffalo.yaml` or from `--config`)
3. Environment variables (`BUFFALO_*`)
4. Command-line flags

**Example:**
```yaml
# buffalo.yaml
output:
  base_dir: ./generated

# Command overrides config
buffalo build --output ./custom  # Uses ./custom
```

---

## Common Workflows

### Development Workflow

```bash
# 1. Initialize project
buffalo init

# 2. Edit buffalo.yaml (enable languages, set paths)
vim buffalo.yaml

# 3. Check everything is OK
buffalo check

# 4. Initial build
buffalo build

# 5. Start watch mode for development
buffalo watch --lang python,go
```

### CI/CD Workflow

```bash
# 1. Check configuration
buffalo check --verbose || exit 1

# 2. List proto files (for logging)
buffalo list

# 3. Full rebuild (no cache for clean build)
buffalo rebuild --lang python,go,rust,cpp

# 4. Show statistics
buffalo stats --detailed
```

### Cleanup Workflow

```bash
# 1. Show current stats
buffalo stats

# 2. Clear cache
buffalo clear --cache --confirm

# 3. Rebuild
buffalo rebuild

# 4. Show new stats
buffalo stats
```

---

## Tips & Tricks

### 1. Quick Proto File Count

```bash
buffalo list | head -1
```

### 2. Watch with Custom Debounce for Slow Systems

```bash
# Wait 2 seconds before rebuilding
buffalo watch --debounce 2000
```

### 3. Dry Run Before Real Build

```bash
# See what will happen
buffalo build --dry-run --verbose

# Then do it
buffalo build
```

### 4. Check Only Specific Tool

```bash
# Check if protoc is installed
buffalo check --verbose | grep protoc
```

### 5. Generate Only What You Need

```bash
# Only Python for quick iteration
buffalo watch --lang python
```

---

## Troubleshooting

### Command Not Found

```bash
# Check if buffalo is in PATH
which buffalo  # Unix
where buffalo  # Windows

# If not, add to PATH or use full path
./bin/buffalo build
```

### Watch Not Detecting Changes

```bash
# Increase debounce
buffalo watch --debounce 1000

# Check if paths are correct
buffalo list --proto-path ./your/path
```

### Permission Denied on Clear

```bash
# Use --confirm to skip prompt issues
buffalo clear --all --confirm

# Or manually delete
rm -rf ./generated .buffalo-cache
```

---

## See Also

- [BUILD_SYSTEM.md](BUILD_SYSTEM.md) - Build and installation guide
- [INSTALL.md](INSTALL.md) - Detailed installation instructions
- [README.md](README.md) - Project overview
- [buffalo.yaml](buffalo.yaml) - Configuration reference

---

**Last Updated:** 2025-01-17 | **Version:** v0.5.0
