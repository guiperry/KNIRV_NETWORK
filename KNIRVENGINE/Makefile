# KNIRVENGINE Makefile
# Orchestrates all build, test, and deployment processes for the three-engine cognitive architecture

.PHONY: help build test deploy clean install dev docs

# Default target
help:
	@echo "🚀 KNIRVENGINE Build System"
	@echo "=========================="
	@echo ""
	@echo "Available targets:"
	@echo ""
	@echo "📦 Build Commands:"
	@echo "  build          - Build all components (desktop-client with integrated wallet)"
	@echo "  build-desktop  - Build desktop-client Go backend"
	@echo "  build-frontend - Build React frontend"
	@echo "  build-all      - Build complete desktop application"
	@echo ""
	@echo "🧪 Test Commands:"
	@echo "  test           - Run all tests"
	@echo "  test-wallet    - Test wallet integration"
	@echo "  test-controller - Test KNIRVCONTROLLER linking"
	@echo ""
	@echo "🚀 Development Commands:"
	@echo "  dev            - Start development environment"
	@echo "  start          - Start production system"
	@echo "  deploy         - Deploy to production"
	@echo ""
	@echo "🧹 Maintenance Commands:"
	@echo "  clean          - Clean all build artifacts"
	@echo "  install        - Install all dependencies"
	@echo "  docs           - Generate documentation"
	@echo ""

# Install all dependencies
install:
	@echo "📦 Installing dependencies..."
	@cd desktop-client && go mod download
	@cd desktop-client/gui && npm install
	@echo "✅ Dependencies installed"

# Build all components
build: build-all
	@echo "🎉 KNIRVENGINE built successfully!"

# Build desktop-client Go backend
build-desktop:
	@echo "🖥️  Building desktop-client backend..."
	@cd desktop-client && go build -ldflags="-w -s" -o desktop-client main.go
	@echo "✅ Desktop backend built"

# Build React frontend
build-frontend:
	@echo "🌐 Building React frontend..."
	@cd desktop-client/gui && npm run build
	@echo "✅ Frontend built"

# Build complete desktop application
build-all: build-frontend build-desktop
	@echo "🎯 Complete desktop application built"

# Run all tests
test: test-wallet test-controller
	@echo "🎉 All tests completed!"

# Test wallet integration
test-wallet:
	@echo "💰 Testing wallet integration..."
	@cd desktop-client && go test ./api/... -v -run TestWallet

# Test KNIRVCONTROLLER linking
test-controller:
	@echo "🔗 Testing KNIRVCONTROLLER linking..."
	@cd desktop-client && go test ./api/... -v -run TestController

# Start development environment
dev:
	@echo "🛠️  Starting development environment..."
	@echo "Building frontend first..."
	@cd desktop-client/gui && npm run build
	@echo "Starting desktop-client..."
	@cd desktop-client && go run main.go

# Start production system
start: build-all
	@echo "🚀 Starting production system..."
	@cd desktop-client && ./desktop-client

# Deploy to production
deploy: build-all
	@echo "🚀 Deploying to production..."
	@echo "✅ Build completed - ready for deployment"

# Clean all build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@cd desktop-client && rm -f desktop-client
	@cd desktop-client/gui && rm -rf dist/
	@cd desktop-client/gui && rm -rf node_modules/.cache/
	@echo "✅ Clean completed"

# Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@echo "Documentation available in:"
	@echo "  - README.md (main documentation with integrated wallet info)"
	@echo "  - desktop-client/README.md (desktop client details)"
	@echo "  - desktop-client/docs/ (API documentation)"

# Development shortcuts
.PHONY: quick-build quick-test

# Quick build (Go binary only)
quick-build:
	@echo "⚡ Quick build (Go binary only)..."
	@cd desktop-client && go build -o desktop-client main.go
	@echo "✅ Quick build completed"

# Quick test (essential tests only)
quick-test:
	@echo "⚡ Quick test..."
	@cd desktop-client && go test ./... -short

# Platform-specific builds
.PHONY: build-linux build-windows build-macos

build-linux:
	@echo "🐧 Building for Linux..."
	@cd desktop-client && GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o desktop-client-linux main.go

build-windows:
	@echo "🪟 Building for Windows..."
	@cd desktop-client && GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o desktop-client-windows.exe main.go

build-macos:
	@echo "🍎 Building for macOS..."
	@cd desktop-client && GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o desktop-client-macos main.go

# Check system requirements
check:
	@echo "🔍 Checking system requirements..."
	@echo -n "Go: "; go version 2>/dev/null || echo "❌ Not installed"
	@echo -n "Node.js: "; node --version 2>/dev/null || echo "❌ Not installed"
	@echo -n "npm: "; npm --version 2>/dev/null || echo "❌ Not installed"

# Version information
version:
	@echo "📋 KNIRVENGINE Version Information"
	@echo "================================="
	@echo "Version: 2.0.0"
	@echo "Build Date: $(shell date)"
	@echo "Components:"
	@echo "  - Desktop-Client: Go backend with integrated wallet"
	@echo "  - React Frontend: TypeScript/React GUI"
	@echo "  - Wallet Integration: KNIRVCONTROLLER linking"
	@echo "  - Multi-chain Support: BTC, ETH, SOL, NRN"
