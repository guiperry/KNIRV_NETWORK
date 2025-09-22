# KNIRVCORTEX User Guide

## Overview

KNIRVCORTEX is a WebAssembly-based cognitive processing engine designed for deterministic execution of cognitive tasks. It bundles an orchestrator and model runtime into a single WASM module, providing a standardized ProtoBuf interface for seamless communication.

## Getting Started

### Installation

1. **Prerequisites:** You'll need Rust 1.70+ with the `wasm32-unknown-unknown` target, the Protocol Buffers compiler (`protoc`), and Make.
2. **Build:** Clone the KNIRVCORTEX repository and navigate to the project directory.  Then, use the following commands to build the core components:

   ```bash
   make build-cortex  # Builds the entire system
   ```

   This will create the necessary WASM module and other artifacts in the `dist/` directory.  You can also build individual components using `make proto-gen`, `make build-forge`, and `make build-inner-runtime` if needed.

### Running KNIRVCORTEX

KNIRVCORTEX interacts primarily through its ProtoBuf API.  The core functions include:

* `initialize`: Initializes CORTEX with configuration data.
* `load_weights`: Loads model weights into the engine.
* `run_cognitive_task`: Executes a cognitive task using provided input.
* `set_context` and `set_tools`: Set the context and tools for the task.
* `get_model_info` and `get_weights_info`: Retrieve information about the loaded model and weights.

The ProtoBuf message definitions (`InferenceInput`, `InferenceOutput`, `Envelope`) specify the data format for input and output.  Refer to the API Reference section for detailed information.

### Model Management

The `model-forge` pipeline processes and prepares models for use with KNIRVCORTEX.  This involves several steps: discovery, normalization, runtime binding, compilation, validation, and packaging.  You can use the following command to run the model forge:

```bash
./model-forge/target/release/forge --input models/ --output processed/
```

Replace `models/` and `processed/` with your input and output directories.

## Troubleshooting

### Common Issues

* **Error Codes:**  KNIRVCORTEX returns error codes to indicate issues.  See the "Error Codes" section for a list of codes and their descriptions.
* **Memory Limits:**  KNIRVCORTEX uses dynamic memory allocation. If you encounter `MEMORY_LIMIT_EXCEEDED` errors, adjust the memory limits in your configuration.
* **Model Loading:** Ensure that the model weights are correctly loaded using the `load_weights` function.  Check the `get_weights_info` function for confirmation.
* **External AI Integration (Beta):** The beta integration with external AI providers (Gemini, Claude, OpenAI, Deepseek) might experience occasional instability.

### Advanced Troubleshooting

* **Debugging:** Use the `--debug` flag when running KNIRVCORTEX to enable debug logging.
* **Memory Profiling:** Use tools like `valgrind` or `gprof` to profile memory usage and identify performance bottlenecks.

## API Reference

The core functions are accessible via a ProtoBuf API. Key messages include `InferenceInput`, `InferenceOutput`, and `Envelope`.  See the original README for detailed ProtoBuf definitions.

## Integration with other KNIRV Network Components

KNIRVCORTEX is designed to integrate with other KNIRV Network projects, including KNIRVCONTROLLER (web interface), KNIRVCHAIN (blockchain integration), and KNIRVGRAPH (error tracing).

## Performance

* **WASM Size:** 290KB
* **Memory Usage:** Dynamic, configurable limits.
* **Inference Time:** Varies based on model complexity.
* **Startup Time:** < 10ms

## Error Codes

| Code | Constant       | Description                 |
|------|-----------------|-----------------------------|
| 1000 | INVALID_INPUT   | Input validation failed      |
| 1001 | PROCESSING_FAILED | Task processing error        |
| 1002 | MEMORY_LIMIT_EXCEEDED | Memory allocation failed   |
| 1003 | TIMEOUT         | Operation timed out          |
| 1004 | MODEL_NOT_LOADED | No model weights loaded     |
| 1005 | UNSUPPORTED_OPERATION | Operation not supported    |
| 1006 | RUNTIME_ERROR   | Inner runtime error         |

IMPROVEMENTS:

1. **Clearer headings and sectioning:** Use clear headings and sectioning to make the guide easier to navigate.
2. **More detailed explanations:** Provide more detailed explanations of technical concepts, such as the ProtoBuf API and WebAssembly.
3. **Example code snippets:** Include example code snippets to illustrate how to use KNIRVCORTEX.
4. **Troubleshooting section:** Add a troubleshooting section to provide more detailed guidance on resolving common issues.
5. **Performance metrics:** Include more detailed performance metrics, such as inference time and memory usage.

<div class="footer-links">
<a href="documentation/static/legal/CODE_OF_CONDUCT.html" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="documentation/static/legal/PRIVACY_POLICY.html" class="footer-link">PRIVACY_POLICY.md</a> | <a href="documentation/static/legal/TERMS_AND_CONDITIONS.html" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
