.PHONY: help build test lint clean install run fmt vet check coverage

# Variables
BINARY_NAME=buffalo
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.5.0-dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S' 2>/dev/null || echo "unknown")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-X github.com/massonsky/buffalo/internal/version.Version=$(VERSION) \
                  -X github.com/massonsky/buffalo/internal/version.BuildTime=$(BUILD_TIME) \
                  -X github.com/massonsky/buffalo/internal/version.GitCommit=$(GIT_COMMIT) \
                  -s -w"
BUILD_DIR=./build
BIN_DIR=./bin
DIST_DIR=./dist
INSTALL_PREFIX?=/usr/local
INSTALL_BIN=$(INSTALL_PREFIX)/bin

# Help target
help: ## Show this help message
	@echo "Buffalo - Protocol Buffer Compiler"
	@echo ""
	@echo "Available targets:"
	@echo "  help         - Show this help message"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  install      - Install to GOPATH/bin"
	@echo "  install-system - Install to system (requires admin/sudo)"
	@echo "  uninstall-system - Uninstall from system"
	@echo "  test         - Run all tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  benchmark    - Run benchmarks"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run linter"
	@echo "  check        - Run all checks"
	@echo "  clean        - Clean build artifacts"
	@echo "  clean-all    - Clean all including caches"
	@echo "  run          - Run the application"
	@echo "  example      - Run example build"
	@echo "  dev          - Run in development mode"
	@echo "  release      - Create release builds"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy go.mod"
	@echo "  version      - Show version information"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run in Docker"

# Build targets
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/buffalo

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/buffalo
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/buffalo
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/buffalo
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/buffalo
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/buffalo

install: ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) ./cmd/buffalo

install-system: build ## Install the binary to system (requires sudo on Unix)
	@echo "Installing $(BINARY_NAME) to $(INSTALL_BIN)..."
	@mkdir -p $(INSTALL_BIN) 2>/dev/null || true
	@cp $(BIN_DIR)/$(BINARY_NAME)* $(INSTALL_BIN)/ 2>/dev/null || \
		powershell -Command "Copy-Item -Path '$(BIN_DIR)\$(BINARY_NAME).exe' -Destination '$(INSTALL_BIN)\$(BINARY_NAME).exe' -Force" 2>/dev/null || \
		echo "Run as Administrator or use sudo"
	@echo "Installed to $(INSTALL_BIN)/$(BINARY_NAME)"

uninstall-system: ## Uninstall the binary from system
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN)/$(BINARY_NAME) 2>/dev/null || \
		powershell -Command "Remove-Item '$(INSTALL_BIN)\$(BINARY_NAME).exe' -Force -ErrorAction SilentlyContinue" 2>/dev/null || true
	@echo "Uninstalled"

# Test targets
test: ## Run all tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	$(GO) test -v -race ./internal/... ./pkg/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -v -race ./tests/integration/...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Code quality targets
fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run ./...

check: fmt vet lint ## Run all checks (fmt, vet, lint)

fix: ## Fix auto-fixable issues
	@echo "Fixing auto-fixable issues..."
	golangci-lint run --fix ./...

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) $(BIN_DIR) $(DIST_DIR)
	rm -f coverage.txt coverage.html
	$(GO) clean

clean-all: clean ## Clean all including caches
	@echo "Cleaning all..."
	$(GO) clean -cache -testcache -modcache

# Development targets
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	$(GO) run ./cmd/buffalo

example: build ## Run example build
	@echo "Running example..."
	cd test-project && ../$(BIN_DIR)/$(BINARY_NAME) build --lang python,go

dev: ## Run in development mode with hot reload
	@echo "Starting development mode..."
	air

release: clean check build-all ## Create release builds
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)/archives
	cd $(BUILD_DIR) && \
		tar czf ../$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
		tar czf ../$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
		tar czf ../$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
		tar czf ../$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
		zip ../$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release archives created in $(DIST_DIR)/archives/"

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	$(GO) mod tidy

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GO) mod verify

# Tools installation
install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/cosmtrek/air@latest

# CI/CD
ci: check test ## Run CI checks

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060

# Version
version: ## Show version information
	@echo "Buffalo Protocol Buffer Compiler"
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(shell $(GO) version)"

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t buffalo:latest .

docker-run: ## Run in Docker
	@echo "Running in Docker..."
	docker run -it --rm buffalo:latest

.DEFAULT_GOAL := help
