.PHONY: help build test lint clean install run fmt vet check coverage

# Variables
BINARY_NAME=buffalo
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"
BUILD_DIR=./build
BIN_DIR=./bin

# Help target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

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
	rm -rf $(BUILD_DIR) $(BIN_DIR)
	rm -f coverage.txt coverage.html
	$(GO) clean

# Development targets
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	$(GO) run ./cmd/buffalo

dev: ## Run in development mode with hot reload
	@echo "Starting development mode..."
	air

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
	@$(GO) version
	@echo "Buffalo version: development"

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t buffalo:latest .

docker-run: ## Run in Docker
	@echo "Running in Docker..."
	docker run -it --rm buffalo:latest

.DEFAULT_GOAL := help
