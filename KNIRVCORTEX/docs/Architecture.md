Architecture: The Cognitive Cortex
The core idea remains the same: a two-part WASM system.

Cognitive Shell (Outer WASM): A generic container that provides the core model logic, lifecycle, and communication capabilities. This is a standard component of our system.
Forged Model (Inner WASM): A specific SLM/LLM that has been compiled to WASM. It exposes a standardized inference API. This is the component that users will upload.
The "forging" process happens within a new model-forge crate, and the entire build is localized to the KNIRVCORTEX directory.

Proposed Directory Structure within KNIRVCORTEX
This structure keeps all model-related Rust code and build artifacts neatly organized.

plaintext
KNIRVCORTEX/
├── model-forge/         # (New) The "forge" for compiling models into inner WASM.
│   ├── Cargo.toml
│   ├── README.md
│   └── src/
│       ├── model.onnx   # Placeholder for a model file.
│       └── lib.rs
├── cognitive-shell/     # (New) The "outer" WASM cognitive shell.
│   ├── Cargo.toml
│   └── src/
│       └── lib.rs
├── shared-types/        # (New) Shared Rust data structures.
│   ├── Cargo.toml
│   └── src/
│       └── lib.rs
└── Makefile             # (New) Makefile for building all cortex components.
Step 1: The Forge (model-forge)
This new crate is responsible for "forging" a pre-trained ONNX model into a KNIRV-compatible inner WASM module. It exposes a C-compatible API for initialization, memory management, and inference.

Here are the files for the model-forge:

New file: Cargo.toml
+13
 Show full code block 
[package]
name = "model-forge"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
tract-onnx = "0.21.0"
serde_json = "1.0"
shared-types = { path = "../shared-types" }


New file: README.md

 Show full code block 
# Model Forge

The Model Forge is a Rust crate for compiling pre-trained models (in ONNX format) into KNIRV-compatible WASM modules. The output `.wasm` file is a "Forged Model" that can be uploaded and executed by the KNIRV Cognitive Shell.

## How it Works

This crate uses `tract-onnx` to load and run inference on an `model.onnx` file embedded in the crate. It exposes a standardized C-style API that the WASM host (the Go runtime) can call.

### Exported API

*   `init()`: Initializes the model. Must be called once by the host before inference.
*   `alloc(size: usize) -> *mut u8`: Allocates memory within the WASM module. The host uses this to create buffers for passing data.
*   `dealloc(ptr: *mut u8, size: usize)`: Frees memory allocated by `alloc`.
*   `infer(input_ptr: *const u8, input_len: usize) -> u64`: Runs inference. It takes a pointer to a serialized `InferenceInput` and returns a packed `u64` containing the pointer and length of the serialized `InferenceOutput`.

## Building a Forged Model

1.  Place our quantized `model.onnx` file in the `src/` directory.
2.  Run the build command from the `KNIRVCORTEX` directory:

```bash
make build-forge
```

The final model will be located at `KNIRVCORTEX/dist/forged_model_example.wasm`. This is the file you upload through the KNIRV Controller UI.

New file: Cargo.toml
+7
 Show full code block 
[package]
name = "shared-types"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = { version = "1.0", features = ["derive"] }

New file: lib.rs
+14
 Show full code block 
use serde::{Deserialize, Serialize};

/// Input structure for the model's inference function.
#[derive(Serialize, Deserialize, Debug)]
pub struct InferenceInput {
    pub prompt: String,
    pub context: String,
}

/// Output structure for the model's inference function.
#[derive(Serialize, Deserialize, Debug)]
pub struct InferenceOutput {
    pub response: String,
}

New file: lib.rs
+81
 Show full code block 
use std::sync::Once;
use tract_onnx::prelude::*;
use shared_types::{InferenceInput, InferenceOutput};

// Use a static variable to hold the model, ensuring it's loaded only once.
static mut MODEL: Option<SimplePlan<TypedModel>> = None;
static INIT: Once = Once::new();

/// Initializes the model from the embedded ONNX data.
/// This function should be called by the host before any inference.
#[no_mangle]
pub extern "C" fn init() {
    INIT.call_once(|| {
        // In a real scenario, you would `include_bytes!` our model.
        // For now, we simulate initialization.
        // let model_bytes = include_bytes!("model.onnx");
        // let model = tract_onnx::onnx()
        //     .model_for_read(&mut model_bytes.as_ref())
        //     .unwrap()
        //     .into_runnable()
        //     .unwrap();
        // unsafe { MODEL = Some(model); }
    });
}

