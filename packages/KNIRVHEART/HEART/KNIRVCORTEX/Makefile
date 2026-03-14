# KNIRV-CORTEX Build System
# Reorganized with rust/ and typescript/ directories for clear separation

.PHONY: all proto-gen build-cortex build-forge test-cortex validate-forge clean install-deps

# ============================================================================
# RUST BUILD PATH
# ============================================================================

# Default target
all: proto-gen build-cortex

# Install required dependencies
install-deps:
	@echo "Installing Rust toolchain and dependencies..."
	rustup target add wasm32-unknown-unknown
	cargo install wasm-pack
	@echo "Dependencies installed successfully"

# Generate ProtoBuf code for all languages
proto-gen:
	@echo "Generating ProtoBuf code..."
	cd rust/shared-types && cargo build
	@echo "Rust ProtoBuf code generated"
	# TODO: Add Go and TypeScript generation when needed

# Build stem.wasm - the compiled SLM (separate WASM module)
# stem.wasm: RESERVED for pure SLM inference, NO error handling
build-stem-wasm:
	@echo "Building stem.wasm (SLM runtime)..."
	@echo "  - Pure inference execution ONLY"
	@echo "  - NO error handling"
	cd rust/stem-runtime && cargo build --target wasm32-unknown-unknown --release
	@mkdir -p dist
	cp rust/stem-runtime/target/wasm32-unknown-unknown/release/inner_runtime.wasm dist/stem.wasm
	@echo "✅ stem.wasm built successfully"

# Build stem runtime (native)
build-stem-runtime:
	@echo "Building stem runtime (native)..."
	cd rust/stem-runtime && cargo build --release
	@echo "Stem runtime built successfully"

# Build the core cortex WASM module (raw WASM)
# cortex.wasm: Orchestrator that loads stem.wasm and handles ALL errors
build-cortex: proto-gen build-stem-runtime
	@echo "Building cortex.wasm (Orchestrator)..."
	@echo "  - Loads and orchestrates stem.wasm"
	@echo "  - Handles ALL errors via HEART"
	@echo "  - Manages LoRA adapters"
	cd rust/cortex-wasm && cargo build --target wasm32-unknown-unknown --release
	@mkdir -p dist
	cp rust/cortex-wasm/target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm dist/cortex.wasm
	@echo "✅ cortex.wasm built successfully"

# Build model forge pipeline
build-forge: proto-gen
	@echo "Building model forge..."
	cd rust/model-forge && cargo build --release
	@echo "Model forge built successfully"

# Run cortex unit tests
test-rust: proto-gen
	@echo "Running Rust tests..."
	cd rust/shared-types && cargo test
	cd rust/cortex-wasm && cargo test
	cd rust/tests && cargo test
	@echo "All Rust tests completed successfully!"

# Validate forge with golden tests
validate-forge: build-forge
	@echo "Running forge validation tests..."
	cd rust/model-forge && cargo test
	@echo "Forge validation completed"

# Development helpers
dev-cortex:
	@echo "Building cortex in development mode..."
	cd rust/cortex-wasm && cargo build --target wasm32-unknown-unknown

check-rust:
	@echo "Running cargo check on all Rust crates..."
	cd rust/shared-types && cargo check
	cd rust/cortex-wasm && cargo check
	cd rust/model-forge && cargo check || true
	cd rust/stem-runtime && cargo check

fmt-rust:
	@echo "Formatting Rust code..."
	cd rust/shared-types && cargo fmt
	cd rust/cortex-wasm && cargo fmt
	cd rust/model-forge && cargo fmt || true
	cd rust/stem-runtime && cargo fmt

clippy-rust:
	@echo "Running clippy on Rust code..."
	cd rust/shared-types && cargo clippy
	cd rust/cortex-wasm && cargo clippy
	cd rust/model-forge && cargo clippy || true
	cd rust/stem-runtime && cargo clippy

# ============================================================================
# TYPESCRIPT BUILD PATH
# ============================================================================

# Build cortex with wasm-pack for TypeScript/JavaScript integration
build-typescript: proto-gen build-stem-runtime
	@echo "Building cortex for TypeScript (web target)..."
	cd rust/cortex-wasm && wasm-pack build --target web --out-dir ../../typescript/pkg
	@echo "✅ TypeScript bindings built: typescript/pkg/"

