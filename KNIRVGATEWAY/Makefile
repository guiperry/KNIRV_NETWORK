# KNIRVGATEWAY Go Build Makefile
# This Makefile contains targets for building the Go version of KNIRVGATEWAY

.PHONY: build run test clean install fmt vet lint webgui-build

# Go binary output
BINARY_NAME=knirvgateway
BINARY_DIR=bin
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_CLEAN=$(GO_CMD) clean
GO_TEST=$(GO_CMD) test
GO_GET=$(GO_CMD) get
GO_FMT=$(GO_CMD) fmt
GO_VET=$(GO_CMD) vet
GO_MOD=$(GO_CMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_DIR=./cmd/gateway

# WebGUI build
WEBGUI_DIR=internal/embedded/webgui

# Default target
all: build

# Initialize Go modules
init:
	@echo "Initializing Go modules..."
	$(GO_MOD) tidy
	$(GO_MOD) download

# Install dependencies
install: init
	@echo "Installing Go dependencies..."
	$(GO_GET) -v ./...

# Format Go code
fmt:
	@echo "Formatting Go code..."
	$(GO_FMT) ./...

# Vet Go code
vet:
	@echo "Vetting Go code..."
	$(GO_VET) ./...

# Build the WebGUI
webgui-build:
	@echo "Building WebGUI..."
	@cd $(WEBGUI_DIR) && npm install --legacy-peer-deps
	@cd $(WEBGUI_DIR) && npm run build
	@echo "WebGUI build complete"

# Build the Go application
build: webgui-build fmt vet
	@echo "Building Go application..."
	@mkdir -p $(BINARY_DIR)
	$(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: webgui-build
	@echo "Building for multiple platforms..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe $(BUILD_DIR)
	@echo "Multi-platform build complete"

# Run the Go application
run: build
	@echo "Running Go application..."
	./$(BINARY_DIR)/$(BINARY_NAME)

# Run without building
dev:
	@echo "Running Go application in development mode..."
	$(GO_CMD) run $(BUILD_DIR)/main.go

# Run tests
test:
	@echo "Running Go tests..."
	$(GO_TEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running Go tests with coverage..."
	$(GO_TEST) -v -coverprofile=coverage.out ./...
	$(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning Go build artifacts..."
	$(GO_CLEAN)
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Development mode with auto-reload (requires air)
watch:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@echo "Running with auto-reload..."
	air

# Check if Go is installed
check-go:
	@which go > /dev/null || (echo "Error: Go is not installed" && exit 1)
	@echo "Go version: $$(go version)"

# Full setup
setup: check-go install
	@echo "Go setup complete"

# Production build
build-prod: webgui-build
	@echo "Building production binary..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) -a -installsuffix cgo $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	@echo "Production build complete"

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t knirvgateway:latest -f Dockerfile.go .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env knirvgateway:latest

# Help
help:
	@echo "KNIRVGATEWAY Go Build Targets:"
	@echo "  init          - Initialize Go modules"
	@echo "  install       - Install dependencies"
	@echo "  webgui-build  - Build the WebGUI frontend"
	@echo "  build         - Build WebGUI and Go application"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  build-prod    - Build production binary"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run in development mode"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  watch         - Run with auto-reload (requires air)"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  setup         - Full setup"
	@echo "  help          - Show this help"
