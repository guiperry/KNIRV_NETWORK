# KNIRVBASE Rust Makefile
# Comprehensive build and development automation

.PHONY: all help build build-debug build-release clean check test test-debug test-log test-verbose lint fmt format bench bench-sla flamegraph flamegraph-bench install run run-debug run-release doc doc-open audit audit-full

# Default target
all: help

## Help target
help:
	@echo "KNIRVBASE Rust Development Makefile"
	@echo "==================================="
	@echo ""
	@echo "Build targets:"
	@echo "  build                Build debug version"
	@echo "  build-debug          Build debug version (alias)"
	@echo "  build-release        Build optimized release version"
	@echo "  install              Install binary to system"
	@echo "  clean                Remove build artifacts"
	@echo ""
	@echo "Development targets:"
	@echo "  check                Fast compilation check (no codegen)"
	@echo "  fmt / format         Format code with rustfmt"
	@echo "  lint / clippy        Run Clippy lints with strict warnings"
	@echo "  run                  Run debug build"
	@echo "  run-debug            Run debug build (alias)"
	@echo "  run-release          Run optimized release build"
	@echo ""
	@echo "Testing targets:"
	@echo "  test                 Run all tests"
	@echo "  test-debug           Run tests with debug logging"
	@echo "  test-log             Run tests with full trace logging"
	@echo "  test-verbose         Run tests with verbose output"
	@echo "  bench                Run all benchmarks"
	@echo "  bench-sla            Run SLA performance validation benchmarks"
	@echo ""
	@echo "Analysis targets:"
	@echo "  flamegraph           Generate flamegraph for main binary"
	@echo "  flamegraph-bench     Generate flamegraph for benchmark"
	@echo "  doc                  Build documentation"
	@echo "  doc-open             Build and open documentation in browser"
	@echo "  audit                Run cargo audit for security vulnerabilities"
	@echo "  audit-full           Run full security audit with advisory database"
	@echo ""

## Build targets
build: build-debug

build-debug:
	cargo build

build-release:
	cargo build --release

clean:
	cargo clean
	@rm -f flamegraph.svg
	@rm -rf target/

## Development targets
check:
	cargo check

fmt: format
format:
	cargo fmt --all

lint: clippy
clippy:
	cargo clippy --all-targets --all-features -- -D warnings -D clippy::all -D clippy::pedantic -A clippy::missing_errors_doc

run: run-debug

run-debug:
	cargo run

run-release:
	cargo run --release

install:
	cargo install --path .

## Testing targets
test:
	cargo test --all-features

test-debug:
	RUST_LOG=debug cargo test

test-log:
	RUST_LOG=trace cargo test -- --nocapture

test-verbose:
	cargo test -- --show-output

bench:
	cargo bench

bench-sla:
	cargo bench --bench sla_validation

## Analysis and Documentation
flamegraph:
	cargo flamegraph --bin knirvbase

flamegraph-bench:
	cargo flamegraph --bench vector_search

doc:
	cargo doc --no-deps

doc-open:
	cargo doc --no-deps --open

audit:
	cargo audit

audit-full:
	cargo audit --deny warnings --fetch