/// A more robust way to pass data back to the host.
/// We pack a pointer and a length into a single u64.
/// The high 32 bits are the pointer, the low 32 bits are the length.
fn pack_ptr_len(ptr: *mut u8, len: usize) -> u64 {
    ((ptr as u64) << 32) | (len as u64)
}

/// Runs inference on the loaded model.
///
/// Receives a pointer to a byte slice (JSON serialized `InferenceInput`),
/// and returns a packed u64 (pointer + length) to the byte slice
/// (JSON serialized `InferenceOutput`).
/// The caller is responsible for memory management (dealloc).
#[no_mangle]
pub extern "C" fn infer(input_ptr: *const u8, input_len: usize) -> u64 {
    let input_slice = unsafe { std::slice::from_raw_parts(input_ptr, input_len) };
    let input: InferenceInput = serde_json::from_slice(input_slice)
        .expect("Failed to deserialize input");

    // --- Placeholder for actual model inference ---
    // In a real scenario, you would convert `input.prompt` to tensors,
    // run `MODEL.run()`, and process the output tensors.
    let output_text = format!("Inference result for: '{}'", input.prompt);
    // --- End of placeholder ---

    let output = InferenceOutput { response: output_text };
    let output_bytes = serde_json::to_vec(&output).expect("Failed to serialize output");

    // Allocate memory for the output, copy the data, and return the packed pointer/length.
    let len = output_bytes.len();
    let ptr = alloc(len);
    unsafe {
        std::ptr::copy_nonoverlapping(output_bytes.as_ptr(), ptr, len);
    }

    pack_ptr_len(ptr, len)
}

/// Allocates memory in the WASM module's linear memory.
/// The host calls this to get a buffer for passing data.
#[no_mangle]
pub extern "C" fn alloc(size: usize) -> *mut u8 {
    let mut buf = Vec::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf); // Prevent Rust from freeing the memory
    ptr
}

/// Frees memory allocated by `alloc` or returned from `infer`.
/// The host calls this after it's done with the memory.
#[no_mangle]
pub extern "C" fn dealloc(ptr: *mut u8, size: usize) {
    unsafe {
        let _ = Vec::from_raw_parts(ptr, 0, size);
    }
}

Step 2: The Cognitive Shell (cognitive-shell)
The shell is the "outer" WASM. It defines the model's behavior and imports the infer function, which will be provided at runtime by the host linking it to the "inner" WASM.

New file: Cargo.toml
+11
 Show full code block 
[package]
name = "cognitive-shell"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
serde_json = "1.0"
shared-types = { path = "../shared-types" }

New file: lib.rs
+40
 Show full code block 
use shared_types::{InferenceInput, InferenceOutput};

// This block declares the functions that will be imported from the "inner" WASM model.
// The host environment (Go with Wasmtime) is responsible for linking these at runtime.
#[link(wasm_import_module = "env")]
extern "C" {
    fn infer(input_ptr: *const u8, input_len: usize) -> u64;
    // We might also need to import alloc/dealloc if the shell needs to manage the model's memory directly.
}

/// Unpacks a u64 into a pointer and a length.
fn unpack_ptr_len(packed: u64) -> (*mut u8, usize) {
    let ptr = (packed >> 32) as *mut u8;
    let len = (packed & 0xFFFFFFFF) as usize;
    (ptr, len)
}

/// A function within the cognitive shell that uses the specialized model.
/// This function could be exported for the host to call.
#[no_mangle]
pub extern "C" fn run_cognitive_task(prompt_ptr: *const u8, prompt_len: usize) -> u64 {
    let prompt_slice = unsafe { std::slice::from_raw_parts(prompt_ptr, prompt_len) };
    let prompt = std::str::from_utf8(prompt_slice).unwrap();

    let input = InferenceInput {
        prompt: prompt.to_string(),
        context: "from_cognitive_shell".to_string(),
    };
    let input_bytes = serde_json::to_vec(&input).unwrap();

    // Call the imported `infer` function from the inner WASM model.
    let result_packed = unsafe {
        infer(input_bytes.as_ptr(), input_bytes.len())
    };

    // The shell can inspect the result, but for now, we just pass it through.
    // The host, which called `run_cognitive_task`, will be responsible for
    // reading and deallocating the memory from the inner WASM.
    result_packed
}

