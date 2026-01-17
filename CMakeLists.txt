cmake_minimum_required(VERSION 3.10)
project(buffalo VERSION 0.5.0 LANGUAGES NONE)

# Go configuration
find_program(GO_EXECUTABLE go REQUIRED)
set(CMAKE_GO_FLAGS "-v")

# Set variables
set(BINARY_NAME buffalo)
set(OUTPUT_DIR ${CMAKE_BINARY_DIR}/bin)
set(SOURCE_DIR ${CMAKE_SOURCE_DIR})

# Version information
execute_process(
    COMMAND git describe --tags --always --dirty
    OUTPUT_VARIABLE GIT_VERSION
    OUTPUT_STRIP_TRAILING_WHITESPACE
    ERROR_QUIET
)
if(NOT GIT_VERSION)
    set(GIT_VERSION "v0.5.0-dev")
endif()

execute_process(
    COMMAND git rev-parse --short HEAD
    OUTPUT_VARIABLE GIT_COMMIT
    OUTPUT_STRIP_TRAILING_WHITESPACE
    ERROR_QUIET
)
if(NOT GIT_COMMIT)
    set(GIT_COMMIT "unknown")
endif()

string(TIMESTAMP BUILD_TIME "%Y-%m-%d_%H:%M:%S" UTC)

# Compiler flags
set(VERSION_PKG "github.com/massonsky/buffalo/internal/version")
set(GO_LDFLAGS_VALUE "-X ${VERSION_PKG}.Version=${GIT_VERSION} -X ${VERSION_PKG}.BuildDate=${BUILD_TIME} -X ${VERSION_PKG}.GitCommit=${GIT_COMMIT} -s -w")

# Ensure output directory exists
file(MAKE_DIRECTORY ${OUTPUT_DIR})

# Main build target
add_custom_target(build
    COMMAND ${CMAKE_COMMAND} -E echo "Building ${BINARY_NAME} ${GIT_VERSION}..."
    COMMAND ${GO_EXECUTABLE} build ${CMAKE_GO_FLAGS} -ldflags "${GO_LDFLAGS_VALUE}" -o ${OUTPUT_DIR}/${BINARY_NAME}${CMAKE_EXECUTABLE_SUFFIX} ./cmd/buffalo
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Building Buffalo..."
    VERBATIM
)

