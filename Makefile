# Makefile for idled

# Variables
DEFAULT_BINARY_NAME=idled
# Use GOOS/GOARCH env vars if set, otherwise use go env defaults
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
# Set the final binary name based on OS and Arch
BINARY_NAME ?= $(DEFAULT_BINARY_NAME)-$(GOOS)-$(GOARCH)

BUILD_DIR=bin
CMD_DIR=cmd/$(DEFAULT_BINARY_NAME) # Use default name for cmd path
GOFLAGS=-v
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)

# --- Version Information --- 
# Get version from the latest Git tag
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
# Get the build date in RFC3339 format (UTC)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
# Get short commit hash
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
# Go package path for version variables
VERSION_PKG = github.com/younsl/idled/internal/version

# ldflags for injecting version info into unexported variables
LDFLAGS := -ldflags="\
-X $(VERSION_PKG).version=$(VERSION) \
-X $(VERSION_PKG).buildDate=$(BUILD_DATE) \
-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT)"
# --- End Version Information ---

.PHONY: all build clean fmt test run install help

# Default target
all: clean fmt test build

# Build binary (respects GOOS/GOARCH env vars for cross-compilation)
build:
	@echo "Building $(BINARY_NAME) version $(VERSION) (commit: $(GIT_COMMIT), built: $(BUILD_DATE))..."
	@mkdir -p $(BUILD_DIR)
	# GOOS/GOARCH env vars are respected by go build
	# Output binary name is now dynamic: $(BINARY_NAME)
	@go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w $(GOFMT_FILES)
	@echo "Format complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Tests complete"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Install binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install ./$(CMD_DIR)
	@echo "Installation complete"

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@echo "Dependencies updated"

# Show help
help:
	@echo "Available targets:"
	@echo "  make          - Build after cleaning, formatting and testing"
	@echo "  make build    - Build the binary"
	@echo "  make clean    - Remove build artifacts" 
	@echo "  make fmt      - Format code"
	@echo "  make test     - Run tests"
	@echo "  make run      - Build and run the application"
	@echo "  make install  - Install binary to GOPATH/bin"
	@echo "  make deps     - Update dependencies"
	@echo "  make help     - Show this help message" 