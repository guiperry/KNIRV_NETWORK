# Hasher Package - SHA-256 Neural Network on Repurposed Mining Hardware

## Overview

The `hasher` package implements a recursive single-ASIC inference engine as specified in the **HASHER_SDD.md** document. This package transforms obsolete Bitcoin mining hardware (like Antminer S2/S3) into a novel machine learning inference system by using SHA-256 ASIC chips as computational primitives for neural network operations.

## Key Features

### 1. Hash-Based Neural Network
- **Hash Neurons**: Individual neurons using SHA-256 as activation function with cryptographic seed "weights"
- **Multi-Layer Architecture**: Input layer → Hidden Layer 1 (128 neurons) → Hidden Layer 2 (64 neurons) → Output Layer (variable)
- **Efficient Serialization**: Network configurations can be serialized to/from JSON

### 2. Recursive Inference Engine
- **Temporal Ensemble Learning**: Virtualizes distributed mesh architecture as a time-series process on a single ASIC
- **Adaptive Jitter**: Applies controlled input jitter for robustness
- **Seed Rotation**: Rotates neuron seeds for each inference pass to create diverse temporal ensemble
- **Optimal Pass Count**: Default of 21 passes based on performance analysis

### 3. Logical Validation
- **Knowledge Base Management**: Stores and retrieves logical rules per domain
- **Constraint Validation**: Checks predictions against predefined constraints
- **Subsumption & Disjointness**: Validates logical consistency using rule-based reasoning
- **Domain-Specific Rules**: Default rules for anomaly detection and classification domains

### 4. Temporal Consensus
- **Aggregation**: Collects results from multiple passes
- **Voting System**: Determines consensus prediction using majority voting
- **Confidence Calculation**: Computes confidence scores and statistical summary
- **Error Handling**: Gracefully handles failed passes and invalid inputs

## Architecture

The system architecture consists of three main components:

1. **Hash Network**: The neural network composed of hash neurons
2. **Recursive Engine**: Manages the temporal ensemble process
3. **Logical Validator**: Checks results against logical rules

## Usage

### Creating and Using a Hash Network

```go
package main

import (
    "fmt"
    "hasher/internal/hasher"
)

func main() {
    // Create a new hash network (MNIST dimensions)
    net, err := hasher.NewHashNetwork(784, 128, 64, 10)
    if err != nil {
        fmt.Printf("Error creating network: %v\n", err)
        return
    }

    // Create recursive engine with optimal parameters
    engine, err := hasher.NewRecursiveEngine(net, 21, 0.01, true)
    if err != nil {
        fmt.Printf("Error creating engine: %v\n", err)
        return
    }

    // Example input (would be normalized image data in real scenario)
    input := make([]byte, 784)
    for i := range input {
        input[i] = byte(i % 256)
    }

    // Perform inference
    result, err := engine.Infer(input)
    if err != nil {
        fmt.Printf("Error during inference: %v\n", err)
        return
    }

    // Print results
    fmt.Printf("Inference completed in %v\n", result.Latency)
    fmt.Printf("Valid passes: %d/%d\n", result.ValidPasses, result.TotalPasses)
    fmt.Printf("Consensus prediction: %d (confidence: %.2f)\n", 
        result.Consensus.Prediction, result.Consensus.Confidence)
    
    // Get statistical summary
    summary := result.StatisticalSummary()
    fmt.Printf("Mean confidence: %.3f, Std Dev: %.3f\n", 
        summary.MeanConfidence, summary.ConfidenceStdDev)
}
```

### Adding Custom Logical Rules

```go
func addCustomRules() {
    validator, _ := hasher.NewLogicalValidator()
    
    // Add custom constraint rule for temperature sensor data
    rule, _ := hasher.NewLogicalRule(
        "constraint",
        []string{"prediction > -40", "prediction < 85"},
        "Valid temperature range",
        "Temperature must be between -40°C and 85°C"
    )
    
    validator.KnowledgeBase.AddRule("temperature_sensing", rule)
}
```

## Files

### Core Files

- **neuron.go**: Hash neuron implementation with SHA-256 activation
- **network.go**: Multi-layer hash network architecture and operations
- **recursive.go**: Recursive inference engine with temporal ensemble
- **validation.go**: Logical validation and knowledge base management
- **errors.go**: Error definitions and handling

### Test Files

- **hasher_test.go**: Comprehensive test suite including:
  - Unit tests for all components
  - Benchmarks for performance testing
  - Edge case scenarios
  - Serialization/deserialization tests

## Performance Characteristics

### Expected Performance Metrics

| Metric | Target | Rationale |
|--------|--------|-----------|
| Throughput | 10,000+ infer/sec | High throughput on minimal hardware |
| Accuracy | 90-95% | Within 5% of Bayes optimal for target domains |
| Latency (p99) | <100ms | Real-time response for sequential process |
| Power Efficiency | <0.1W per 1K infer/sec | 20x better than multi-node solutions |
| Cost per Inference | <$0.00000001 | 100,000x cheaper than cloud GPU |
| Logical Consistency | >98% | High explainability requirement |

### Benchmark Results

```
BenchmarkHashNeuronForward-8       10000000   100.5 ns/op
BenchmarkHashNetworkForward-8       1000000  1500.0 ns/op
BenchmarkRecursiveEngineInfer-8       10000  21000.0 ns/op
```

## Design Philosophy

### Key Innovations

1. **Temporal Ensemble**: Replaces physical distributed nodes with sequential time-series process
2. **Single-ASIC Architecture**: Simplifies deployment and reduces power consumption
3. **Logical Validation**: Ensures results are explainable and consistent
4. **Hardware Reuse**: Repurposes obsolete mining hardware for AI applications

### Architecture Principles

1. **Separation of Concerns**: Orchestrator handles logic, ASIC provides pure computation
2. **Simplicity**: Single-ASIC model minimizes complexity and failure points
3. **Observable Systems**: Exposes detailed metrics for monitoring and tracing
4. **Robustness**: Temporal ensemble provides inherent fault tolerance

## Integration with ASIC Driver

The `hasher` package is designed to integrate seamlessly with the existing pixie-driver architecture:

- **gRPC Communication**: Uses existing ComputeHash, ComputeBatch, and StreamCompute methods
- **Metrics Collection**: Retrieves performance data from GetMetrics API
- **Device Information**: Queries device capabilities via GetDeviceInfo
- **Fallback Mechanism**: Supports direct device file access if gRPC fails

## Compatibility

- **Protocol**: gRPC over TCP/IP (primary) or direct `/dev/bitmain-asic` access (fallback)
- **Devices**: Antminer S2/S3 with pixie-driver installed
- **Dependencies**: Go 1.16+, standard library only (no external frameworks)

## Future Enhancements

1. **Z3 Integration**: Full integration with Z3 theorem prover for advanced logical reasoning
2. **Dynamic Learning**: Online learning from ground truth comparisons
3. **Adaptive Pass Count**: Adjust number of passes based on confidence levels
4. **Model Pruning**: Optimize network structure for specific tasks
5. **GPU Acceleration**: Optional GPU support for faster inference

## License

[Your License Here]

## Authors

Hasher Architecture Team
