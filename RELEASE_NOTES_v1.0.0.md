# Buffalo v1.0.0 - Stable Release 🎉

**Release Date:** January 17, 2026

We are thrilled to announce the first stable release of **Buffalo** - a cross-platform, multi-language protobuf/gRPC compiler!

---

## 🌟 Highlights

Buffalo v1.0.0 represents the culmination of extensive development, providing a production-ready tool for building protobuf and gRPC files across multiple languages with intelligent features.

### Core Features

- **🌐 Multi-Language Support**: Python, Go, Rust, C++
- **⚡ Parallel Compilation**: Configurable worker threads for optimal performance
- **📦 Intelligent Caching**: Skip unchanged files for faster builds
- **🔄 Incremental Builds**: Only rebuild what changed
- **🔌 Plugin System**: Extend functionality with custom plugins
- **📊 Build Metrics**: Comprehensive statistics and performance tracking
- **🔍 Diff Mode**: Preview changes before building
- **🩺 Doctor Command**: Diagnose environment issues
- **🛠️ CI/CD Ready**: GitHub Actions, GitLab CI, Pre-commit hooks

---

## 📋 Complete Feature List

### Build Commands
```bash
buffalo build              # Build all proto files
buffalo build --lang go    # Build for specific language
buffalo build --metrics    # Collect build metrics
buffalo rebuild            # Force full rebuild
buffalo clear              # Clear cache and generated files
```

### Analysis & Validation
```bash
buffalo validate           # Validate proto syntax
buffalo lint              # Check style and best practices
buffalo format            # Format proto files
buffalo deps              # Show dependency graph
buffalo diff              # Preview changes
```

### Configuration
```bash
buffalo config validate   # Validate buffalo.yaml
buffalo config show       # Display current config
buffalo config init       # Create new configuration
```

### Plugin Management
```bash
buffalo plugin list       # List plugins
buffalo plugin install    # Install plugin
buffalo plugin remove     # Remove plugin
buffalo plugin enable     # Enable plugin
buffalo plugin disable    # Disable plugin
```

### Templates
```bash
buffalo template list     # List templates
buffalo template generate # Generate from template
buffalo template validate # Validate template
```

### Diagnostics
```bash
buffalo doctor            # Check environment
buffalo metrics show      # Show build metrics
buffalo metrics history   # Build history
buffalo stats             # Project statistics
```

### Development
```bash
buffalo watch             # Watch and rebuild
buffalo init              # Initialize project
buffalo version           # Version info
```

---

## 🔧 Configuration Example

```yaml
# buffalo.yaml
project:
  name: "my-project"
  version: "1.0.0"

proto:
  paths:
    - "./protos"
  import_paths:
    - "./protos"
  exclude:
    - "**/*_test.proto"

output:
  base_dir: "./generated"
  preserve_proto_structure: true
  directories:
    python: "python"
    go: "go"
    rust: "rust"
    cpp: "cpp"

languages:
  python:
    enabled: true
    package: "my_project"
    generator: "grpcio"
  go:
    enabled: true
    module: "github.com/myorg/my-project"
    generator: "protoc-gen-go"
  rust:
    enabled: false
    generator: "tonic"
  cpp:
    enabled: false
    namespace: "myproject"

build:
  workers: 4
  incremental: true
  cache:
    enabled: true
    directory: ".buffalo/cache"

plugins: []
templates: []
```

---

## 📊 Version History

| Version | Description | Date |
|---------|-------------|------|
| v1.0.0 | Stable Release | Jan 2026 |
| v0.9.0 | Templates, Config Validation, Metrics | Jan 2026 |
| v0.8.0 | Diff Command | Jan 2026 |
| v0.7.0 | CI/CD Automation, Plugin CLI | Jan 2026 |
| v0.6.0 | Plugin System | Jan 2025 |
| v0.5.0 | Full CLI, All Compilers | 2025 |
| v0.4.0 | C++ Compiler | 2025 |
| v0.3.0 | Rust Compiler | 2025 |
| v0.2.0 | Go Compiler | 2025 |
| v0.1.0 | Python Compiler | 2025 |

---

## 🚀 Getting Started

### Installation

```bash
# From source
git clone https://github.com/massonsky/buffalo.git
cd buffalo
go build -o buffalo ./cmd/buffalo

# Add to PATH
export PATH=$PATH:$(pwd)
```

### Quick Start

```bash
# Initialize project
buffalo init

# Edit buffalo.yaml for your project

# Build
buffalo build

# Watch for changes
buffalo watch
```

---

## 📚 Documentation

- [README.md](../README.md) - Overview
- [CLI_COMMANDS.md](CLI_COMMANDS.md) - Command reference
- [CONFIG_GUIDE.md](CONFIG_GUIDE.md) - Configuration guide
- [PLUGIN_GUIDE.md](PLUGIN_GUIDE.md) - Plugin development
- [CI_CD_GUIDE.md](CI_CD_GUIDE.md) - CI/CD integration
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contributing guide

---

## 🙏 Acknowledgments

Thank you to everyone who contributed to making Buffalo v1.0.0 a reality!

---

## 📄 License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Happy building with Buffalo! 🦬**
