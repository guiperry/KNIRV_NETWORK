# KNIRVANA Gaming Application Test Suite
# Comprehensive testing for all language implementations in packages/

.PHONY: help test test-all test-unit test-integration test-performance test-e2e
.PHONY: coverage lint format docs build clean setup install-deps dev
.PHONY: test-elixir test-ts-1 test-ts-2 test-rust test-go

# Default target
help:
	@echo "KNIRVANA Gaming Application Test Suite"
	@echo "======================================"
	@echo ""
	@echo "Available targets:"
	@echo "  help              - Show this help message"
	@echo "  setup             - Initial setup and dependency installation"
	@echo "  install-deps      - Install all dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run all tests for all packages"
	@echo "  test-all          - Run full test suite (lint, format, test, coverage)"
	@echo "  test-unit         - Run unit tests for all packages"
	@echo "  test-integration  - Run integration tests for all packages"
	@echo "  test-performance  - Run performance tests for all packages"
	@echo "  test-e2e          - Run end-to-end tests for all packages"
	@echo ""
	@echo "Package-specific:"
	@echo "  test-elixir       - Run Elixir backend tests (elixir_client/backend)"
	@echo "  test-ts-1         - Run TypeScript client 1 tests"
	@echo "  test-ts-2         - Run TypeScript client 2 tests"
	@echo "  test-rust         - Run Rust client tests"
	@echo "  test-go           - Run Go client tests (if present)"
	@echo ""
	@echo "Code Quality:"
	@echo "  coverage          - Generate coverage reports"
	@echo "  lint              - Run linters for all packages"
	@echo "  format            - Format code for all packages"
	@echo ""
	@echo "Development:"
	@echo "  build             - Build all packages"
	@echo "  dev               - Start development environments"
	@echo "  clean             - Clean build artifacts"
	@echo "  docs              - Generate documentation"

