# Plugin Agent Server Makefile

# Variables
BINARY_NAME=plugin-server
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_DARWIN=$(BINARY_NAME)_darwin

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_UNIX) .

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS) .

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DARWIN) .

# Build for all platforms
build-all: build-linux build-windows build-darwin

# Run the server
run:
	@echo "Starting plugin agent server..."
	go run . --port 8080 --agents ./agents --name "Development Server"

# Run with custom settings
run-dev:
	@echo "Starting development server..."
	go run . --port 8081 --agents ./dev-agents --name "Dev Server" --cors

# Test the server
test:
	@echo "Testing plugin agent server..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_UNIX) $(BINARY_WINDOWS) $(BINARY_DARWIN)
	rm -rf ./agents ./dev-agents

# Create agent directories
setup:
	@echo "Setting up agent directories..."
	mkdir -p ./agents ./dev-agents

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-windows- Build for Windows"
	@echo "  build-darwin - Build for macOS"
	@echo "  build-all    - Build for all platforms"
	@echo "  run          - Run the server"
	@echo "  run-dev      - Run development server"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  setup        - Create agent directories"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  help         - Show this help"

.PHONY: all build build-linux build-windows build-darwin build-all run run-dev test clean setup deps fmt lint help
