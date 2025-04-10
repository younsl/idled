# Makefile for idled

# Variables
BINARY_NAME=idled
BUILD_DIR=bin
CMD_DIR=cmd/$(BINARY_NAME)
GOFLAGS=-v
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)

.PHONY: all build clean fmt test run install help

# Default target
all: clean fmt test build

# Build binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
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