# Build all platforms
add_custom_target(build-all
    COMMAND ${CMAKE_COMMAND} -E echo "Building for all platforms..."
    COMMAND ${CMAKE_COMMAND} -E make_directory ${OUTPUT_DIR}
    COMMAND ${CMAKE_COMMAND} -E env GOOS=linux GOARCH=amd64 ${GO_EXECUTABLE} build -ldflags "${GO_LDFLAGS_VALUE}" -o ${OUTPUT_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/buffalo
    COMMAND ${CMAKE_COMMAND} -E env GOOS=linux GOARCH=arm64 ${GO_EXECUTABLE} build -ldflags "${GO_LDFLAGS_VALUE}" -o ${OUTPUT_DIR}/${BINARY_NAME}-linux-arm64 ./cmd/buffalo
    COMMAND ${CMAKE_COMMAND} -E env GOOS=darwin GOARCH=amd64 ${GO_EXECUTABLE} build -ldflags "${GO_LDFLAGS_VALUE}" -o ${OUTPUT_DIR}/${BINARY_NAME}-darwin-amd64 ./cmd/buffalo
    COMMAND ${CMAKE_COMMAND} -E env GOOS=darwin GOARCH=arm64 ${GO_EXECUTABLE} build -ldflags "${GO_LDFLAGS_VALUE}" -o ${OUTPUT_DIR}/${BINARY_NAME}-darwin-arm64 ./cmd/buffalo
    COMMAND ${CMAKE_COMMAND} -E env GOOS=windows GOARCH=amd64 ${GO_EXECUTABLE} build -ldflags "${GO_LDFLAGS_VALUE}" -o ${OUTPUT_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/buffalo
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Building for all platforms..."
    VERBATIM
)

# Test target
add_custom_target(test
    COMMAND ${GO_EXECUTABLE} test -v -race -coverprofile=coverage.out ./...
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Running tests..."
    VERBATIM
)

# Coverage target
add_custom_target(test-coverage
    COMMAND ${GO_EXECUTABLE} test -v -race -coverprofile=coverage.out ./...
    COMMAND ${GO_EXECUTABLE} tool cover -html=coverage.out -o coverage.html
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Running tests with coverage..."
    VERBATIM
)

# Lint target
add_custom_target(lint
    COMMAND golangci-lint run ./...
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Running linter..."
    VERBATIM
)

# Format target
add_custom_target(fmt
    COMMAND ${GO_EXECUTABLE} fmt ./...
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Formatting code..."
    VERBATIM
)

# Vet target
add_custom_target(vet
    COMMAND ${GO_EXECUTABLE} vet ./...
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Running go vet..."
    VERBATIM
)

# Install to GOPATH/bin
add_custom_target(install-go
    COMMAND ${GO_EXECUTABLE} install -ldflags "${GO_LDFLAGS_VALUE}" ./cmd/buffalo
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Installing to GOPATH/bin..."
    VERBATIM
)

# Install prefix configuration
if(NOT INSTALL_PREFIX)
    if(WIN32)
        set(INSTALL_PREFIX "$ENV{ProgramFiles}/buffalo")
    else()
        set(INSTALL_PREFIX "/usr/local")
    endif()
endif()

# Install to system
add_custom_target(install-system
    DEPENDS build
    COMMAND ${CMAKE_COMMAND} -E make_directory "${INSTALL_PREFIX}/bin"
    COMMAND ${CMAKE_COMMAND} -E copy ${OUTPUT_DIR}/${BINARY_NAME}${CMAKE_EXECUTABLE_SUFFIX} "${INSTALL_PREFIX}/bin/${BINARY_NAME}${CMAKE_EXECUTABLE_SUFFIX}"
    COMMENT "Installing to ${INSTALL_PREFIX}/bin..."
    VERBATIM
)

# Uninstall from system
add_custom_target(uninstall-system
    COMMAND ${CMAKE_COMMAND} -E remove -f "${INSTALL_PREFIX}/bin/${BINARY_NAME}${CMAKE_EXECUTABLE_SUFFIX}"
    COMMENT "Uninstalling from ${INSTALL_PREFIX}/bin..."
    VERBATIM
)

# Clean target
add_custom_target(clean-build
    COMMAND ${CMAKE_COMMAND} -E remove_directory ${OUTPUT_DIR}
    COMMAND ${CMAKE_COMMAND} -E remove -f coverage.out coverage.html
    COMMAND ${GO_EXECUTABLE} clean
    WORKING_DIRECTORY ${SOURCE_DIR}
    COMMENT "Cleaning build artifacts..."
    VERBATIM
)

# Check target (all quality checks)
add_custom_target(check
    DEPENDS fmt vet lint test
    COMMENT "Running all checks..."
)

# Version target
add_custom_target(version
    COMMAND ${CMAKE_COMMAND} -E echo "Buffalo Protocol Buffer Compiler"
    COMMAND ${CMAKE_COMMAND} -E echo "Version:    ${GIT_VERSION}"
    COMMAND ${CMAKE_COMMAND} -E echo "Build Time: ${BUILD_TIME}"
    COMMAND ${CMAKE_COMMAND} -E echo "Git Commit: ${GIT_COMMIT}"
    COMMENT "Showing version information..."
    VERBATIM
)

# Display usage information
add_custom_target(usage
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "Buffalo - Protocol Buffer Compiler"
    COMMAND ${CMAKE_COMMAND} -E echo "CMake Build System"
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "Usage: cmake --build . --target <target>"
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "Available targets:"
    COMMAND ${CMAKE_COMMAND} -E echo "  build            - Build the binary"
    COMMAND ${CMAKE_COMMAND} -E echo "  build-all        - Build for all platforms"
    COMMAND ${CMAKE_COMMAND} -E echo "  test             - Run tests"
    COMMAND ${CMAKE_COMMAND} -E echo "  test-coverage    - Run tests with coverage report"
    COMMAND ${CMAKE_COMMAND} -E echo "  fmt              - Format code"
    COMMAND ${CMAKE_COMMAND} -E echo "  vet              - Run go vet"
    COMMAND ${CMAKE_COMMAND} -E echo "  lint             - Run linter"
    COMMAND ${CMAKE_COMMAND} -E echo "  check            - Run all checks"
    COMMAND ${CMAKE_COMMAND} -E echo "  clean-build      - Clean build artifacts"
    COMMAND ${CMAKE_COMMAND} -E echo "  install-go       - Install to GOPATH/bin"
    COMMAND ${CMAKE_COMMAND} -E echo "  install-system   - Install to system (${INSTALL_PREFIX}/bin)"
    COMMAND ${CMAKE_COMMAND} -E echo "  uninstall-system - Uninstall from system"
    COMMAND ${CMAKE_COMMAND} -E echo "  version          - Show version information"
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "Build options:"
    COMMAND ${CMAKE_COMMAND} -E echo "  cmake -B build"
    COMMAND ${CMAKE_COMMAND} -E echo "  cmake --build build --target build"
    COMMAND ${CMAKE_COMMAND} -E echo "  cmake --build build --target install-system"
    COMMAND ${CMAKE_COMMAND} -E echo ""
    COMMAND ${CMAKE_COMMAND} -E echo "Custom install prefix:"
    COMMAND ${CMAKE_COMMAND} -E echo "  cmake -B build -DINSTALL_PREFIX=/custom/path"
    COMMENT "Showing usage information..."
    VERBATIM
)

# Set default build output directory
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

message(STATUS "Buffalo ${GIT_VERSION}")
message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "Output directory: ${OUTPUT_DIR}")
message(STATUS "Install prefix: ${INSTALL_PREFIX}")
message(STATUS "Git commit: ${GIT_COMMIT}")
message(STATUS "Build time: ${BUILD_TIME}")
message(STATUS "")
message(STATUS "Run 'cmake --build . --target usage' for available targets")