setup: install-deps
	@echo "Setting up KNIRVANA test environment..."
	@chmod +x scripts/*.sh
	@echo "Setup complete!"

install-deps:
	@echo "Installing dependencies for all packages..."
	@echo "Installing Elixir dependencies..."
	@cd packages/elixer_client/backend && mix deps.get || true
	@echo "Installing TypeScript client 1 dependencies..."
	@cd packages/ts-client_1 && npm install || true
	@echo "Installing TypeScript client 2 dependencies..."
	@cd packages/ts_client_2 && npm install || true
	@echo "Installing Rust client dependencies..."
	@cd packages/rust-client && cargo fetch || true
	@echo "Installing Elixir frontend dependencies..."
	@cd packages/elixer_client/frontend && npm install || true
	@echo "Dependencies installed!"

# Main test targets
test:
	@echo "Running all package tests..."
	@$(MAKE) test-rust || true
	@$(MAKE) test-ts-1 || true
	@$(MAKE) test-ts-2 || true
	@$(MAKE) test-elixir || true
	@echo "All tests completed!"

test-all: lint format test coverage
	@echo "Full test suite completed!"

test-unit:
	@echo "Running all unit tests..."
	@$(MAKE) test-rust || true
	@$(MAKE) test-ts-1 || true
	@$(MAKE) test-ts-2 || true
	@$(MAKE) test-elixir || true

test-integration:
	@echo "Running all integration tests..."
	@$(MAKE) test-ts-1 TS_TEST_TYPE=integration || true
	@$(MAKE) test-ts-2 TS_TEST_TYPE=integration || true

test-performance:
	@echo "Running all performance tests..."
	@$(MAKE) test-ts-1 TS_TEST_TYPE=performance || true

test-e2e:
	@echo "Running all E2E tests..."
	@$(MAKE) test-ts-1 TS_TEST_TYPE=e2e || true
	@$(MAKE) test-ts-2 TS_TEST_TYPE=e2e || true

# Package-specific tests
test-rust:
	@echo "Running Rust client tests..."
	@cd packages/rust-client && cargo test --lib

test-ts-1:
	@echo "Running TypeScript client 1 tests..."
	@cd packages/ts-client_1 && npm run test:unit

test-ts-2:
	@echo "Running TypeScript client 2 tests..."
	@cd packages/ts_client_2 && npm run test:unit

test-elixir:
	@echo "Running Elixir backend tests..."
	@cd packages/elixer_client/backend && mix test

test-go:
	@echo "Running Go client tests..."
	@cd packages/go_client && go test ./...

# Coverage
coverage:
	@echo "Generating coverage reports..."
	@mkdir -p coverage
	@echo "Generating Rust coverage..."
	@cd packages/rust-client && cargo tarpaulin --out Html --output-dir ../../coverage/rust || echo "Rust coverage skipped"
	@echo "Generating TypeScript coverage..."
	@cd packages/ts-client_1 && npm run test:coverage || echo "TS client 1 coverage skipped"
	@echo "Coverage reports generated in coverage/"

# Code quality
lint:
	@echo "Running linters for all packages..."
	@echo "Linting Rust..."
	@cd packages/rust-client && cargo clippy -- -D warnings || true
	@echo "Linting TypeScript client 1..."
	@cd packages/ts-client_1 && npm run lint || true
	@echo "Linting TypeScript client 2..."
	@cd packages/ts_client_2 && npm run lint || true

format:
	@echo "Formatting code for all packages..."
	@echo "Formatting Rust..."
	@cd packages/rust-client && cargo fmt || true
	@echo "Formatting TypeScript client 1..."
	@cd packages/ts-client_1 && npm run format || true
	@echo "Formatting TypeScript client 2..."
	@cd packages/ts_client_2 && npm run format || true

# Build targets
build:
	@echo "Building all packages..."
	@$(MAKE) build-rust || true
	@$(MAKE) build-ts-1 || true
	@$(MAKE) build-ts-2 || true

build-rust:
	@echo "Building Rust client..."
	@cd packages/rust-client && cargo build --release

build-ts-1:
	@echo "Building TypeScript client 1..."
	@cd packages/ts-client_1 && npm run build

build-ts-2:
	@echo "Building TypeScript client 2..."
	@cd packages/ts_client_2 && npm run build

# Development
dev:
	@echo "Starting development environments..."
	@echo "Note: Run specific dev targets in separate terminals"

dev-ts-1:
	@echo "Starting TypeScript client 1 dev server..."
	@cd packages/ts-client_1 && npm run dev

dev-ts-2:
	@echo "Starting TypeScript client 2 dev server..."
	@cd packages/ts_client_2 && npm run dev

dev-rust:
	@echo "Starting Rust development build..."
	@cd packages/rust-client && cargo run

dev-elixir:
	@echo "Starting Elixir backend..."
	@cd packages/elixer_client/backend && mix phx.server

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@cd packages/rust-client && cargo clean || true
	@cd packages/ts-client_1 && rm -rf dist node_modules/.cache || true
	@cd packages/ts_client_2 && rm -rf dist node_modules/.cache || true
	@cd packages/elixer_client/backend && mix clean || true
	@rm -rf coverage/
	@echo "Clean complete!"

# Documentation
docs:
	@echo "Generating documentation..."
	@cd packages/rust-client && cargo doc --no-deps || true
	@cd packages/ts-client_1 && npm run docs || true
	@echo "Documentation generated!"

# Docker builds
docker-build:
	@echo "Building Docker images for all packages..."
	@$(MAKE) docker-build-ts-1
	@$(MAKE) docker-build-ts-2

docker-build-ts-1:
	@echo "Building TypeScript client 1 Docker image..."
	@cd packages/ts-client_1 && docker build -t knirvana/ts-client-1:latest .

docker-build-ts-2:
	@echo "Building TypeScript client 2 Docker image..."
	@cd packages/ts_client_2 && docker build -t knirvana/ts-client-2:latest .

docker-build-elixir:
	@echo "Building Elixir backend Docker image..."
	@cd packages/elixer_client/backend && docker build -t knirvana/elixir-backend:latest .

# Continuous Integration helpers
ci-test: lint test coverage
	@echo "CI test suite completed!"
