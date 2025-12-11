# HEART: Heuristic Error Analysis and Recognition Transformer

## Cerebras WSE Implementation

This directory contains the Cerebras Wafer-Scale Engine (WSE) implementation of the HEART transformer system, designed for ultra-low-latency inference on network metrics and heuristic command generation.

## Overview

HEART is a specialized transformer architecture optimized for the KNIRV network that:
- Processes real-time streams of network metrics from 256 nodes across 64 time slices
- Generates heuristic analysis commands for network management
- Leverages the massive parallelism of Cerebras WSE for inference speeds measured in microseconds
- Implements attention mechanisms distributed across processing elements (PEs)

### Architecture Specifications

**Model Parameters:**
- Sequence Length (T): 64 time slices
- Number of Nodes (N): 256 network nodes
- Embedding Dimension (D_model): 256
- Number of Attention Heads (H): 8
- Number of Transformer Layers (L): 3
- Input Vector Size: 16 (network metrics per measurement)
- Output Vector Size: 8 (control commands)

**Cerebras Configuration:**
- PE Grid: 8x8 (configurable)
- Sequence Per PE: 8 positions
- Embedding Per PE: 32 dimensions
- Total PEs: 64

## Directory Structure

```
cerebras/
├── layout.csl              # CSL layout defining PE grid and routing
├── heart_pe.csl            # Processing element kernel (transformer computation)
├── run_heart.py            # Python runner for SDK integration
├── params.json             # Model and grid configuration
├── Makefile                # Build system
├── README.md               # This file
├── build/                  # Compiled programs (generated)
├── weights/                # Model weights (generated)
└── data/                   # Input/output data (generated)
```

## Prerequisites

### Software Requirements

1. **Cerebras SDK**
   ```bash
   # The SDK should be installed and available in your PATH
   # Verify installation:
   which cslc
   which cs_python
   ```

2. **Python 3.8+** with NumPy
   ```bash
   cs_python -c "import numpy; print(numpy.__version__)"
   ```

3. **Go 1.21+** (for weight export from Gorgonia)
   ```bash
   go version
   ```

4. **jq** (for Makefile info target)
   ```bash
   sudo apt-get install jq  # Ubuntu/Debian
   ```

### Hardware Access

- **Fabric Simulator**: No special hardware required, uses CPU simulation
- **Cerebras WSE**: Requires access to a Cerebras system with CM address

## Quick Start

### 1. Compile the CSL Program

```bash
cd cerebras/
make compile
```

This compiles the CSL source files into an executable program for the WSE fabric simulator.

**Output:** `build/heart_program/`

### 2. Initialize Weights

For testing with random weights:
```bash
make init-weights
```

For production with trained Gorgonia weights:
```bash
make weights
```

### 3. Generate Test Data

```bash
make test-data
```

### 4. Run on Fabric Simulator

```bash
make run-simulator
```

This executes the HEART transformer on the Cerebras fabric simulator with synthetic data.

### 5. Verify Output

```bash
make verify-output
```

## Detailed Usage

### Compilation

**Standard compilation:**
```bash
make compile
```

**Debug compilation:**
```bash
make compile-debug
```

**Custom grid size:**
```bash
make compile GRID_WIDTH=16 GRID_HEIGHT=16
```

### Weight Management

**Export from Gorgonia model:**
```bash
# From parent directory
cd ..
go run . -export-weights cerebras/weights/model_weights.npz
cd cerebras/
```

**Initialize random weights (testing only):**
```bash
make init-weights
```

The weights file (`weights/model_weights.npz`) contains:
- `embedding_weights`: (MEASURE_VEC_SIZE, EMBEDDING_DIM)
- `positional_encoding`: (SEQUENCE_LENGTH, EMBEDDING_DIM)
- `layer_weights`: Attention and FFN weights for all layers
- `output_weights`: (EMBEDDING_DIM, CONTROL_VEC_SIZE)

### Running Inference

**On Fabric Simulator:**
```bash
make run-simulator
```

**On Actual WSE Device:**
```bash
export CEREBRAS_CMADDR="10.10.10.100:9000"  # Your Cerebras system address
make run-device
```

**Python Runner Directly:**
```bash
cs_python run_heart.py \
  --program build/heart_program \
  --weights weights/model_weights.npz \
  --input data/test_input.npz \
  --output data/test_output.npz
```

**With Test Data Generation:**
```bash
cs_python run_heart.py \
  --program build/heart_program \
  --weights weights/model_weights.npz \
  --test \
  --output data/output.npz
```

### Input Data Format

Input data should be an NPZ file containing:

```python
np.savez('input.npz',
    measurements=measurements,  # Shape: (N*T, MEASURE_VEC_SIZE), dtype: float32
    node_ids=node_ids,         # Shape: (N*T,), dtype: uint32
    time_slices=time_slices    # Shape: (N*T,), dtype: uint32
)
```

Where:
- `measurements`: Network metric vectors (traffic, latency, errors, CPU load, etc.)
- `node_ids`: Network node identifiers
- `time_slices`: Time slice indices

### Output Data Format

Output data is an NPZ file containing:

```python
data = np.load('output.npz')
commands = data['heuristic_commands']  # Shape: (N*T, CONTROL_VEC_SIZE), dtype: float32
node_ids = data['node_ids']            # Shape: (N*T,), dtype: uint32
time_slices = data['time_slices']      # Shape: (N*T,), dtype: uint32
```

The `heuristic_commands` vector format:
- `[0]`: Alert Level (1-5)
- `[1]`: Heuristic ID (which analysis method to apply)
- `[2]`: Target Node ID (for actions)
- `[3]`: Confidence Score (0-1)
- `[4-7]`: Action Parameters

## Integration with Go Transformer

The `cerebras_bridge.go` file in the parent directory provides Go integration:

```go
// Initialize bridge
bridge := NewCerebrasBridge("cerebras/build/heart_program",
                           "cerebras/weights/model_weights.npz",
                           true)  // true = use simulator

// Export weights from Gorgonia model
err := bridge.ExportWeightsFromGorgonia(gptModel, "cerebras/weights/model_weights.npz")

// Prepare input from network metrics
processor := NewNetworkMetricsProcessor(64, 256, 16, 8)
input := processor.ProcessRawMetrics(networkMetrics)

// Run inference
output, err := bridge.RunInference(input)

// Apply commands to network
commands := processor.ApplyHeuristicCommands(output)
for _, cmd := range commands {
    applyToNetwork(cmd)
}
```

## Integration with KNIRVCORTEX

HEART now provides an HTTP service that KNIRVCORTEX (the WebAssembly cognitive shell) uses for intelligent error analysis. When the inner runtime (stem.cortex) encounters errors during inference, it queries HEART for heuristic recommendations.

### HEART HTTP Service

The `heart_service.go` file provides a REST API bridge between CORTEX and the HEART transformer:

```go
// Initialize HEART service with Cerebras bridge
bridge := NewCerebrasBridge("cerebras/build/heart_program",
                           "cerebras/weights/model_weights.npz",
                           true)  // true = use simulator

service := NewHEARTService(bridge)

// Start HTTP server
http.HandleFunc("/heart/analyze", service.handleAnalyze)
http.HandleFunc("/heart/health", service.handleHealth)
http.HandleFunc("/heart/stats", service.handleStats)
http.ListenAndServe(":8080", nil)
```

### API Endpoints

#### POST /heart/analyze

Analyzes an error inquiry and returns heuristic recommendations.

**Request:**
```json
{
  "error_id": "err-1234",
  "error_type": "InferenceError",
  "error_message": "Model output confidence below threshold",
  "error_context": "Processing batch 42",
  "stack_trace": "...",
  "metadata": {
    "model_id": "cortex-v1",
    "batch_size": "32"
  },
  "prompt": "Analyze network traffic...",
  "model_response": "Traffic patterns show...",
  "confidence_score": 0.45,
  "timestamp": 1699564800
}
```

**Response:**
```json
{
  "inquiry_id": "err-1234",
  "command_vector": [3, 301, 0, 0.82, 1, 0, 0, 0],
  "alert_level": 3,
  "heuristic_id": 301,
  "target_node_id": 0,
  "confidence_score": 0.82,
  "action_parameters": [1, 0, 0, 0],
  "analysis_summary": "Model confidence drop detected. Input may be out-of-distribution.",
  "recommended_actions": [
    "Verify input preprocessing matches training data",
    "Check for anomalous input patterns",
    "Consider retraining with similar samples"
  ],
  "debug_insights": [
    "Confidence score 0.45 is below typical threshold 0.7",
    "Error occurred during batch processing",
    "Model: cortex-v1"
  ],
  "identified_patterns": [
    {
      "pattern_id": "PAT-001",
      "pattern_name": "Low Confidence Output",
      "description": "Model produces low-confidence predictions",
      "match_confidence": 0.89,
      "indicators": ["confidence_score < 0.5", "InferenceError"]
    }
  ],
  "similar_errors": [
    {
      "error_id": "err-987",
      "error_type": "InferenceError",
      "similarity_score": 0.76,
      "resolution": "Adjusted input normalization",
      "skill_id": "skill-norm-v2"
    }
  ],
  "processing_time_ms": 12.4,
  "timestamp": 1699564801
}
```

#### GET /heart/health

Returns HEART service health status.

**Response:**
```json
{
  "available": true,
  "status": "online",
  "avg_latency_ms": 15.2,
  "total_queries": 1247,
  "success_rate": 0.987,
  "cerebras_status": "simulator"
}
```

#### GET /heart/stats

Returns cumulative statistics about HEART usage.

**Response:**
```json
{
  "total_inquiries": 1247,
  "successful_responses": 1231,
  "failed_responses": 16,
  "avg_response_time_ms": 15.2,
  "min_response_time_ms": 8.1,
  "max_response_time_ms": 234.5,
  "error_type_counts": {
    "InferenceError": 523,
    "TypeError": 312,
    "NetworkError": 189,
    "ModelError": 145
  },
  "heuristic_usage": {
    "101": 312,
    "201": 189,
    "301": 668
  }
}
```

### Error-to-Metrics Synthesis

The HEART service converts CORTEX error characteristics into network metrics that the transformer can process:

```go
func (hs *HEARTService) synthesizeMetricsFromError(
    inquiry *HEARTErrorInquiry) *HEARTInput {

    measurements := make([]float32, 16)

    // Map error characteristics to measurement vector
    measurements[0] = float32(hash(inquiry.ErrorType) / 1000.0)  // Error type hash
    measurements[1] = calculateSeverity(inquiry)                  // Severity score
    measurements[2] = float32(len(inquiry.ErrorMessage)) / 100.0  // Message length
    measurements[3] = float32(len(inquiry.StackTrace)) / 1000.0   // Stack depth
    measurements[4] = inquiry.ConfidenceScore                     // Model confidence
    measurements[5] = float32(len(inquiry.Metadata))              // Metadata count
    measurements[6] = calculateContextComplexity(inquiry)         // Context complexity
    measurements[7] = timeOfDay(inquiry.Timestamp)                // Time of day factor
    // measurements[8-15] reserved for future use

    return &HEARTInput{
        Measurements: measurements,
        NodeIDs: []uint32{0},
        TimeSlices: []uint32{0},
    }
}
```

### CORTEX Client Integration

KNIRVCORTEX includes a HEART client (`heart_client.rs`) that automatically queries the service:

```rust
// In KNIRVCORTEX rust-wasm/src/lib_with_heart.rs
impl KnirvCortex {
    pub async fn run_cognitive_task(&mut self, input: InferenceInput)
        -> Result<InferenceOutput, CortexError> {

        match self.inner_runtime.run_inference(&input).await {
            Ok(output) => Ok(output),
            Err(e) => {
                // Query HEART for error analysis
                if let Some(response) = self.query_heart_for_error(
                    &e.error_type,
                    &e.message,
                    &input.context
                ).await {
                    // Apply HEART recommendations
                    self.apply_heart_heuristics(&response);

                    // Return enhanced error with recommendations
                    Err(CortexError::enhanced(e, response))
                } else {
                    Err(e)
                }
            }
        }
    }
}
```

### Running HEART Service

**Start the service:**
```bash
# From go_transformer directory
go run . -heart-service -port 8080

# Or with Cerebras device
export CEREBRAS_CMADDR="10.10.10.100:9000"
go run . -heart-service -port 8080 -use-device
```

**Test the service:**
```bash
# Health check
curl http://localhost:8080/heart/health

# Analyze an error
curl -X POST http://localhost:8080/heart/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "error_id": "test-001",
    "error_type": "InferenceError",
    "error_message": "Low confidence output",
    "error_context": "Testing HEART integration",
    "timestamp": 1699564800
  }'
```

### Performance Impact

- **CORTEX Query Latency**: 10-50ms (includes network + HEART processing)
- **HEART Processing**: 1-15ms on simulator, <1ms on WSE
- **Network Overhead**: 2-10ms (localhost)
- **Caching**: Recent responses cached in CORTEX for 60 seconds
- **Rate Limiting**: Max 1 query/second from each CORTEX instance

### Configuration

**CORTEX Configuration** (`HEARTConfig` in cortex.wasm):
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

**HEART Service Configuration** (environment variables):
```bash
export HEART_PORT=8080
export HEART_CEREBRAS_PROGRAM=cerebras/build/heart_program
export HEART_WEIGHTS=cerebras/weights/model_weights.npz
export HEART_USE_SIMULATOR=true
export HEART_LOG_LEVEL=info
```

## Performance Characteristics

### Latency (Expected on WSE-3)

- **End-to-End Inference**: <1ms for full batch (16,384 samples)
- **Per-Sample Latency**: <100 microseconds
- **Host Transfer Overhead**: ~2-5ms (amortized over batch)

### Throughput

- **Samples/Second**: >1,000,000 (with batching)
- **Network Updates/Second**: >15,000 (64-sample sequences)

### Memory Footprint

- **Model Weights**: ~10 MB
- **Activation Memory**: ~2 MB per PE
- **Total PE Memory Used**: ~128 MB (64 PEs)

## Configuration Parameters

Edit `params.json` to adjust model configuration:

```json
{
  "GRID_WIDTH": 8,           // Number of PEs horizontally
  "GRID_HEIGHT": 8,          // Number of PEs vertically
  "SEQUENCE_LENGTH": 64,     // Time slices (T)
  "NUM_NODES": 256,          // Network nodes (N)
  "EMBEDDING_DIM": 256,      // Model embedding dimension
  "NUM_HEADS": 8,            // Attention heads
  "NUM_LAYERS": 3,           // Transformer layers
  "MEASURE_VEC_SIZE": 16,    // Input metric vector size
  "CONTROL_VEC_SIZE": 8,     // Output command vector size
  "SEQ_PER_PE": 8,           // Sequences per PE (auto-calculated)
  "EMBED_PER_PE": 32         // Embedding dims per PE (auto-calculated)
}
```

**Important:** After changing parameters, recompile:
```bash
make clean
make compile
```

## Troubleshooting

### Compilation Errors

**Issue:** `cslc: command not found`
```bash
# Ensure Cerebras SDK is in PATH
export PATH="/opt/cerebras/sdk/bin:$PATH"
```

**Issue:** Fabric dimension mismatch
```bash
# Ensure GRID_WIDTH * GRID_HEIGHT matches your fabric allocation
# For simulator, typical max is 16x16
make compile GRID_WIDTH=8 GRID_HEIGHT=8
```

### Runtime Errors

**Issue:** Weight shape mismatch
```bash
# Verify weight file shapes match params.json
cs_python -c "import numpy as np; data = np.load('weights/model_weights.npz'); \
              print({k: v.shape for k, v in data.items()})"
```

**Issue:** Out of memory on PE
```bash
# Reduce sequences per PE by increasing grid size
# Or reduce embedding dimensions
```

### Performance Issues

**Issue:** Slow inference
```bash
# Check if running on simulator vs actual hardware
# Simulator is 10-100x slower

# Verify batch size utilization
# Larger batches amortize transfer overhead
```

## Development

### Modifying the CSL Kernel

1. Edit `heart_pe.csl` to modify computation logic
2. Recompile: `make compile`
3. Test: `make test`

### Adding New Metrics

1. Update `MEASURE_VEC_SIZE` in `params.json`
2. Update input data preparation in Go bridge
3. Recompile and test

### Changing Model Architecture

1. Update layer counts, dimensions in `params.json`
2. Adjust weight tensors in `run_heart.py`
3. Recompile CSL and export new weights

## References

- [Cerebras SDK Documentation](https://sdk.cerebras.net/)
- [CSL Language Reference](https://sdk.cerebras.net/csl/language/index.html)
- [HEART System Design Document](../HEART_SDD.md)
- [Gorgonia Transformer Implementation](../main.go)

## License

Part of the KNIRV Network project. See top-level LICENSE.

## Support

For Cerebras WSE specific issues:
- Cerebras SDK Support: support@cerebras.net
- SDK Documentation: https://sdk.cerebras.net/

For HEART/KNIRV issues:
- Project Repository: [GitHub](https://github.com/guiperry/KNIRV_NETWORK)
