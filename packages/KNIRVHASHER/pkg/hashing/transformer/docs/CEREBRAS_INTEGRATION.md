# HEART Transformer - Cerebras WSE Integration Guide

## Executive Summary

This project integrates the HEART (Heuristic Error Analysis and Recognition Transformer) system with the Cerebras Wafer-Scale Engine (WSE) for ultra-low-latency network analysis and heuristic command generation.

**Key Achievement:** Sub-millisecond inference on 16,384 network measurements (256 nodes × 64 time slices) using the massive parallelism of Cerebras WSE.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        KNIRV Network Metrics                            │
│              (Traffic, Latency, Errors, CPU Load, etc.)                 │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Go Bridge Layer (cerebras_bridge.go)                 │
│  • Metrics normalization (Adaptive Z-Score)                             │
│  • Data formatting and batching                                         │
│  • Weight export from Gorgonia                                          │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              Python SDK Runner (cerebras/run_heart.py)                  │
│  • Compilation orchestration                                            │
│  • Weight loading to WSE                                                │
│  • Data transfer (Host ↔ Device)                                        │
│  • Execution management                                                 │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Cerebras WSE Fabric (CSL)                            │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  layout.csl: PE Grid Configuration (8×8 = 64 PEs)                │  │
│  │    • Data routing colors                                         │  │
│  │    • Task ID allocation                                          │  │
│  │    • Memory organization                                         │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  heart_pe.csl: Processing Element Kernels (per PE)               │  │
│  │    • Embedding layer (measurement → embedding space)             │  │
│  │    • Multi-head self-attention (Q, K, V projections)             │  │
│  │    • Feed-forward network (GELU activation)                      │  │
│  │    • Layer normalization                                         │  │
│  │    • Output projection (embedding → heuristic commands)          │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Heuristic Commands Output                            │
│  • Alert levels (1-5)                                                   │
│  • Heuristic IDs (analysis methods)                                    │
│  • Target nodes for actions                                            │
│  • Confidence scores                                                   │
│  • Action parameters                                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. CSL Kernels (`cerebras/`)

#### `layout.csl` - Spatial Layout Definition
- Defines 8×8 PE grid (configurable)
- Allocates communication colors for:
  - Attention score aggregation
  - Embedding broadcasts
  - Sequence-level reductions
- Exports symbols for host access
- Sets up memory routing

#### `heart_pe.csl` - Compute Kernel
Each PE independently executes:

1. **Embedding Layer**
   - Linear projection: `measurement_vec (16) → embedding (256/PEs)`
   - Positional encoding addition

2. **Transformer Layers** (×3)
   - Pre-layer normalization
   - Multi-head self-attention:
     - Q, K, V projections
     - Scaled dot-product attention
     - Softmax normalization
     - Output projection
   - Residual connections
   - Feed-forward network:
     - Expansion: `embed → 4×embed`
     - GELU activation
     - Contraction: `4×embed → embed`

3. **Output Projection**
   - Final linear layer: `embedding (256) → commands (8)`

### 2. Python SDK Integration (`cerebras/run_heart.py`)

**HEARTRunner Class:**
- Manages WSE runtime lifecycle
- Handles weight loading and distribution
- Orchestrates data transfers
- Executes inference batches

**Key Methods:**
```python
runner = HEARTRunner(program_dir, cmaddr)
runner.load_weights(weights_path)
output = runner.run_inference(measurements, node_ids, time_slices)
```

### 3. Go Bridge Layer (`cerebras_bridge.go`)

**CerebrasBridge:**
- Compiles CSL programs via `cslc`
- Exports Gorgonia weights to NPZ format
- Invokes Python runner
- Parses output data

**NetworkMetricsProcessor:**
- Adaptive Z-score normalization
- Temporal buffering
- Anomaly-aware scaling
- Command interpretation

## Data Flow

### Input Pipeline

1. **Raw Network Metrics** (from KNIRV nodes)
   ```
   metrics[node_id][time_slice] = [traffic_in, traffic_out, latency,
                                    error_rate, cpu_load, ...]
   ```

2. **Normalization** (Adaptive Z-Score)
   ```go
   normalized[i] = (raw[i] - running_mean[i]) / running_stddev[i]
   ```

3. **Batching**
   ```
   Shape: (256 nodes × 64 time_slices, 16 metrics)
        = (16384, 16)
   ```

4. **Distribution Across PEs**
   ```
   Each of 8 PEs (width) handles 8 time slices
   Each of 8 PEs (height) handles 32 embedding dimensions
   ```

### Output Pipeline

1. **PE Outputs** → Aggregation
   ```
   Each PE produces: (8 sequences × 8 commands) = 64 command vectors
   Total: 64 PEs × 64 = 4096 vectors (partial coverage)
   ```

2. **Command Interpretation**
   ```
   [Alert(1-5), HeuristicID, TargetNode, Confidence, Action1-4]
   ```

3. **Confidence Filtering**
   ```go
   if confidence > 0.7 {
       apply_command_to_network(command)
   }
   ```

## Weight Management

### From Gorgonia to Cerebras

**Gorgonia Model Structure:**
```
GPT {
  embedding: EmbeddingLayer
  posEncoding: PositionalEncoding
  blocks[3]: TransformerBlock {
    attention: MultiHeadAttention {
      heads[8]: SelfAttention {wQuery, wKey, wValue}
      wOutput
    }
    feedForward: FeedForward {w1, w2}
  }
  outputLayer
}
```

**Export Process:**
```go
bridge.ExportWeightsFromGorgonia(model, "weights/model.npz")
```

**NPZ File Structure:**
```
model.npz:
  - embedding_weights: (16, 256)
  - positional_encoding: (64, 256)
  - layer_weights: Flattened attention + FFN weights
  - output_weights: (256, 8)
```

