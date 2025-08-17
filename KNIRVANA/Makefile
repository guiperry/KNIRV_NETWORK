# KNIRVANA Gaming Application Test Suite
# Comprehensive testing for both Rust (on-device) and TypeScript (web portal) implementations

.PHONY: help test test-rust test-ts test-all clean setup install-deps
.PHONY: test-unit test-integration test-performance test-e2e
.PHONY: rust-unit rust-integration rust-performance rust-mobile
.PHONY: ts-unit ts-integration ts-e2e ts-performance
.PHONY: coverage coverage-rust coverage-ts
.PHONY: lint lint-rust lint-ts format format-rust format-ts
.PHONY: docs build build-rust build-ts
.PHONY: dev dev-rust dev-ts

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
	@echo "  test              - Run all tests (both Rust and TypeScript)"
	@echo "  test-rust         - Run all Rust tests"
	@echo "  test-ts           - Run all TypeScript tests"
	@echo "  test-unit         - Run unit tests for both implementations"
	@echo "  test-integration  - Run integration tests for both implementations"
	@echo "  test-performance  - Run performance tests for both implementations"
	@echo "  test-e2e          - Run end-to-end tests"
	@echo ""
	@echo "Rust-specific:"
	@echo "  rust-unit         - Run Rust unit tests"
	@echo "  rust-integration  - Run Rust integration tests"
	@echo "  rust-performance  - Run Rust performance tests"
	@echo "  rust-mobile       - Run Rust mobile-specific tests"
	@echo ""
	@echo "TypeScript-specific:"
	@echo "  ts-unit           - Run TypeScript unit tests"
	@echo "  ts-integration    - Run TypeScript integration tests"
	@echo "  ts-e2e            - Run TypeScript E2E tests"
	@echo "  ts-performance    - Run TypeScript performance tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  coverage          - Generate coverage reports for both implementations"
	@echo "  coverage-rust     - Generate Rust coverage report"
	@echo "  coverage-ts       - Generate TypeScript coverage report"
	@echo "  lint              - Run linters for both implementations"
	@echo "  lint-rust         - Run Rust linter (clippy)"
	@echo "  lint-ts           - Run TypeScript linter (eslint)"
	@echo "  format            - Format code for both implementations"
	@echo "  format-rust       - Format Rust code (rustfmt)"
	@echo "  format-ts         - Format TypeScript code (prettier)"
	@echo ""
	@echo "Development:"
	@echo "  build             - Build both implementations"
	@echo "  build-rust        - Build Rust client"
	@echo "  build-ts          - Build TypeScript client"
	@echo "  dev               - Start development servers for both"
	@echo "  dev-rust          - Start Rust development build"
	@echo "  dev-ts            - Start TypeScript development server"
	@echo "  clean             - Clean build artifacts"
	@echo "  docs              - Generate documentation"

# Setup and Dependencies
setup: install-deps
	@echo "Setting up KNIRVANA test environment..."
	@chmod +x scripts/*.sh
	@echo "Setup complete!"

install-deps:
	@echo "Installing dependencies..."
	@echo "Installing Rust dependencies..."
	@cd rust-client && cargo fetch
	@echo "Installing TypeScript dependencies..."
	@cd ts-client && npm install
	@echo "Dependencies installed!"

# Main test targets
test: test-rust test-ts
	@echo "All tests completed!"

test-all: lint format test coverage
	@echo "Full test suite completed!"

# Rust tests
test-rust: rust-unit rust-integration rust-performance
	@echo "All Rust tests completed!"

rust-unit:
	@echo "Running Rust unit tests..."
	@cd rust-client && cargo test --lib

rust-integration:
	@echo "Running Rust integration tests..."
	@cd rust-client && cargo test --test '*'

rust-performance:
	@echo "Running Rust performance tests..."
	@cd rust-client && cargo test --release --test performance_tests

rust-mobile:
	@echo "Running Rust mobile-specific tests..."
	@cd rust-client && cargo test --features mobile --test mobile_tests

# TypeScript tests
test-ts: ts-unit ts-integration ts-e2e
	@echo "All TypeScript tests completed!"

ts-unit:
	@echo "Running TypeScript unit tests..."
	@cd ts-client && npm run test:unit

ts-integration:
	@echo "Running TypeScript integration tests..."
	@cd ts-client && npm run test:integration

ts-e2e:
	@echo "Running TypeScript E2E tests..."
	@cd ts-client && npm run test:e2e

ts-performance:
	@echo "Running TypeScript performance tests..."
	@cd ts-client && npm run test:performance

# Combined test categories
test-unit: rust-unit ts-unit
test-integration: rust-integration ts-integration
test-performance: rust-performance ts-performance

# Coverage
coverage: coverage-rust coverage-ts
	@echo "Coverage reports generated!"

coverage-rust:
	@echo "Generating Rust coverage report..."
	@cd rust-client && cargo tarpaulin --out Html --output-dir ../coverage/rust

coverage-ts:
	@echo "Generating TypeScript coverage report..."
	@cd ts-client && npm run test:coverage

# Code quality
lint: lint-rust lint-ts

lint-rust:
	@echo "Running Rust linter..."
	@cd rust-client && cargo clippy -- -D warnings

lint-ts:
	@echo "Running TypeScript linter..."
	@cd ts-client && npm run lint

format: format-rust format-ts

format-rust:
	@echo "Formatting Rust code..."
	@cd rust-client && cargo fmt

format-ts:
	@echo "Formatting TypeScript code..."
	@cd ts-client && npm run format

# Build targets
build: build-rust build-ts

build-rust:
	@echo "Building Rust client..."
	@cd rust-client && cargo build --release

build-ts:
	@echo "Building TypeScript client..."
	@cd ts-client && npm run build

# Development
dev-rust:
	@echo "Starting Rust development build..."
	@cd rust-client && cargo run

dev-ts:
	@echo "Starting TypeScript development server..."
	@cd ts-client && npm run dev

dev:
	@echo "Starting both development environments..."
	@echo "Note: Run 'make dev-rust' and 'make dev-ts' in separate terminals"

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@cd rust-client && cargo clean
	@cd ts-client && rm -rf dist node_modules/.cache
	@rm -rf coverage/
	@echo "Clean complete!"

# Documentation
docs:
	@echo "Generating documentation..."
	@cd rust-client && cargo doc --no-deps
	@cd ts-client && npm run docs
	@echo "Documentation generated!"

# Test data and utilities
test-data:
	@echo "Setting up test data..."
	@./scripts/setup-test-data.sh

# Continuous Integration helpers
ci-test: lint test coverage
	@echo "CI test suite completed!"

# Watch mode for development
watch-rust:
	@echo "Watching Rust tests..."
	@cd rust-client && cargo watch -x test

watch-ts:
	@echo "Watching TypeScript tests..."
	@cd ts-client && npm run test:watch
