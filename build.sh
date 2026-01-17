#!/usr/bin/env bash
# Buffalo Build Script for Unix-like systems (Linux, macOS)
# This script provides convenient building, testing, and installation

set -e

# Configuration
BINARY_NAME="buffalo"
MODULE="github.com/massonsky/buffalo"
VERSION="v0.5.0-dev"
BIN_DIR="bin"
BUILD_DIR="build"
INSTALL_PREFIX="${INSTALL_PREFIX:-/usr/local}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

ensure_directory() {
    if [ ! -d "$1" ]; then
        mkdir -p "$1"
    fi
}

get_version() {
    if git describe --tags --always --dirty 2>/dev/null; then
        return
    fi
    echo "$VERSION"
}

get_build_time() {
    date "+%Y-%m-%d_%H:%M:%S"
}

get_git_commit() {
    git rev-parse --short HEAD 2>/dev/null || echo "unknown"
}

build_binary() {
    local goos="$1"
    local goarch="$2"
    local output_name="$3"
    
    local version=$(get_version)
    local build_time=$(get_build_time)
    local git_commit=$(get_git_commit)
    
    local ldflags="-X ${MODULE}/internal/version.Version=${version} \
        -X ${MODULE}/internal/version.BuildTime=${build_time} \
        -X ${MODULE}/internal/version.GitCommit=${git_commit} \
        -s -w"
    
    log_info "Building $output_name..."
    
    if [ -n "$goos" ] && [ -n "$goarch" ]; then
        GOOS="$goos" GOARCH="$goarch" go build \
            -ldflags "$ldflags" \
            -o "$output_name" \
            ./cmd/buffalo
    else
        go build \
            -ldflags "$ldflags" \
            -o "$output_name" \
            ./cmd/buffalo
    fi
    
    log_success "Built: $output_name"
}

target_help() {
    cat << EOF
${BLUE}╔════════════════════════════════════════════════════════╗${NC}
${BLUE}║   Buffalo - Protocol Buffer Compiler                 ║${NC}
${BLUE}║   Unix Build Script                                   ║${NC}
${BLUE}╚════════════════════════════════════════════════════════╝${NC}

${GREEN}Usage: ./build.sh [target] [options]${NC}

${BLUE}Targets:${NC}
  help            Show this help message
  build           Build the binary (default)
  build-all       Build for all platforms
  test            Run tests
  test-coverage   Run tests with coverage
  fmt             Format code
  vet             Run go vet
  lint            Run linter (golangci-lint)
  check           Run all checks (fmt, vet, lint, test)
  install         Install to ${INSTALL_PREFIX}/bin
  uninstall       Uninstall from system
  clean           Clean build artifacts
  clean-all       Clean all including caches
  version         Show version information
  example         Run example build

${BLUE}Options:${NC}
  INSTALL_PREFIX=<path>   Installation directory (default: /usr/local)
  VERBOSE=1               Show detailed output

${GREEN}Examples:${NC}
  ./build.sh build
  ./build.sh build-all
  ./build.sh test
  INSTALL_PREFIX=~/.local ./build.sh install

EOF
}

target_build() {
    ensure_directory "$BIN_DIR"
    build_binary "" "" "$BIN_DIR/$BINARY_NAME"
}

target_build_all() {
    ensure_directory "$BUILD_DIR"
    
    log_info "Building for all platforms..."
    
    build_binary "linux" "amd64" "$BUILD_DIR/${BINARY_NAME}-linux-amd64"
    build_binary "linux" "arm64" "$BUILD_DIR/${BINARY_NAME}-linux-arm64"
    build_binary "darwin" "amd64" "$BUILD_DIR/${BINARY_NAME}-darwin-amd64"
    build_binary "darwin" "arm64" "$BUILD_DIR/${BINARY_NAME}-darwin-arm64"
    build_binary "windows" "amd64" "$BUILD_DIR/${BINARY_NAME}-windows-amd64.exe"
    
    log_success "All platforms built in $BUILD_DIR/"
}

target_test() {
    log_info "Running tests..."
    go test -v -race -coverprofile=coverage.out ./...
    log_success "Tests passed"
}