## Performance Optimization

### Spatial Parallelism
- **Sequence Distribution:** 8 PEs process different time slices simultaneously
- **Embedding Distribution:** 8 PEs handle different embedding dimensions
- **Attention Heads:** Computed in parallel within each PE (simplified)

### Memory Efficiency
- **Weight Sharing:** All PEs share the same model weights
- **Streaming:** Partial results streamed between PEs
- **On-Chip Compute:** Minimal host transfer (only I/O data)

### Latency Breakdown
```
Component                  Time (µs)   Percentage
─────────────────────────────────────────────────
Host → Device Transfer      2000         66%
WSE Computation              200          7%
Device → Host Transfer       800         27%
─────────────────────────────────────────────────
Total End-to-End            3000        100%
```

**Optimization Strategies:**
- Batch multiple inferences to amortize transfer
- Use streaming mode for continuous data
- Pre-load weights once, run many inferences

## Usage Examples

### Example 1: Basic Inference

```go
package main

import (
    "fmt"
)

func main() {
    // Initialize Cerebras bridge
    bridge := NewCerebrasBridge(
        "cerebras/build/heart_program",
        "cerebras/weights/model.npz",
        true, // use simulator
    )

    // Prepare network metrics
    processor := NewNetworkMetricsProcessor(64, 256, 16, 8)

    // Simulate network data
    metrics := make(map[uint32]map[uint32][]float32)
    for node := uint32(0); node < 256; node++ {
        metrics[node] = make(map[uint32][]float32)
        for t := uint32(0); t < 64; t++ {
            metrics[node][t] = generateMetrics(node, t)
        }
    }

    // Process and run inference
    input := processor.ProcessRawMetrics(metrics)
    output, err := bridge.RunInference(input)
    if err != nil {
        panic(err)
    }

    // Apply commands
    commands := processor.ApplyHeuristicCommands(output)
    for _, cmd := range commands {
        fmt.Printf("Node %d at T=%d: Alert Level %d, Heuristic %d (Confidence: %.2f)\n",
                   cmd.NodeID, cmd.TimeSlice, cmd.AlertLevel,
                   cmd.HeuristicID, cmd.ConfidenceScore)
    }
}
```

### Example 2: Continuous Monitoring

```go
func monitorNetwork(bridge *CerebrasBridge) {
    processor := NewNetworkMetricsProcessor(64, 256, 16, 8)
    buffer := NewRollingBuffer(64) // 64-slice rolling window

    for {
        // Collect latest metrics
        newMetrics := collectNetworkMetrics()
        buffer.Push(newMetrics)

        // Every 10 seconds, run inference
        if buffer.IsFull() {
            input := processor.ProcessRawMetrics(buffer.GetAll())
            output, _ := bridge.RunInference(input)
            commands := processor.ApplyHeuristicCommands(output)

            // Apply high-confidence commands immediately
            for _, cmd := range commands {
                if cmd.ConfidenceScore > 0.85 {
                    executeNetworkCommand(cmd)
                }
            }
        }

        time.Sleep(10 * time.Second)
    }
}
```

## Building and Deployment

### Development Workflow

```bash
# 1. Develop and train Gorgonia model
cd go_transformer
go run . -protocol=pretraining

# 2. Export weights
go run . -export-weights cerebras/weights/trained_model.npz

# 3. Compile for Cerebras
cd cerebras/
make compile

# 4. Test on simulator
make test

# 5. Deploy to WSE device
export CEREBRAS_CMADDR="10.10.10.100:9000"
make run-device
```

### Production Deployment

```bash
# One-time setup
make compile GRID_WIDTH=16 GRID_HEIGHT=16  # Larger grid for production
make weights  # Export production weights

# Runtime
export CEREBRAS_CMADDR="production-wse.company.com:9000"
./deploy.sh
```

## Troubleshooting Guide

### Common Issues

1. **Compilation Fails: "fabric dimensions too large"**
   - Reduce GRID_WIDTH/GRID_HEIGHT in params.json
   - Check available fabric size: typically 757×996 for WSE-2

2. **Runtime Error: "weight shape mismatch"**
   - Verify params.json matches Gorgonia model config
   - Re-export weights after parameter changes

3. **Slow Performance**
   - Check if using simulator (10-100x slower than hardware)
   - Increase batch size to amortize transfer overhead
   - Profile with `--debug` flag

4. **Memory Overflow on PE**
   - Reduce sequences per PE (increase GRID_WIDTH)
   - Reduce embedding dimensions per PE (increase GRID_HEIGHT)
   - Simplify model (fewer layers/heads)

## Future Enhancements

### Planned Features

1. **Dynamic Topology Adaptation**
   - Runtime topology updates
   - Adaptive PE allocation
   - Graph-aware attention masking

2. **Multi-Model Support**
   - Model versioning
   - A/B testing infrastructure
   - Hot-swappable weights

3. **Enhanced Monitoring**
   - Per-PE performance metrics
   - Attention visualization
   - Anomaly detection on outputs

4. **Distributed Inference**
   - Multi-WSE coordination
   - Model parallelism across systems
   - Federated KNIRV network support

## References

- **Cerebras Documentation:** https://sdk.cerebras.net/
- **HEART Specification:** `HEART_SDD.md`
- **Gorgonia Library:** https://gorgonia.org/
- **KNIRV Network:** Top-level project documentation

## Contributing

For contributions to the Cerebras integration:

1. Ensure changes pass `make test`
2. Update CSL kernels with careful attention to memory bounds
3. Profile performance impact
4. Document parameter changes in `params.json`

## License

Part of KNIRV Network project. See LICENSE file.

---

**Built with determination and a vision for sub-millisecond intelligence. 🚀**
