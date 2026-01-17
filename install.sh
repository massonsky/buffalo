#!/usr/bin/env bash

# Buffalo Installation Script for Linux/macOS
# Usage: curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
# Or: ./install.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="buffalo"
REPO="massonsky/buffalo"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$os" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        *)
            log_error "Unsupported OS: $os"
            ;;
    esac
    
    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    log_info "Detected platform: $PLATFORM"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check for required tools
    for cmd in curl tar; do
        if ! command -v $cmd &> /dev/null; then
            log_error "$cmd is required but not installed"
        fi
    done
    
    # Check for protoc
    if ! command -v protoc &> /dev/null; then
        log_warning "protoc is not installed. Buffalo requires protoc to work."
        log_info "Install protoc from: https://github.com/protocolbuffers/protobuf/releases"
    else
        log_info "protoc version: $(protoc --version)"
    fi
    
    log_success "Prerequisites check passed"
}

get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        log_info "Fetching latest version..."
        VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$VERSION" ]; then
            log_warning "Could not fetch latest version, using master branch"
            VERSION="main"
        else
            log_info "Latest version: $VERSION"
        fi
    fi
}

download_binary() {
    log_info "Downloading Buffalo $VERSION for $PLATFORM..."
    
    local download_url
    if [ "$VERSION" = "main" ]; then
        # Build from source
        build_from_source
        return
    else
        download_url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${VERSION}-${PLATFORM}.tar.gz"
    fi
    
    local tmp_dir=$(mktemp -d)
    local tmp_file="${tmp_dir}/${BINARY_NAME}.tar.gz"
    
    if curl -sSL -f "$download_url" -o "$tmp_file"; then
        log_success "Downloaded successfully"
        tar -xzf "$tmp_file" -C "$tmp_dir"
        BINARY_PATH="${tmp_dir}/${BINARY_NAME}-${PLATFORM}"
    else
        log_warning "Release not found, building from source..."
        rm -rf "$tmp_dir"
        build_from_source
    fi
}

build_from_source() {
    log_info "Building Buffalo from source..."
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        log_error "Go is required to build from source. Install from: https://golang.org/dl/"
    fi
    
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $go_version"
    
    # Clone or download source
    local tmp_dir=$(mktemp -d)
    cd "$tmp_dir"
    
    log_info "Cloning repository..."
    if [ "$VERSION" = "main" ]; then
        git clone --depth 1 "https://github.com/${REPO}.git" . || \
            log_error "Failed to clone repository"
    else
        curl -sSL "https://github.com/${REPO}/archive/refs/tags/${VERSION}.tar.gz" | tar xz --strip-components=1 || \
            log_error "Failed to download source"
    fi
    
    log_info "Building binary..."
    go build -ldflags "-s -w" -o "${BINARY_NAME}" ./cmd/buffalo || \
        log_error "Build failed"
    
    BINARY_PATH="${tmp_dir}/${BINARY_NAME}"
    log_success "Built successfully"
}

install_binary() {
    log_info "Installing to $INSTALL_DIR..."
    
    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        log_info "Need sudo permissions to install to $INSTALL_DIR"
        sudo install -m 755 "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME" || \
            log_error "Installation failed"
    else
        install -m 755 "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME" || \
            log_error "Installation failed"
    fi
    
    log_success "Installed to $INSTALL_DIR/$BINARY_NAME"
}

verify_installation() {
    log_info "Verifying installation..."
    
    if command -v $BINARY_NAME &> /dev/null; then
        local installed_version=$($BINARY_NAME --version 2>&1 | head -n 1)
        log_success "Installation verified: $installed_version"
        log_info "Run '$BINARY_NAME --help' to get started"
    else
        log_error "Installation verification failed. Binary not found in PATH"
    fi
}

print_instructions() {
    echo ""
    log_success "Buffalo installed successfully!"
    echo ""
    echo -e "${BLUE}Quick Start:${NC}"
    echo "  1. Create a buffalo.yaml configuration file"
    echo "  2. Add your .proto files"
    echo "  3. Run: buffalo build"
    echo ""
    echo -e "${BLUE}Examples:${NC}"
    echo "  buffalo init              # Initialize new project"
    echo "  buffalo build             # Build proto files"
    echo "  buffalo build --lang go   # Build only for Go"
    echo "  buffalo --help            # Show all commands"
    echo ""
    echo -e "${BLUE}Documentation:${NC}"
    echo "  https://github.com/${REPO}"
    echo ""
}

main() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║   Buffalo Installation Script         ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
    echo ""
    
    detect_platform
    check_prerequisites
    get_latest_version
    download_binary
    install_binary
    verify_installation
    print_instructions
}

# Run main function
main "$@"