target_test_coverage() {
    log_info "Running tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./...
    
    log_info "Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    log_success "Coverage report: coverage.html"
}

target_fmt() {
    log_info "Formatting code..."
    go fmt ./...
    log_success "Code formatted"
}

target_vet() {
    log_info "Running go vet..."
    go vet ./...
    log_success "Vet passed"
}

target_lint() {
    log_info "Running linter..."
    
    if ! command -v golangci-lint &> /dev/null; then
        log_warning "golangci-lint not found. Installing..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi
    
    golangci-lint run ./...
    log_success "Lint passed"
}

target_check() {
    log_info "Running all checks..."
    target_fmt
    target_vet
    target_lint
    target_test
    log_success "All checks passed"
}

target_install() {
    target_build
    
    log_info "Installing to $INSTALL_PREFIX/bin..."
    
    if [ ! -w "$INSTALL_PREFIX/bin" ]; then
        log_info "Using sudo for installation..."
        sudo cp "$BIN_DIR/$BINARY_NAME" "$INSTALL_PREFIX/bin/$BINARY_NAME"
        sudo chmod +x "$INSTALL_PREFIX/bin/$BINARY_NAME"
    else
        ensure_directory "$INSTALL_PREFIX/bin"
        cp "$BIN_DIR/$BINARY_NAME" "$INSTALL_PREFIX/bin/$BINARY_NAME"
        chmod +x "$INSTALL_PREFIX/bin/$BINARY_NAME"
    fi
    
    log_success "Installed to $INSTALL_PREFIX/bin/$BINARY_NAME"
    
    if [[ "$INSTALL_PREFIX/bin" != *"$PATH"* ]]; then
        log_warning "Add '$INSTALL_PREFIX/bin' to your PATH"
    fi
}

target_uninstall() {
    if [ -f "$INSTALL_PREFIX/bin/$BINARY_NAME" ]; then
        if [ ! -w "$INSTALL_PREFIX/bin" ]; then
            sudo rm "$INSTALL_PREFIX/bin/$BINARY_NAME"
        else
            rm "$INSTALL_PREFIX/bin/$BINARY_NAME"
        fi
        log_success "Uninstalled"
    else
        log_warning "Not installed at $INSTALL_PREFIX/bin"
    fi
}

target_clean() {
    log_info "Cleaning build artifacts..."
    rm -rf "$BIN_DIR" "$BUILD_DIR"
    rm -f coverage.out coverage.html
    go clean
    log_success "Clean complete"
}

target_clean_all() {
    target_clean
    log_info "Cleaning caches..."
    go clean -cache -testcache -modcache
    log_success "All clean"
}

target_version() {
    local version=$(get_version)
    local build_time=$(get_build_time)
    local git_commit=$(get_git_commit)
    local go_version=$(go version)
    
    echo ""
    echo -e "${GREEN}Buffalo Protocol Buffer Compiler${NC}"
    echo -e "${GREEN}Version:    $version${NC}"
    echo -e "${GREEN}Build Time: $build_time${NC}"
    echo -e "${GREEN}Git Commit: $git_commit${NC}"
    echo -e "${GREEN}Go Version: $go_version${NC}"
    echo ""
}

target_example() {
    target_build
    log_info "Running example build..."
    pushd test-project > /dev/null
    ../"$BIN_DIR"/"$BINARY_NAME" build --lang python,go
    popd > /dev/null
}

# Main execution
target="${1:-build}"

case "$target" in
    help)           target_help ;;
    build)          target_build ;;
    build-all)      target_build_all ;;
    test)           target_test ;;
    test-coverage)  target_test_coverage ;;
    fmt)            target_fmt ;;
    vet)            target_vet ;;
    lint)           target_lint ;;
    check)          target_check ;;
    install)        target_install ;;
    uninstall)      target_uninstall ;;
    clean)          target_clean ;;
    clean-all)      target_clean_all ;;
    version)        target_version ;;
    example)        target_example ;;
    *)              log_error "Unknown target: $target. Run './build.sh help' for usage." ;;
esac