build-typescript-nodejs: proto-gen build-stem-runtime
	@echo "Building cortex for Node.js..."
	cd rust/cortex-wasm && wasm-pack build --target nodejs --out-dir ../../typescript/pkg-nodejs
	@echo "Node.js package built: typescript/pkg-nodejs/"

build-typescript-bundler: proto-gen build-stem-runtime
	@echo "Building cortex for bundlers..."
	cd rust/cortex-wasm && wasm-pack build --target bundler --out-dir ../../typescript/pkg-bundler
	@echo "Bundler package built: typescript/pkg-bundler/"

# Build all TypeScript targets
build-typescript-all: build-typescript build-typescript-nodejs build-typescript-bundler
	@echo "✅ All TypeScript builds completed successfully!"

# Run TypeScript tests
test-typescript:
	@echo "Running TypeScript tests..."
	cd typescript/tests && node test_cortex.js
	cd typescript/tests && node test_cortex_comprehensive.js
	@echo "✅ All TypeScript tests completed!"

# TypeScript manager app
dev-manager:
	@echo "Starting TypeScript manager in dev mode..."
	cd typescript/manager && npm run dev

build-manager:
	@echo "Building TypeScript manager..."
	cd typescript/manager && npm run build

# ============================================================================
# COMBINED TARGETS
# ============================================================================

# Build everything (Rust + TypeScript)
build-all: build-cortex build-stem-wasm build-typescript-all
	@echo "✅ All builds completed!"
	@echo ""
	@echo "Built artifacts:"
	@echo "  - dist/cortex.wasm      (Orchestrator)"
	@echo "  - dist/stem.wasm        (SLM Runtime)"
	@echo "  - typescript/pkg/       (TypeScript bindings)"

# Run all tests (Rust + TypeScript)
test-all: test-rust test-typescript
	@echo "✅ All tests completed!"

# Format all code
fmt: fmt-rust
	@echo "✅ All code formatted!"

# Lint all code
clippy: clippy-rust
	@echo "✅ All code linted!"

# Clean all build artifacts
clean:
	@echo "Cleaning build artifacts..."
	cd rust/shared-types && cargo clean
	cd rust/cortex-wasm && cargo clean
	cd rust/stem-runtime && cargo clean
	cd rust/model-forge && cargo clean || true
	rm -rf dist/
	rm -rf typescript/pkg/ typescript/pkg-nodejs/ typescript/pkg-bundler/
	@echo "✅ Clean completed"

# ============================================================================
# HELP
# ============================================================================

help:
	@echo "KNIRV-CORTEX Build System"
	@echo "========================="
	@echo ""
	@echo "ARCHITECTURE:"
	@echo "  rust/         - Rust build path (cortex.wasm, stem.wasm)"
	@echo "  typescript/   - TypeScript build path (bindings, manager, tests)"
	@echo ""
	@echo "RUST TARGETS:"
	@echo "  build-cortex         - Build cortex.wasm (orchestrator)"
	@echo "  build-stem-wasm      - Build stem.wasm (SLM runtime)"
	@echo "  build-forge          - Build model forge pipeline"
	@echo "  test-rust            - Run Rust tests"
	@echo "  fmt-rust             - Format Rust code"
	@echo "  clippy-rust          - Lint Rust code"
	@echo ""
	@echo "TYPESCRIPT TARGETS:"
	@echo "  build-typescript     - Build TypeScript bindings (web)"
	@echo "  build-typescript-all - Build all TypeScript targets"
	@echo "  test-typescript      - Run TypeScript tests"
	@echo "  dev-manager          - Start manager in dev mode"
	@echo "  build-manager        - Build manager for production"
	@echo ""
	@echo "COMBINED TARGETS:"
	@echo "  build-all            - Build everything (Rust + TypeScript)"
	@echo "  test-all             - Run all tests"
	@echo "  clean                - Clean all build artifacts"
	@echo ""
	@echo "ALIASES (for compatibility):"
	@echo "  test-cortex          - Alias for test-rust"
	@echo "  check                - Alias for check-rust"

# Aliases for backwards compatibility
test-cortex: test-rust
check: check-rust
