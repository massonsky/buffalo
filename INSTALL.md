# Installation Guide

Complete installation guide for Buffalo Protocol Buffer Compiler.

## Table of Contents

- [Quick Install](#quick-install)
- [System Requirements](#system-requirements)
- [Installation Methods](#installation-methods)
  - [Automated Scripts](#automated-scripts)
  - [From Source](#from-source)
  - [Pre-built Binaries](#pre-built-binaries)
  - [Docker](#docker)
  - [Package Managers](#package-managers)
- [Post-Installation](#post-installation)
- [Troubleshooting](#troubleshooting)

## Quick Install

### Linux / macOS

```bash
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
```

### Windows (PowerShell as Administrator)

```powershell
irm https://raw.githubusercontent.com/massonsky/buffalo/main/install.ps1 | iex
```

## System Requirements

### Minimum Requirements

- **Operating System:** Linux, macOS, or Windows
- **Architecture:** amd64 (x86_64) or arm64
- **Protocol Buffers Compiler:** protoc 3.x or higher
- **Memory:** 512 MB RAM
- **Disk Space:** 50 MB

### Required Dependencies

- **protoc** - Protocol Buffers compiler (required)
- **Go** 1.21+ (only if building from source)

### Language-Specific Dependencies

Install these based on which languages you'll be compiling to:

#### Python
```bash
pip install grpcio-tools
```

#### Go
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### Rust
```bash
cargo install protobuf-codegen
```

#### C++
- Install gRPC C++ plugin (included with protoc in most distributions)

## Installation Methods

### Automated Scripts

#### Linux / macOS

The automated script will:
- Detect your platform (OS and architecture)
- Download the latest release or build from source
- Install to `/usr/local/bin` (or `$INSTALL_DIR` if set)
- Optionally add to PATH

**Basic Installation:**
```bash
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
```

**Custom Installation Directory:**
```bash
export INSTALL_DIR="$HOME/.local/bin"
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
```

**Specific Version:**
```bash
export VERSION="v0.5.0"
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
```

**Manual Download:**
```bash
wget https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh
chmod +x install.sh
./install.sh
```

#### Windows (PowerShell)

The PowerShell script will:
- Detect your architecture (amd64/arm64)
- Download the latest release or build from source
- Install to `C:\Program Files\buffalo` (default)
- Add to system PATH

**Basic Installation (as Administrator):**
```powershell
irm https://raw.githubusercontent.com/massonsky/buffalo/main/install.ps1 | iex
```

**Custom Installation Directory:**
```powershell
.\install.ps1 -InstallDir "$env:LOCALAPPDATA\buffalo" -AddToPath
```

**Specific Version:**
```powershell
.\install.ps1 -Version "v0.5.0"
```

**Without PATH modification:**
```powershell
.\install.ps1 -AddToPath:$false
```

### From Source

Building from source gives you the most control and latest features.

#### Prerequisites

Install Go 1.21 or higher:

**Linux / macOS:**
```bash
# Download and install from https://go.dev/dl/
wget https://go.dev/dl/go1.23.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

**Windows:**
```powershell
# Download installer from https://go.dev/dl/
# Or use Chocolatey:
choco install golang
```

#### Build Steps

```bash
# 1. Clone repository
git clone https://github.com/massonsky/buffalo.git
cd buffalo

# 2. Download dependencies
make deps

# 3. Run tests (optional but recommended)
make test

# 4. Build
make build

# 5. Install to system
sudo make install-system

# Or install to $GOPATH/bin
make install
```

#### Build Options

**Build for specific platform:**
```bash
GOOS=linux GOARCH=amd64 make build
```

**Build for all platforms:**
```bash
make build-all
# Outputs to ./build/ directory
```

**Build with custom flags:**
```bash
make build LDFLAGS="-ldflags '-X main.Version=custom'"
```

### Pre-built Binaries

Download pre-built binaries from [GitHub Releases](https://github.com/massonsky/buffalo/releases).

#### Linux / macOS

```bash
# Download
VERSION="v0.5.0"
OS="linux"  # or "darwin" for macOS
ARCH="amd64"  # or "arm64"
curl -LO "https://github.com/massonsky/buffalo/releases/download/${VERSION}/buffalo-${VERSION}-${OS}-${ARCH}.tar.gz"

# Extract
tar -xzf "buffalo-${VERSION}-${OS}-${ARCH}.tar.gz"

# Install
sudo install -m 755 buffalo-${OS}-${ARCH} /usr/local/bin/buffalo

# Verify
buffalo --version
```

#### Windows

```powershell
# Download
$VERSION = "v0.5.0"
Invoke-WebRequest -Uri "https://github.com/massonsky/buffalo/releases/download/$VERSION/buffalo-$VERSION-windows-amd64.zip" -OutFile buffalo.zip

# Extract
Expand-Archive buffalo.zip -DestinationPath .

# Move to Program Files (as Administrator)
Move-Item buffalo-windows-amd64.exe "$env:ProgramFiles\buffalo\buffalo.exe" -Force

# Add to PATH
$path = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$path;$env:ProgramFiles\buffalo", "User")
```

### Docker

Use Docker for isolated and reproducible builds.

#### Pull from Docker Hub (when available)

```bash
docker pull massonsky/buffalo:latest
```

#### Build Locally

```bash
# Clone repository
git clone https://github.com/massonsky/buffalo.git
cd buffalo

# Build image
docker build -t buffalo:latest .

# Or use docker-compose
docker-compose build
```

#### Usage

```bash
# Basic usage
docker run --rm -v $(pwd):/workspace buffalo:latest build

# With custom config
docker run --rm -v $(pwd):/workspace buffalo:latest build --config buffalo.yaml

# Interactive mode
docker run --rm -it -v $(pwd):/workspace buffalo:latest sh

# Using docker-compose
docker-compose run buffalo build
```

### Package Managers

Package manager support is coming soon.

#### Homebrew (macOS/Linux) - Coming Soon

```bash
brew install buffalo
```

#### Chocolatey (Windows) - Coming Soon

```powershell
choco install buffalo
```

#### APT (Debian/Ubuntu) - Coming Soon

```bash
sudo apt install buffalo
```

## Post-Installation

### Install protoc

Buffalo requires `protoc` to be installed and available in PATH.

#### Linux

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install -y protobuf-compiler
```

**Fedora/RHEL:**
```bash
sudo dnf install protobuf-compiler
```

**Arch Linux:**
```bash
sudo pacman -S protobuf
```

**From Source:**
```bash
# Download latest release from https://github.com/protocolbuffers/protobuf/releases
VERSION="28.3"
wget https://github.com/protocolbuffers/protobuf/releases/download/v${VERSION}/protoc-${VERSION}-linux-x86_64.zip
unzip protoc-${VERSION}-linux-x86_64.zip -d /usr/local
```

#### macOS

```bash
brew install protobuf
```

#### Windows

**Using Chocolatey:**
```powershell
choco install protoc
```

**Manual Installation:**
1. Download from [Protobuf Releases](https://github.com/protocolbuffers/protobuf/releases)
2. Extract to `C:\Program Files\protoc`
3. Add `C:\Program Files\protoc\bin` to PATH

### Install Language-Specific Plugins

#### Python

```bash
# Install grpcio-tools
pip install grpcio-tools

# Verify
python -c "import grpc_tools"
```

#### Go

```bash
# Install protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Install protoc-gen-go-grpc
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure $GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

#### Rust

```bash
# Using cargo
cargo install protobuf-codegen

# Or add to Cargo.toml
# [build-dependencies]
# protobuf-codegen = "3.0"
```

#### C++

Usually included with protobuf installation. Verify:
```bash
which grpc_cpp_plugin
```

### Verify Installation

```bash
# Check Buffalo
buffalo --version
buffalo --help

# Check protoc
protoc --version

# Check plugins (Go example)
protoc-gen-go --version
protoc-gen-go-grpc --version

# Test with example
buffalo init test-project
cd test-project
buffalo build
```

### Shell Completion (Optional)

Generate shell completion scripts:

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

## Troubleshooting

### Common Issues

#### "buffalo: command not found"

**Solution:** Add installation directory to PATH.

**Linux/macOS:**
```bash
export PATH=$PATH:/usr/local/bin
# Add to ~/.bashrc or ~/.zshrc to make permanent
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
```

**Windows:**
```powershell
$path = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$path;C:\Program Files\buffalo", "User")
```

#### "protoc: command not found"

**Solution:** Install protoc as described in [Post-Installation](#post-installation).

#### Permission denied on Linux/macOS

**Solution:** Use sudo for system-wide installation:
```bash
sudo make install-system
```

Or install to user directory:
```bash
export INSTALL_DIR="$HOME/.local/bin"
./install.sh
```

#### Windows: "Running scripts is disabled"

**Solution:** Enable script execution in PowerShell:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### Go version too old

**Solution:** Update Go to 1.21 or higher:
```bash
# Linux/macOS
go version  # Check current version
# Download and install from https://go.dev/dl/

# Windows
choco upgrade golang
```

#### Build fails with "module not found"

**Solution:**
```bash
go mod download
go mod tidy
make build
```

### Getting Help

- **Issues:** [GitHub Issues](https://github.com/massonsky/buffalo/issues)
- **Discussions:** [GitHub Discussions](https://github.com/massonsky/buffalo/discussions)
- **Documentation:** [Full Documentation](https://github.com/massonsky/buffalo/tree/main/docs)

### Uninstall

#### Installed via script

**Linux/macOS:**
```bash
sudo rm /usr/local/bin/buffalo
```

**Windows:**
```powershell
Remove-Item "$env:ProgramFiles\buffalo\buffalo.exe"
# Remove from PATH manually
```

#### Installed via make

```bash
sudo make uninstall-system
```

#### Installed via Go

```bash
rm $(go env GOPATH)/bin/buffalo
```

---

**Next Steps:** After installation, see the [Quick Start Guide](README.md#quick-start) to begin using Buffalo.
