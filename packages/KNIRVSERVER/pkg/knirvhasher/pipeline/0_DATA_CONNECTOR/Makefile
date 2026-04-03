# Data Connector Makefile
# Data Connection Engine - Data source connection and ingestion pipeline

# Project configuration
PROJECT_NAME := data-connector
BINARY_NAME := data-connector
MAIN_PACKAGE := ./cmd/$(PROJECT_NAME)
BIN_DIR := bin
BUILD_DIR := build
DIST_DIR := dist

# Go configuration
GO_FILES := $(shell find . -name '*.go' -type f)
GO_VERSION := $(shell go version | awk '{print $$3}')
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
BUILD_FLAGS := -trimpath -buildmode=pie $(LDFLAGS)

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Default target
.PHONY: all
all: clean deps build

# Help target
.PHONY: help
help:
	@echo "Data Connector - Data Connection Engine"
	@echo "======================================="
	@echo ""
	@echo "Available targets:"
	@echo "  all          Clean, deps, and build"
	@echo "  build        Build the binary"
	@echo "  build-all    Build for all platforms"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Download dependencies"
	@echo "  test         Run tests"
	@echo "  test-cover   Run tests with coverage"
	@echo "  lint         Run linter"
	@echo "  fmt          Format code"
	@echo "  vet          Run go vet"
	@echo "  install      Install binary to system"
	@echo "  uninstall    Remove binary from system"
	@echo "  run          Run with default configuration"
	@echo "  run-prod     Run in production mode"
	@echo "  run-test     Run in test mode"
	@echo "  run-opt      Run in optimized mode"
	@echo "  run-hybrid   Run in hybrid mode"
	@echo "  run-workflow Run integrated workflow"
	@echo "  dev          Development build and run"
	@echo "  debug        Debug mode with dry-run"
	@echo "  monitor     Production mode with monitoring"
	@echo "  status       Show project status"
	@echo "  kill-servers Kill all background processes"
	@echo "  release      Create release build"
	@echo "  docker       Build Docker image"
	@echo "  docs         Generate documentation"
	@echo ""
	@echo "Application Commands:"
	@echo "  run-workflow    Run production workflow"
	@echo "  test-workflow   Run test workflow"
	@echo "  run-optimized   Run with optimizations"
	@echo "  run-hybrid      Run with hybrid connections"
	@echo "  build           Build the project"
	@echo "  deps            Check dependencies"
	@echo "  clean           Clean build artifacts"
	@echo "  help            Show application help"
	@echo "  list-modes      List available modes"
	@echo ""
	@echo "Execution Modes:"
	@echo "  production      Production workflow mode"
	@echo "  test            Test mode with limited scope"
	@echo "  optimized       CPU/GPU optimized mode"
	@echo "  hybrid          Hybrid connection mode"
	@echo "  workflow        Integrated workflow"
	@echo ""
	@echo "Examples:"
	@echo "  make run                    # Run with default config"
	@echo "  make run-prod               # Production mode"
	@echo "  make run-test INPUT=./test  # Test mode with custom input"
	@echo "  make build-all              # Build for all platforms"

# Dependencies
.PHONY: deps
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies updated"

# Build targets
.PHONY: build
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Build completed: $(BIN_DIR)/$(BINARY_NAME)"

# Build for all platforms
.PHONY: build-all
build-all:
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	
	@echo "✅ Multi-platform build completed in $(DIST_DIR)/"

# Development build
.PHONY: dev
dev: build
	@echo "🚀 Running in development mode..."
	$(BIN_DIR)/$(BINARY_NAME) -test -dry-run

# Clean targets
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f $(BINARY_NAME)
	@echo "✅ Clean completed"

# Test targets
.PHONY: test
test:
	@echo "🧪 Running tests..."
	go test -v ./...

.PHONY: test-cover
test-cover:
	@echo "🧪 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# Code quality targets
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "🔍 Running go vet..."
	go vet ./...

.PHONY: lint
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Installation targets
.PHONY: install
install: build
	@echo "📦 Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installation completed"

.PHONY: uninstall
uninstall:
	@echo "🗑️  Removing $(BINARY_NAME) from system..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Uninstallation completed"

# Application run targets
.PHONY: run
run: build
	@echo "🚀 Running Data Connector with default configuration..."
	$(BIN_DIR)/$(BINARY_NAME) run

.PHONY: run-prod
run-prod: build
	@echo "🚀 Running Data Connector in production mode..."
	$(BIN_DIR)/$(BINARY_NAME) production

.PHONY: run-test
run-test: build
	@echo "🧪 Running Data Connector in test mode..."
	$(BIN_DIR)/$(BINARY_NAME) test

.PHONY: run-opt
run-opt: build
	@echo "⚡ Running Data Connector in optimized mode..."
	$(BIN_DIR)/$(BINARY_NAME) optimized

.PHONY: run-hybrid
run-hybrid: build
	@echo "🔄 Running Data Connector in hybrid mode..."
	$(BIN_DIR)/$(BINARY_NAME) hybrid

.PHONY: run-workflow
run-workflow: build
	@echo "🔄 Running Data Connector in workflow mode..."
	$(BIN_DIR)/$(BINARY_NAME) -production

# Application command targets (using script commands)
.PHONY: app-run-workflow
app-run-workflow: build
	@echo "🏭 Running production workflow command..."
	$(BIN_DIR)/$(BINARY_NAME) run-workflow

