.PHONY: build run test clean android ios docs fmt lint

# Default target
all: build

# Build the game client
build:
	cargo build --release

# Run the game client
run:
	cargo run --release

# Run with mobile features
run-mobile:
	cargo run --release --features mobile

# Run the basic example
example:
	cargo run --example basic_game

# Run tests
test:
	cargo test

# Run tests with output
test-verbose:
	cargo test -- --nocapture

# Clean build artifacts
clean:
	cargo clean

# Build for Android
android:
	@echo "Building for Android..."
	@if [ -z "$$ANDROID_NDK_ROOT" ]; then \
		echo "Error: ANDROID_NDK_ROOT environment variable not set"; \
		exit 1; \
	fi
	cargo install cargo-apk
	cargo apk build --release

# Build for iOS
ios:
	@echo "Building for iOS..."
	cargo install cargo-lipo
	cargo lipo --release

# Generate documentation
docs:
	cargo doc --open

# Format code
fmt:
	cargo fmt

# Lint code
lint:
	cargo clippy -- -D warnings

# Check code without building
check:
	cargo check

# Install dependencies
deps:
	rustup update
	cargo update

# Development build (faster compilation)
dev:
	cargo build

# Run with debug logging
debug:
	RUST_LOG=debug cargo run

# Profile the application
profile:
	cargo build --release
	perf record --call-graph=dwarf ./target/release/knirvana_game
	perf report

# Benchmark
bench:
	cargo bench

# Security audit
audit:
	cargo audit

# Update dependencies
update:
	cargo update

# Install the game client
install:
	cargo install --path .

# Uninstall the game client
uninstall:
	cargo uninstall knirvana-game-client

# Create a release build with optimizations
release:
	cargo build --release --target-cpu=native

# Run integration tests
integration-test:
	cargo test --test integration_test

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the game client"
	@echo "  run            - Run the game client"
	@echo "  run-mobile     - Run with mobile features"
	@echo "  example        - Run the basic example"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  android        - Build for Android"
	@echo "  ios            - Build for iOS"
	@echo "  docs           - Generate documentation"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  help           - Show this help"
