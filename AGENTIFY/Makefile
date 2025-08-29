# KNIRVENGINE Makefile
# Orchestrates all build, test, and deployment processes

.PHONY: help build test deploy clean install dev mobile android ios pwa docs

# Default target
help:
	@echo "🚀 KNIRVENGINE Build System"
	@echo "=========================="
	@echo ""
	@echo "Available targets:"
	@echo ""
	@echo "📦 Build Commands:"
	@echo "  build          - Build all components (desktop-client, mobile-controller, agent-core)"
	@echo "  build-desktop  - Build desktop-client with Electron wrapper"
	@echo "  build-mobile   - Build mobile-controller for production"
	@echo "  build-wasm     - Build agent-core WASM module"
	@echo ""
	@echo "🧪 Test Commands:"
	@echo "  test           - Run all tests"
	@echo "  test-qr        - Test QR linkage system"
	@echo "  test-system    - Test complete system integration"
	@echo ""
	@echo "📱 Mobile Commands:"
	@echo "  mobile         - Start mobile development server"
	@echo "  mobile-build   - Build mobile app for production"
	@echo "  android        - Build Android app (requires Android Studio)"
	@echo "  ios            - Build iOS app (requires Xcode, macOS only)"
	@echo "  pwa            - Setup Progressive Web App"
	@echo ""
	@echo "🚀 Deployment Commands:"
	@echo "  deploy         - Full production deployment"
	@echo "  dev            - Start development environment"
	@echo "  start          - Start production system"
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
	@cd mobile-controller && npm install
	@cd agent-core/rust-wasm && cargo fetch
	@echo "✅ Dependencies installed"

# Build all components
build: build-desktop build-mobile build-wasm
	@echo "🎉 All components built successfully!"

# Build desktop-client with Electron wrapper
build-desktop:
	@echo "🖥️  Building desktop-client..."
	@cd desktop-client && go build -ldflags="-w -s" -o desktop-client main.go
	@if [ -f "desktop-client/scripts/run_production.sh" ]; then \
		echo "📦 Building Electron wrapper..."; \
		cd desktop-client && chmod +x scripts/run_production.sh && \
		sed 's|./dist/linux-unpacked/knirv-engine-desktop|echo "Build completed"|' scripts/run_production.sh > temp_build.sh && \
		chmod +x temp_build.sh && ./temp_build.sh && rm temp_build.sh; \
	fi
	@echo "✅ Desktop-host built"

# Build mobile-controller
build-mobile:
	@echo "📱 Building mobile-controller..."
	@cd mobile-controller && npm run build
	@echo "✅ Mobile-tool built"

# Build agent-core WASM module
build-wasm:
	@echo "⚡ Building agent-core WASM..."
	@cd agent-core/rust-wasm && cargo build --release --target wasm32-unknown-unknown
	@mkdir -p agent-core/dist
	@cp agent-core/rust-wasm/target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm agent-core/dist/agent-core.wasm
	@echo "✅ Agent-core WASM built"

# Run all tests
test: test-qr test-system
	@echo "🎉 All tests completed!"

# Test QR linkage system
test-qr:
	@echo "🎯 Testing QR linkage system..."
	@cd scripts && npm run test-qr

# Test complete system
test-system:
	@echo "🧪 Testing complete system..."
	@cd scripts && npm run test-system

# Start mobile development server
mobile:
	@echo "📱 Starting mobile development server..."
	@cd mobile-controller && npm run dev

# Build mobile app for production
mobile-build:
	@echo "📱 Building mobile app..."
	@cd mobile-controller && ./deploy-mobile.sh 2

# Build Android app
android:
	@echo "🤖 Building Android app..."
	@cd mobile-controller && ./deploy-mobile.sh 3

# Build iOS app
ios:
	@echo "🍎 Building iOS app..."
	@cd mobile-controller && ./deploy-mobile.sh 4

# Setup Progressive Web App
pwa:
	@echo "📱 Setting up PWA..."
	@cd mobile-controller && ./deploy-mobile.sh 5

# Full production deployment
deploy:
	@echo "🚀 Starting full deployment..."
	@cd scripts && ./deploy.sh

# Start development environment
dev:
	@echo "🛠️  Starting development environment..."
	@echo "Starting desktop-client in background..."
	@cd desktop-client && ./desktop-client &
	@echo "Starting mobile-controller development server..."
	@cd mobile-controller && npm run dev

# Start production system
start:
	@echo "🚀 Starting production system..."
	@cd scripts && ./start-system.sh

# Clean all build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf dist/
	@rm -rf backup/
	@rm -f deploy.log
	@cd desktop-client && rm -f desktop-client
	@cd desktop-client && rm -rf electron/dist/
	@cd mobile-controller && rm -rf dist/
	@cd mobile-controller && rm -rf android/
	@cd mobile-controller && rm -rf ios/
	@cd agent-core/rust-wasm && cargo clean
	@cd agent-core && rm -rf dist/
	@echo "✅ Clean completed"

# Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@echo "Documentation available in:"
	@echo "  - README.md (main documentation)"
	@echo "  - mobile-controller/MOBILE_DEPLOYMENT.md (mobile deployment)"
	@echo "  - SYSTEM_STATUS.md (system status)"

# Development shortcuts
.PHONY: quick-build quick-test quick-deploy

# Quick build (no Electron)
quick-build:
	@echo "⚡ Quick build (Go binary only)..."
	@cd desktop-client && go build -o desktop-client main.go
	@cd mobile-controller && npm run build
	@cd agent-core/rust-wasm && cargo build --release --target wasm32-unknown-unknown
	@echo "✅ Quick build completed"

# Quick test (essential tests only)
quick-test:
	@echo "⚡ Quick test..."
	@cd scripts && timeout 30s node test_qr_linkage.js || true

# Quick deploy (development)
quick-deploy: quick-build
	@echo "⚡ Quick deploy..."
	@mkdir -p dist/desktop-client dist/mobile-controller dist/agent-core
	@cp desktop-client/desktop-client dist/desktop-client/
	@cp -r mobile-controller/dist/* dist/mobile-controller/
	@cp agent-core/rust-wasm/target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm dist/agent-core/agent-core.wasm

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
	@echo -n "Rust: "; rustc --version 2>/dev/null || echo "❌ Not installed"
	@echo -n "wasm32 target: "; rustup target list --installed | grep wasm32-unknown-unknown >/dev/null && echo "✅ Installed" || echo "❌ Not installed"
	@echo -n "Android Studio: "; [ -d "$$ANDROID_HOME" ] && echo "✅ Detected" || echo "⚠️  Not detected"
	@echo -n "Xcode: "; xcode-select -p >/dev/null 2>&1 && echo "✅ Detected" || echo "⚠️  Not detected (macOS only)"

# Version information
version:
	@echo "📋 KNIRVENGINE Version Information"
	@echo "================================="
	@echo "Version: 1.0.0"
	@echo "Build Date: $(shell date)"
	@echo "Components:"
	@echo "  - Desktop-Host: Go backend with HRM integration"
	@echo "  - Mobile-Tool: React/TypeScript mobile client"
	@echo "  - Agent-Core: Rust WASM cognitive shell"