Step 3: Build Orchestration
A new Makefile inside KNIRVCORTEX will handle building the Rust crates.

New file: Makefile
+42
 Show full code block 
# Makefile for KNIRVCORTEX components

TARGET := wasm32-unknown-unknown
DIST_DIR := ./dist

.PHONY: all
all: setup build ## Build all components

.PHONY: setup
setup:
	@echo "Creating distribution directory..."
	@mkdir -p $(DIST_DIR)

.PHONY: build
build: build-shell build-forge

.PHONY: build-shell
build-shell:
	@echo "Building Cognitive Shell..."
	@cd cognitive-shell && cargo build --target $(TARGET) --release
	@cp cognitive-shell/target/$(TARGET)/release/cognitive_shell.wasm $(DIST_DIR)/cognitive_shell.wasm
	@echo "✓ Cognitive Shell built at $(DIST_DIR)/cognitive_shell.wasm"

.PHONY: build-forge
build-forge:
	@echo "Building Model Forge..."
	@cd model-forge && cargo build --target $(TARGET) --release
	@cp model-forge/target/$(TARGET)/release/model_forge.wasm $(DIST_DIR)/forged_model_example.wasm
	@echo "✓ Model Forge built at $(DIST_DIR)/forged_model_example.wasm"

.PHONY: test
test: ## Run tests for all cortex crates
	@echo "Testing KNIRVCORTEX components..."
	@cd shared-types && cargo test
	@cd cognitive-shell && cargo test
	@cd model-forge && cargo test

.PHONY: clean
clean: ## Clean all build artifacts
	@cd cognitive-shell && cargo clean
	@cd model-forge && cargo clean
	@rm -rf $(DIST_DIR)

And we can update the root Makefile.mk to delegate to this new file.

Makefile.mk
-3
+3
 Show full code block 
.PHONY: test-cortex
test-cortex: ## Test KNIRVCORTEX (AI Agent Framework)
	@echo "$(BLUE)Testing KNIRVCORTEX...$(NC)"
	@if [ -f "KNIRVCORTEX/scripts/run-tests.sh" ]; then \
		cd KNIRVCORTEX && ./scripts/run-tests.sh; \
		echo "$(GREEN)✓ KNIRVCORTEX tests completed$(NC)"; \
	@if [ -f "KNIRVCORTEX/Makefile" ]; then \
		cd KNIRVCORTEX && $(MAKE) test; \
		echo "$(GREEN)✓ KNIRVCORTEX tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVCORTEX test script not found$(NC)"; \
	fi

Next Steps: Integration with KNIRVCONTROLLER
With the build process defined, here is the clear path to full integration:

Build: A developer runs make inside the KNIRVCORTEX directory. This produces dist/cognitive_shell.wasm (which is deployed with our host application) and dist/forged_model_example.wasm (which is the distributable agent model).

Upload: In the KNIRVCONTROLLER UI, a user navigates to the AgentManager. The "Upload WASM Agent" button, powered by the handleAgentUpload function in AgentManager.tsx, sends the forged_model_example.wasm file to our backend.

Host Orchestration (Go Backend): Your Go application, acting as the WASM host, performs these steps at runtime using a library like wasmtime-go:

It loads both cognitive_shell.wasm and the user-uploaded forged_model.wasm from storage.
It instantiates the forged_model.wasm first.
It creates a Linker and finds the exported init, infer, alloc, and dealloc functions from the forged model instance.
It uses the Linker to define these functions, satisfying the imports required by cognitive_shell.wasm.
Finally, it instantiates cognitive_shell.wasm using this configured linker.
Execution: The Go host can now call exported functions on the cognitive_shell instance (like run_cognitive_task). When the shell's code calls infer, the host transparently routes the call to the infer function inside the forged_model instance, managing the memory and data transfer between the two isolated WASM modules.

This revised plan provides a robust, self-contained architecture for building and deploying custom AI models within the KNIRV ecosystem.