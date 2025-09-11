# KNIRV-CORTEX

**Deterministic Cognitive Shell with ProtoBuf ABI**

KNIRV-CORTEX is a WebAssembly-based cognitive processing engine that bundles an orchestrator and model runtime into a single deterministic WASM artifact. It provides a standardized ProtoBuf-based ABI for cognitive task execution with embedded memory management and tool integration.

## 🏗️ Architecture

KNIRV-CORTEX follows a single-module WASM design where the orchestrator and inner runtime are statically linked together, eliminating cross-module imports and ensuring deterministic execution.

### Core Components

- **🧠 Cognitive Shell**: Main orchestrator handling task routing and memory management
- **⚙️ Inner Runtime**: Embedded ML inference engine with model weight management
- **📡 ProtoBuf ABI**: Standardized binary interface for all communications
- **🔧 Model Forge**: 6-phase pipeline for model discovery, normalization, and packaging
- **🧪 Golden Tests**: Comprehensive ABI conformance and integration testing

### Memory Protocol

All data exchange uses a packed pointer/length format:
- **High 32 bits**: Memory pointer (WASM linear memory offset)
- **Low 32 bits**: Data length in bytes
- **Serialization**: Protocol Buffers for all structured data

## 🚀 Quick Start

### Prerequisites

- Rust 1.70+ with `wasm32-unknown-unknown` target
- Protocol Buffers compiler (`protoc`)
- Make

### Building

```bash
# Build the complete CORTEX system
make build-cortex

# Build individual components
make proto-gen        # Generate ProtoBuf code
make build-forge      # Build model forge pipeline
make build-inner-runtime  # Build inner runtime

# Run tests
make test-cortex      # Run all tests including golden tests
```

### Generated Artifacts

- `dist/cortex.wasm` - Main CORTEX WASM module (290KB)
- `model-forge/target/release/forge` - Model processing pipeline
- `inner-runtime/target/release/` - Standalone inner runtime

## 📋 API Reference

### Core Functions

```rust
// Initialize CORTEX with configuration
initialize(config_ptr: *const u8, config_len: usize) -> bool

// Load model weights
load_weights(weights_ptr: *const u8, weights_len: usize) -> bool

// Execute cognitive task
run_cognitive_task(input_ptr: *const u8, input_len: usize) -> u64

// Set context and tools
set_context(context_ptr: *const u8, context_len: usize) -> bool
set_tools(tools_ptr: *const u8, tools_len: usize) -> bool

// Introspection
get_model_info() -> String
get_weights_info() -> String
```

### ProtoBuf Messages

#### InferenceInput
```protobuf
message InferenceInput {
  string prompt = 1;
  string context = 2;
  optional Config config = 3;
  optional MemoryPolicy memory_policy = 4;
}
```

#### InferenceOutput
```protobuf
message InferenceOutput {
  string response = 1;
  float confidence = 2;
  float processing_time_ms = 3;
  repeated string debug_info = 4;
}
```

#### Envelope (Response Wrapper)
```protobuf
message Envelope {
  EnvelopeKind kind = 1;
  bytes payload = 2;
  uint64 timestamp = 3;
  optional string trace_id = 4;
}
```

## 🔧 Model Forge Pipeline

The Model Forge implements a 6-phase processing pipeline:

1. **Discovery**: Scan and identify model files
2. **Normalization**: Convert to safetensors format
3. **Runtime Binding**: Generate runtime-specific bindings
4. **Compilation**: Optimize for target runtime
5. **Validation**: Verify model integrity and performance
6. **Packaging**: Create deployment-ready artifacts

```bash
# Run model forge
./model-forge/target/release/forge --help
./model-forge/target/release/forge --input models/ --output processed/
```

## 🧪 Testing

### Golden Tests

Comprehensive ABI conformance tests covering:

- ProtoBuf encoding/decoding
- Memory protocol validation
- Error handling
- Large payload processing
- Tool integration
- Context management

```bash
# Run all tests
make test-cortex

# Run specific test suites
cd tests && cargo test
cd shared-types && cargo test
cd rust-wasm && cargo test
```

### Test Coverage

- ✅ 11 golden tests passing
- ✅ ProtoBuf ABI conformance
- ✅ Memory protocol validation
- ✅ Error code verification
- ✅ Large payload handling (10KB+)

## 📊 Performance

- **WASM Size**: 290KB (optimized release build)
- **Memory Usage**: Dynamic allocation with configurable limits
- **Inference Time**: Varies by model size and complexity
- **Startup Time**: < 10ms for initialization

## 🔍 Error Codes

| Code | Constant | Description |
|------|----------|-------------|
| 1000 | INVALID_INPUT | Input validation failed |
| 1001 | PROCESSING_FAILED | Task processing error |
| 1002 | MEMORY_LIMIT_EXCEEDED | Memory allocation failed |
| 1003 | TIMEOUT | Operation timed out |
| 1004 | MODEL_NOT_LOADED | No model weights loaded |
| 1005 | UNSUPPORTED_OPERATION | Operation not supported |
| 1006 | RUNTIME_ERROR | Inner runtime error |

## 🗂️ Project Structure

```
KNIRVCORTEX/
├── shared-types/          # ProtoBuf definitions and ABI helpers
├── rust-wasm/            # Main CORTEX WASM module
├── inner-runtime/        # Embedded ML inference runtime
├── model-forge/          # Model processing pipeline
├── tests/               # Golden tests and ABI conformance
├── dist/               # Built artifacts
└── docs/               # Architecture and implementation docs
```

## 🎯 Implementation Status

- ✅ **Week 1**: ProtoBuf ABI foundation
- ✅ **Week 2**: WASM module with ProtoBuf integration
- ✅ **Week 3**: Model Forge 6-phase pipeline
- ✅ **Week 4**: Inner Runtime integration
- ✅ **Week 5**: Golden tests and ABI conformance
- 🔄 **Week 6**: Integration demos (KNIRVCONTROLLER/KNIRVENGINE)
- 📋 **Week 7**: Agentic Memory integration
- 📋 **Week 8**: Performance optimization and documentation

## 🤝 Integration

KNIRV-CORTEX is designed to integrate with:

- **KNIRVCONTROLLER** (TypeScript): Web-based cognitive task management
- **KNIRVENGINE** (Go): Server-side cognitive processing
- **KNIRVCHAIN**: Blockchain-based skill execution
- **KNIRVGRAPH**: Error cluster tracing and skill discovery

## 📄 License

Part of the KNIRV Network ecosystem. See project root for licensing information.

## 🔗 Related Projects

- [KNIRV_NETWORK](../): Main project repository
- [KNIRVCONTROLLER](../KNIRVCONTROLLER/): Web interface
- [KNIRVENGINE](../KNIRVENGINE/): Server runtime
- [KNIRVCHAIN](../KNIRVCHAIN/): Blockchain integration