.PHONY: app-test-workflow
app-test-workflow: build
	@echo "🧪 Running test workflow command..."
	$(BIN_DIR)/$(BINARY_NAME) test-workflow

.PHONY: app-run-optimized
app-run-optimized: build
	@echo "⚡ Running optimized workflow command..."
	$(BIN_DIR)/$(BINARY_NAME) run-optimized

.PHONY: app-run-hybrid
app-run-hybrid: build
	@echo "🔄 Running hybrid workflow command..."
	$(BIN_DIR)/$(BINARY_NAME) run-hybrid

.PHONY: app-help
app-help: build
	@echo "📖 Showing application help..."
	$(BIN_DIR)/$(BINARY_NAME) help

.PHONY: list-modes
list-modes: build
	@echo "📋 Listing available modes..."
	$(BIN_DIR)/$(BINARY_NAME) list-modes

.PHONY: check-deps
check-deps: build
	@echo "🔍 Checking dependencies..."
	$(BIN_DIR)/$(BINARY_NAME) deps

.PHONY: app-clean
app-clean: build
	@echo "🧹 Cleaning application artifacts..."
	$(BIN_DIR)/$(BINARY_NAME) clean

# Custom run with parameters
.PHONY: run-custom
run-custom: build
	@if [ -z "$(INPUT)" ]; then \
		echo "❌ INPUT parameter required"; \
		echo "Usage: make run-custom INPUT=/path/to/data [OUTPUT=/path/to/output]"; \
		exit 1; \
	fi
	@echo "🚀 Running Data Connector with custom configuration..."
	$(BIN_DIR)/$(BINARY_NAME) -input "$(INPUT)" -output "$(OUTPUT)"

# Release build
.PHONY: release
release: clean fmt vet lint test build-all
	@echo "📦 Creating release package..."
	@mkdir -p $(DIST_DIR)/release
	@cp -r $(DIST_DIR)/* $(DIST_DIR)/release/
	@echo "✅ Release package created in $(DIST_DIR)/release/"

# Docker targets
.PHONY: docker
docker:
	@echo "🐳 Building Docker image..."
	docker build -t $(PROJECT_NAME):$(VERSION) .
	docker tag $(PROJECT_NAME):$(VERSION) $(PROJECT_NAME):latest
	@echo "✅ Docker image built: $(PROJECT_NAME):$(VERSION)"

.PHONY: docker-run
docker-run:
	@echo "🐳 Running Docker container..."
	docker run --rm -it \
		-v $(PWD)/data:/app/data \
		$(PROJECT_NAME):latest

# Documentation
.PHONY: docs
docs:
	@echo "📚 Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "🌐 Documentation available at: http://localhost:6060/pkg/"; \
		godoc -http=:6060 & \
		sleep 2 && \
		echo "📖 Opening documentation in browser..." && \
		xdg-open http://localhost:6060/pkg/ 2>/dev/null || \
		open http://localhost:6060/pkg/ 2>/dev/null || \
		echo "🌐 Visit http://localhost:6060/pkg/ to view documentation"; \
	else \
		echo "⚠️  godoc not found, installing..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
		$(MAKE) docs; \
	fi

# Development utilities
.PHONY: watch
watch:
	@echo "👀 Watching for changes and rebuilding..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -r $(MAKE) build; \
	else \
		echo "⚠️  entr not found, install with: brew install entr or apt-get install entr"; \
	fi

.PHONY: benchmark
benchmark:
	@echo "⏱️  Running benchmarks..."
	go test -bench=. -benchmem ./...

# Security
.PHONY: security
security:
	@echo "🔒 Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec not found, installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Project information
.PHONY: info
info:
	@echo "📊 Project Information:"
	@echo "  Name: $(PROJECT_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  OS/Arch: $(GOOS)/$(GOARCH)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Binary: $(BIN_DIR)/$(BINARY_NAME)"

# Debug and monitoring targets
.PHONY: debug
debug: build
	@echo "🐛 Running Data Connector in debug mode..."
	$(BIN_DIR)/$(BINARY_NAME) -test -dry-run

.PHONY: monitor
monitor: build
	@echo "📊 Running Data Connector with monitoring..."
	$(BIN_DIR)/$(BINARY_NAME) -production

.PHONY: status
status:
	@echo "📊 Project Status:"
	@echo "================="
	@if [ -f $(BIN_DIR)/$(BINARY_NAME) ]; then \
		echo "✅ Binary: $(BIN_DIR)/$(BINARY_NAME)"; \
		echo "📅 Build: $(shell date -r $(BIN_DIR)/$(BINARY_NAME) '+%Y-%m-%d %H:%M:%S')"; \
	else \
		echo "❌ Binary not found - run 'make build'"; \
	fi

.PHONY: kill-servers
kill-servers:
	@echo "🛑 Killing background servers..."
	./scripts/kill_servers.sh

# Quick development shortcuts
.PHONY: q
q: build run

.PHONY: qc
qc: clean build run

.PHONY: qt
qt: clean build test

# Prevent make from deleting intermediate files
.PRECIOUS: $(GO_FILES)

# Default shell
SHELL := /bin/bash

# Enable secondary expansion for advanced patterns
.SECONDEXPANSION: