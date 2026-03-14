# KNIRV-CORTEX

**Deterministic Cognitive Shell with ProtoBuf ABI**

KNIRV-CORTEX is a WebAssembly-based cognitive processing engine that bundles an orchestrator and model runtime into a single deterministic WASM artifact. It provides a standardized ProtoBuf-based ABI for cognitive task execution with embedded memory management and tool integration.

## � Recent Updates

### ✅ External AI Integration (Beta Phase)
- **rust-wasm**: Added external inference module with support for Gemini, Claude, OpenAI, and Deepseek
- **model-forge**: Enhanced compilation pipeline with external API integration and validation
- **Cognitive Shell**: Updated orchestrator to route inference through external providers during beta
- **WASM Compatibility**: Maintained deterministic processing while enabling external fallback options

### ✅ HEART Integration (Error Analysis System)
- **HEART Client**: Integrated heuristic error analysis transformer for intelligent error handling
- **cortex.wasm**: Orchestrator queries HEART when stem.wasm errors occur for pattern recognition and recommendations
- **Adaptive Learning**: Error inquiries cached and analyzed for continuous improvement
- **Fallback Heuristics**: Local rule-based analysis when HEART service is unavailable
- **ProtoBuf Messages**: New `heart.proto` definitions for error inquiries and heuristic responses

## 🧠 HEART Integration

KNIRV-CORTEX now integrates with the **HEART (Heuristic Error Analysis and Recognition Transformer)** system to provide intelligent error analysis and recovery recommendations. When cortex.wasm receives errors from stem.wasm during inference, it queries HEART for heuristic analysis.

### Architecture

**CRITICAL: Separation of Concerns**

- **stem.wasm**: RESERVED FOR COMPILED SLM (Small Language Model)
  - Pure inference execution
  - Model weight loading
  - NO error handling or recovery
  - NO HEART integration

- **cortex.wasm**: Cognitive Shell Orchestrator
  - Loads and orchestrates stem.wasm
  - Loads and applies LoRA adapters
  - Handles ALL errors from stem.wasm
  - HEART integration for error analysis
  - Memory policy management
  - External inference fallbacks

```
cortex.wasm (Cognitive Shell Orchestrator)
  ├─> Load stem.wasm (Compiled SLM)
  │    └─> Pure Inference Only, NO Error Handling
  │
  ├─> Load LoRA Adapters
  │    └─> Apply to stem.wasm
  │
  └─> Error Detection & Handling
       └─> HEART Client (heart_client.rs)
            └─> HTTP POST /heart/analyze
                 └─> HEART Service (heart_service.go)
                      └─> Cerebras WSE Transformer
                           └─> Heuristic Command Vector
```

### Key Features

- **Automatic Error Analysis**: cortex.wasm automatically queries HEART when errors from stem.wasm occur
- **Pattern Recognition**: HEART identifies error patterns and similar historical errors
- **Recommended Actions**: Receives concrete debugging steps and recovery strategies
- **Confidence Scoring**: Only applies heuristics above configurable confidence threshold (default: 0.7)
- **Local Fallback**: Rule-based heuristics when HEART service is unavailable or slow
- **Caching**: Recent heuristic responses cached to reduce latency
- **Rate Limiting**: Max 1 HEART query per second to prevent overload

### Configuration

```rust
HEARTConfig {
    endpoint: "http://localhost:8080/heart/analyze",
    timeout_ms: 5000,
    min_confidence_threshold: 0.7,
    enable_pattern_recognition: true,
    enable_similarity_search: true,
    fallback_to_local: true,
}
```

### Error Inquiry Format

When cortex.wasm encounters an error from stem.wasm, it constructs a `HEARTErrorInquiry`:

```protobuf
message HEARTErrorInquiry {
  string error_id = 1;
  string error_type = 2;           // e.g., "TypeError", "InferenceError"
  string error_message = 3;
  string error_context = 4;
  string stack_trace = 5;
  map<string, string> metadata = 6;
  string prompt = 8;              // Input prompt when error occurred
  string model_response = 9;      // Partial response before error
  float confidence_score = 10;
  uint64 timestamp = 11;
}
```

### Heuristic Response

HEART returns a comprehensive analysis:

```protobuf
message HEARTHeuristicResponse {
  string inquiry_id = 1;
  repeated float command_vector = 2;     // 8-element heuristic vector
  uint32 alert_level = 3;                // 1-5 severity
  uint32 heuristic_id = 4;               // Analysis method used
  float confidence_score = 6;            // 0-1 confidence
  string analysis_summary = 8;           // Human-readable analysis
  repeated string recommended_actions = 9;
  repeated string debug_insights = 10;
  repeated ErrorPattern identified_patterns = 11;
  repeated SimilarError similar_errors = 12;
  float processing_time_ms = 13;
}
```

### Usage Example

```rust
// HEART integration happens automatically in cortex.wasm
let input = InferenceInput {
    prompt: "Analyze network traffic patterns".to_string(),
    context: "Real-time monitoring".to_string(),
    config: Some(config),
    memory_policy: None,
};

// If stem.wasm returns an error during inference, cortex.wasm will:
// 1. Detect the error from stem.wasm
// 2. Query HEART for heuristic analysis
// 3. Apply recommended mitigations
// 4. Cache the heuristic response
// 5. Return enhanced error information

let result = cortex.run_cognitive_task(input).await;
```

### Local Fallback Heuristics

When HEART is unavailable, cortex.wasm uses rule-based fallbacks:

| Error Type | Alert Level | Recommended Actions |
|------------|-------------|---------------------|
| TypeError, ReferenceError | 3 | Review variable declarations, check for undefined references |
| NetworkError, FetchError | 4 | Verify endpoint availability, check CORS configuration |
| InferenceError, ModelError | 5 | Verify model is loaded, validate input shape and format |
| Generic | 2 | Review error message, check stack trace |

### Statistics and Monitoring

HEART client tracks performance metrics:

```json
{
  "total_inquiries": 142,
  "successful_responses": 138,
  "failed_responses": 4,
  "avg_response_time_ms": 234.5,
  "success_rate": 0.972
}
```

### Integration with KNIRV Network

HEART integration enables:

- **Skill Discovery**: Error patterns mapped to LoRA adapters in KNIRVCHAIN
- **Knowledge Graphs**: Error clusters traced in KNIRVGRAPH
- **Network Intelligence**: Collective learning from distributed errors
- **Autonomous Recovery**: cortex.wasm orchestrates self-healing using HEART recommendations

## 🏗️ Architecture

KNIRV-CORTEX follows a **dual-module WASM design** for proper separation of concerns:

### Core Components

- **🧠 cortex.wasm (Cognitive Shell)**: Main orchestrator handling:
  - Loading and orchestrating stem.wasm
  - Task routing and memory management
  - LoRA adapter loading and application
  - ALL error handling via HEART integration
  - External inference fallbacks

- **⚙️ stem.wasm (SLM Runtime)**: RESERVED for compiled Small Language Model:
  - Pure ML inference execution
  - Model weight management
  - NO error handling or recovery
  - Called by cortex.wasm only

- **📡 ProtoBuf ABI**: Standardized binary interface for all communications
- **🔧 Model Forge**: 6-phase pipeline for model discovery, normalization, and packaging
- **🧪 Golden Tests**: Comprehensive ABI conformance and integration testing

### Current State

**Note**: Currently, stem.wasm is statically linked into cortex.wasm for simplicity. Future versions will dynamically load stem.wasm as a separate WASM module using WebAssembly.instantiate().

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
# Build Rust components
make build-cortex        # Build cortex.wasm (orchestrator)
make build-stem-wasm     # Build stem.wasm (SLM runtime)
make build-forge         # Build model forge pipeline

# Build TypeScript bindings
make build-typescript    # Build TypeScript bindings (web)
make build-typescript-all # Build all TypeScript targets

# Build everything
make build-all           # Build Rust + TypeScript

# Run tests
make test-rust           # Run Rust tests
make test-typescript     # Run TypeScript tests
make test-all            # Run all tests

# Development
make dev-cortex          # Build cortex in dev mode
make dev-manager         # Start TypeScript manager dev server
```

### Generated Artifacts

- `dist/cortex.wasm` - Main CORTEX orchestrator module (~290KB)
- `dist/stem.wasm` - Compiled SLM runtime module (reserved for pure inference)
- `model-forge/target/release/forge` - Model processing pipeline
- `inner-runtime/target/release/` - Standalone inner runtime (native build)

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
# Run all tests (Rust + TypeScript)
make test-all

# Run Rust tests only
make test-rust

# Run TypeScript tests only
make test-typescript

# Run specific Rust test suites
cd rust/tests && cargo test
cd rust/shared-types && cargo test
cd rust/cortex-wasm && cargo test

# Run specific TypeScript tests
cd typescript/tests && node test_cortex.js
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

**Reorganized with clear separation of Rust and TypeScript build paths:**

```
KNIRVCORTEX/
├── rust/                      # 🦀 RUST BUILD PATH
│   ├── cortex-wasm/          #    Main CORTEX orchestrator WASM module
│   ├── stem-runtime/         #    SLM inference runtime (stem.wasm)
│   ├── shared-types/         #    ProtoBuf definitions and ABI helpers
│   ├── model-forge/          #    Model processing pipeline
│   ├── lora-engine/          #    LoRA adapter compilation
│   ├── hrm/                  #    Heuristic runtime modules
│   └── tests/                #    Rust golden tests and ABI conformance
│
├── typescript/                # 📘 TYPESCRIPT BUILD PATH
│   ├── wrapper/              #    TypeScript wrapper (CortexWrapper.ts)
│   ├── pkg/                  #    Auto-generated WASM bindings
│   ├── manager/              #    TypeScript manager application
│   ├── examples/             #    TypeScript usage examples
│   └── tests/                #    TypeScript/JavaScript tests
│
├── dist/                      # 📦 BUILD ARTIFACTS
│   ├── cortex.wasm           #    Orchestrator (handles errors, loads LoRAs)
│   └── stem.wasm             #    SLM runtime (pure inference only)
│
├── Makefile                   # Build system (rust + typescript targets)
└── README.md                  # This file
```

### Build Path Separation

This structure makes it clear that KNIRV-CORTEX has **two separate build paths**:

1. **Rust Build Path** (`rust/`): Builds the WASM modules
   - `cortex.wasm` - Orchestrator
   - `stem.wasm` - SLM runtime

2. **TypeScript Build Path** (`typescript/`): TypeScript bindings and applications
   - Auto-generated from Rust via `wasm-pack`
   - Wrapper classes and examples
   - Manager application

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
- **KNIRVCHAIN**: Blockchain-based skill execution
- **KNIRVGRAPH**: Error cluster tracing and skill discovery

## 📄 License

Part of the KNIRV Network ecosystem. See project root for licensing information.

## 🔗 Related Projects

- [KNIRV_NETWORK](../): Main project repository
- [KNIRVCONTROLLER](../KNIRVCONTROLLER/): Web interface
- [KNIRVCHAIN](../KNIRVCHAIN/): Blockchain integration
