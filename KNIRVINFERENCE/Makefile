# KNIRV-CORTEX Build System
# Implements the Architecture_ImplementationPlan.md build targets

.PHONY: all proto-gen build-cortex build-forge test-cortex validate-forge clean install-deps

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
	cd shared-types && cargo build
	@echo "Rust ProtoBuf code generated"
	# TODO: Add Go and TypeScript generation when needed

# Build the core cortex WASM module
build-cortex: proto-gen build-inner-runtime
	@echo "Building cortex.wasm..."
	cd rust-wasm && cargo build --target wasm32-unknown-unknown --release
	@mkdir -p dist
	cp rust-wasm/target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm dist/cortex.wasm
	@echo "cortex.wasm built successfully"

build-inner-runtime:
	@echo "Building inner runtime..."
	cd inner-runtime && cargo build --release
	@echo "Inner runtime built successfully"

# Build model forge pipeline
build-forge: proto-gen
	@echo "Building model forge..."
	cd model-forge && cargo build --release
	@echo "Model forge built successfully"

# Run cortex unit tests
test-cortex: proto-gen
	@echo "Running cortex tests..."
	cd shared-types && cargo test
	cd rust-wasm && cargo test
	cd tests && cargo test
	@echo "All cortex tests completed successfully!"

# Validate forge with golden tests
validate-forge: build-forge
	@echo "Running forge validation tests..."
	cd model-forge && cargo test
	@echo "Forge validation completed"

# Clean all build artifacts
clean:
	@echo "Cleaning build artifacts..."
	cd shared-types && cargo clean
	cd rust-wasm && cargo clean
	cd model-forge && cargo clean || true
	rm -rf dist/
	@echo "Clean completed"

# Development helpers
dev-cortex:
	@echo "Building cortex in development mode..."
	cd rust-wasm && cargo build --target wasm32-unknown-unknown

check:
	@echo "Running cargo check on all crates..."
	cd shared-types && cargo check
	cd rust-wasm && cargo check
	cd model-forge && cargo check || true

fmt:
	@echo "Formatting code..."
	cd shared-types && cargo fmt
	cd rust-wasm && cargo fmt
	cd model-forge && cargo fmt || true

clippy:
	@echo "Running clippy..."
	cd shared-types && cargo clippy
	cd rust-wasm && cargo clippy
	cd model-forge && cargo clippy || true