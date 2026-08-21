# HASHER - SHA-256 Neural Network on Repurposed Mining Hardware
<p align="center">
<img src="assets/logo-clear.png" width="350" alt="logo">
</p>

## Overview

HASHER implements a recursive single-ASIC inference engine as specified in the **HASHER_SDD.md** document. This package transforms obsolete Bitcoin mining hardware (like Antminer S2/S3) into a novel machine learning inference system by using SHA-256 ASIC chips as computational primitives for neural network operations.

### KNIRVHASHER Pipeline

KNIRVHASHER extends HASHER with a complete data pipeline for training user-centric logic gate hash networks. The experimental stealth mode includes three pipeline stages that can be called non-interactively as background processes:

1. **0_DATA_CONNECTOR**: Receives gRPC streams from KNIRVSERVER, decrypts chunks, and writes raw `.md` files to KNIRVBASE
2. **1_DATA_MINER**: Processes `.md` files through SpaCy NLP, normalizes security data, and writes `.arrow` IPC files
3. **2_DATA_ENCODER**: Encodes `.arrow` batches into 80-byte `.nrv` Tier-3 Brackets with BGE embeddings and NRV KB lookups
4. **3_DATA_SEEDER**: Mines proof-of-work-witnessed assertions via `EvolutionaryHarness` — a real (1+1)-style evolution strategy over candidate nonces, scored by Hamming-similarity fitness (see below) — and writes them to the seed ledger

The pipeline transforms user ontology data into `.nrv` datasets for future global model updates across the KNIRV network.

### 3_DATA_SEEDER Components

- **EvolutionaryHarness** (`pkg/training/evolutionary.go`): population of candidate nonces per context, `BitwiseMutation`/`BitcoinAwareMutate` for mutation, `calculateAlignmentReward` for fitness, `GetEliteSeeds` for selection — this is the real, running mining loop
- **DualSeedWriter**: persists winning seeds to the canonical `seed_writes.jsonl` ledger

**Updated Algorithm:** The system now uses the ASIC as a **deterministic bucket generator** rather than a generic hash function. By setting a "Difficulty 1" target, we use the first valid Nonce discovered as the LSH (Locality Sensitive Hashing) signature, maintaining the 500 GH/s speed advantage by repurposing the mining hardware's natural state.

**Training Innovation:** Discovery of "Golden Seeds" is now optimized using a **Hamming Similarity Gradient** in the evolutionary process. Instead of binary prefix matching, the system rewards total matching bits across the 32-bit word, providing a continuous signal for convergence even at high difficulties.

## Key Features

### 1. Unified Hasher Engine

The **UnifiedHasherEngine** merges `HasherTransformer`, `HashNetwork`, and `RecursiveEngine` into a single seed-based neural system with hardware-accelerated inference and safe software fallbacks.

- **Three Inference Modes**: Transformer (MHA + FFN), Recursive (21-pass temporal ensemble), and Feedforward (simple 3-layer)
- **Hardware-First, Software-Fallback**: Every projection attempts `hashMethod.ComputeBatch` first; falls back to `seedToFloat` if hardware is unavailable or errors
- **Mode Polymorphism**: The same seed network can run in any mode without retraining
- **Train-Once, Run-Anywhere**: Seeds trained by the evolutionary harness are valid for all modes

```go
engine := transformer.NewUnifiedHasherEngineWithConfig(cfg, seeds, hashMethod, transformer.ModeTransformer)
out := engine.Forward(tokenIDs)

engine.SetMode(transformer.ModeRecursive)
result, _ := engine.Infer(inputBytes)

engine.SetMode(transformer.ModeFeedforward)
pred, conf, _ := engine.Predict(inputBytes)
```

### 2. Hardware Router

The **HardwareRouter** routes projections through hardware (`hashMethod.ComputeBatch`) when available, falling back to software (`seedToFloat`) when needed.

- **Fallback Strategies**: `FallbackSoftware`, `FallbackError`, `FallbackMixed`
- **Projection Caching**: Stores recent `(inputHash, seedHash)` → `float32[]` mappings to compensate for lower software throughput
- **Per-Projection Fallback**: If `ComputeBatch` fails mid-forward-pass, falls back to `seedToFloat` for that projection only

```go
router := transformer.NewHardwareRouter(hashMethod, transformer.FallbackMixed)
out, err := router.Project(input, seeds)
scores := router.HashToVocab(hidden, outputSeed, vocabSize)
```

### 3. SeedStore

The **SeedStore** holds all seed parameters in one place, replacing scattered seed maps across `HasherTransformer`, `HashNetwork`, and `RecursiveEngine`.

```go
store := &transformer.SeedStore{
    Embeddings: embeddings,
    Positional: positional,
    Layers:     layers,
    OutputSeed: outputSeed,
    Seeds1:     seeds1,
    Seeds2:     seeds2,
    SeedsOut:   seedsOut,
}
```

### 4. Hardware Acceleration Tiers

| Tier | Method | Expected Throughput |
|------|--------|-------------------|
| 1 | ASIC (`ASICMethod`) | 10^9+ hashes/sec |
| 2 | CUDA (`CudaMethod`) | 10^6-10^7 hashes/sec |
| 3 | eBPF/uBPF | 10^5-10^6 hashes/sec |
| 4 | Software fallback | 10^3-10^4 hashes/sec |

### 5. Hash-Based Neural Network
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

## 🔐 Updated Algorithm: Nonce-Mining for LSH

### Hardware Reality Pivot

The ASIC is designed for Bitcoin mining (finding nonces where SHA256(SHA256(header+nonce)) < target), but the HASHER needs deterministic hashing (SHA256(input || seed) → fixed output). The original design assumed the BM1382 could perform arbitrary SHA-256 hashes. Testing has confirmed the ASIC is hard-wired for the Bitcoin mining loop.

**Revised Core Innovation:** Instead of using the ASIC as a hash function, HASHER now uses it as a **deterministic bucket generator**. By setting a "Difficulty 1" target, we use the first valid **Nonce** discovered as the LSH signature. This maintains the 500 GH/s speed advantage by repurposing the mining hardware's natural state.

### From Hashing to Mining

We pack the 128-bit LSH projections into the standard 80-byte Bitcoin block header structure to be processed by the `0x52 (TXTASK)` protocol.

**Header Mapping:**

* **Version (4 bytes):** Used as a "Salt" or Seed for the LSH forest.
* **Previous Block Hash (32 bytes):** Stores the first 4 LSH projections (4x32-bit floats).
* **Merkle Root (32 bytes):** Stores the next 4 LSH projections.
* **Timestamp/Bits (8 bytes):** Fixed metadata to ensure determinism.
* **Nonce (4 bytes):** The output we seek from the ASIC.

### Temporal Recursive Nonces

To handle the **61MB RAM constraint**, we implement a **Temporal Recursive Algorithm**. If a bucket collision occurs, we "mine deeper" by using the previously found nonce as a seed for the next task.

**Multi-Nonce Signature:**

Instead of one 128-bit signature, we collect the first N nonces that satisfy the target.

* **LSH Bucket ID:** Determined by the first valid nonce
* **Rerank Filter:** Uses subsequent nonces for fine-grained similarity
* **Determinism Guarantee:** We specify a `Nonce Range` (e.g., 0 to 1,000,000). The ASIC will always find the same "Golden Nonce" for the same projection data within that range.

### Performance Comparison

| Metric | Original Design | Updated Design (Mining) |
| --- | --- | --- |
| **Hashing Latency** | 100µs (Direct) | ~1-2ms (Mining Search + USB) |
| **Determinism** | High (SHA-256) | High (First Nonce in Range) |
| **CPU Load** | Moderate | Low (Header packing is simple math) |
| **RAM Usage** | 61MB | 61MB (No change, uses mmap index) |

**Bottleneck Analysis:**
The primary bottleneck is no longer SHA-256 computation, but the **USB Bulk Transfer** and the **time-to-first-nonce**. At 500 GH/s, the hardware finds a Difficulty 1 nonce in nanoseconds, meaning the total search time remains dominated by the network call to the API server (~42ms).

---

## 🔧 Usage

### Current Working Make Commands

```bash
# Show all available commands
make help

# Build the CLI (primary command-line interface)
make cli

# Build hasher-host for current platform (orchestrator)
make build-host

# Build hasher-server for MIPS (Antminer)
make build-server-mips

# Build hasher-host for all platforms
make build-host-all

# Build all components (host, server, CLI)
make build

# Build and deploy everything
make embed-binaries

# Deploy server to Antminer
make deploy-server

# Clean build artifacts
make clean

# Install dependencies
make deps

# Run tests
make test

# Run linters
make lint
```

### Device-Specific Commands


### Development Commands

```bash

# Test training functionality
make test-training

# Generate protobuf code
make proto

# Compile eBPF programs
make ebpf

# Generate all code (proto, ebpf, go generate)
make generate
```

### Platform-Specific Host Builds

```bash
# Build for Linux x86_64
make build-host-linux-amd64

# Build for macOS Intel
make build-host-darwin-amd64

# Build for macOS Apple Silicon
make build-host-darwin-arm64
```

### Quick Start

1. **Build the CLI and Host**
   ```bash
   make embed-binaries
   ```

2. **Run the Host with ASIC Device**
   ```bash
   ./bin/hasher-host --device=your-asic-ip --discover=false
   ```

3. **Run the CLI**
   ```bash
   ./bin/hasher
   ```

## Architecture

The system architecture consists of three main components:

1. **UnifiedHasherEngine**: Single seed-based engine supporting transformer, recursive, and feedforward inference modes
2. **HardwareRouter**: Hardware-first projection layer with software fallback
3. **RecursiveEngine**: Legacy temporal ensemble (now delegates to UnifiedHasherEngine in recursive mode)

### Unified Inference Architecture

```
                     ┌──────────────────────────┐
                     │   UnifiedHasherEngine    │
                     │  (replaces legacy systems)│
                     └──────────┬───────────────┘
                                │
                ┌───────────────┼───────────────┐
                │               │               │
       ┌────────▼──────┐ ┌─────▼──────┐ ┌──────▼────────────┐
       │ Transformer   │ │ Recursive  │ │  FeedForward      │
       │ Mode          │ │ Mode       │ │  Mode             │
       │ (MHA + FFN)   │ │ (21-pass)  │ │  (simple 3-layer) │
       │ with HW accel │ │ temporal   │ │  with HW accel    │
       └───────┬──────┘ └─────┬──────┘ └──────┬────────────┘
               │               │               │
               └───────────────┼───────────────┘
                               │
                     ┌─────────▼─────────┐
                     │  HardwareRouter  │
                     │  ┌─────────────┐ │
                     │  │ ComputeBatch│ │ ← ASIC/CUDA/eBPF
                     │  └─────────────┘ │
                     │  ┌─────────────┐ │
                     │  │ seedToFloat │ │ ← CPU fallback
                     │  └─────────────┘ │
                      └──────────────────┘
```

### FoX Attention Mechanism

The transformer mode uses **FoX (Formula X)** attention, which combines sharp softmax-style attention weights with a scalar decay chain for temporal forgetting:

```
sum_{j=1}^{t} exp(q_t^T k_j) * (prod_{s=j+1}^{t} alpha_s) * v_j
```

- **Sharp attention**: `exp(q_t^T k_j)` concentrates weight on relevant tokens
- **Temporal decay**: `prod_{s=j+1}^{t} alpha_s` reduces influence of distant tokens
- **Numerical stability**: Clamped exp values and epsilon-floored cumulative decay products
- **Backward compatibility**: Legacy `HasherTransformer` automatically upgrades to FoX via `NewUnifiedHasherEngineFromHasherTransformer`

### ASIC Protocol Implementation

The updated implementation uses the BM1382 mining protocol:

```go
// BuildMiningHeader packs LSH projections into 80-byte Bitcoin header
func BuildMiningHeader(projections []float32, salt uint32) []byte {
    header := make([]byte, 80)
    
    // Version (Salt)
    binary.LittleEndian.PutUint32(header[0:4], salt)
    
    // Previous Block Hash (Projections 0-7)
    for i := 0; i < 8; i++ {
        val := math.Float32bits(projections[i])
        binary.LittleEndian.PutUint32(header[4+(i*4):8+(i*4)], val)
    }
    
    // Merkle Root (Projections 8-15)
    for i := 0; i < 8; i++ {
        val := math.Float32bits(projections[8+i])
        binary.LittleEndian.PutUint32(header[36+(i*4):40+(i*4)], val)
    }

    // Fixed Difficulty Bits (0x1d00ffff = Difficulty 1)
    binary.LittleEndian.PutUint32(header[72:76], 0x1d00ffff)
    
    return header
}
```

## CLI Usage

The Hasher CLI provides an interactive terminal interface for managing ASIC devices, running diagnostics, and performing inference. Built with Bubble Tea, it features a modern TUI with real-time logs, chat interface, and menu navigation.

### Starting the CLI

```bash
# Build and run
make cli
./bin/hasher

# Or run directly
make run-cli
```

The CLI automatically starts the `hasher-host` orchestrator in the background and displays initialization logs in a blue bordered view while the server starts up.

### Menu Structure

The CLI uses a hierarchical menu system with a **Primary Menu** and an **ASIC Configuration submenu**.

#### Primary Menu

The main menu provides top-level access to HASHER functionality:

| Option | Name | Description |
|--------|------|-------------|
| **1** | Data Pipeline | Run the data processing pipeline (miner → encoder → trainer) |
| **2** | ASIC Config | Open ASIC configuration submenu (Discovery, Probe, Provision, etc.) |
| **3** | Test Chat | Interactive chat interface for hasher-based inference |
| **0** | Quit | Exit the application |

#### ASIC Configuration Submenu

Accessed by selecting **Option 2 (ASIC Config)** from the Primary Menu:

| Option | Name | Description |
|--------|------|-------------|
| **1** | Discovery | Discover ASIC devices on the network via network scanning |
| **2** | Probe | Probe connected ASIC device to gather system information |
| **3** | Protocol | Detect ASIC device communication protocol |
| **4** | Provision | Deploy hasher-server binary to the ASIC device |
| **5** | Troubleshoot | Run comprehensive troubleshooting diagnostics |
| **6** | Configure | Configure hasher inference service settings |
| **7** | Rules | Manage logical validation rules for inference |
| **8** | Test | Test ASIC communication patterns and verify connectivity |
| **9** | Back | Return to Primary Menu |

### Navigation & Controls

**Global Shortcuts:**
- `↑/↓` - Navigate menu items or scroll views
- `Enter` - Select menu item
- `ESC` - Return to previous menu (Primary Menu from submenus, exits from views)
- `Ctrl+C` - Copy selected text to clipboard
- `Mouse` - Scroll views, select text

**Menu Navigation:**
- Select **ASIC Config** (Option 2) from Primary Menu to access device configuration options
- Select **Back** (Option 9) from ASIC Config to return to Primary Menu
- Press `ESC` in any view to return to the appropriate menu level

**Chat View Navigation:**
- `Tab` - Switch between chat and log views
- `Ctrl+V` - Toggle text selection mode
- `PgUp/PgDn` - Scroll by page
- `↑/↓` or `j/k` - Scroll line by line

**Text Selection Mode (Ctrl+V):**
- Click and drag to select text
- `Ctrl+C` - Copy selected text
- Right-click - Copy all text from active view

### Chat Commands

When in chat mode, you can use these commands:

```
/quit           - Exit the application
/menu           - Return to main menu
/help           - Show available commands
/rule add       - Add a logical rule
/rule delete    - Delete a logical rule
/rule list      - List all rules
/status         - Show server and ASIC status
/train          - Train crypto-transformer
```

**Rule Management Examples:**
```
/rule add temperature constraint "Valid range: -40 to 85"
/rule list
/rule delete temperature 0
```

### Data Pipeline Workflow

The Data Pipeline (**Primary Menu → Option 1**) automates the complete data processing workflow:

```
Data Miner → Data Encoder → Data Trainer
```

**Pipeline Stages:**

1. **Data Miner** (⛏️)
   - Processes documents and PDFs
   - Structures raw data for ML pipelines
   - Reaches quota automatically

2. **Data Encoder** (🔐)
   - Performs tokenization
   - Generates embeddings
   - Prepares training data

3. **Data Trainer** (🧠)
   - Trains neural networks
   - Optimizes model weights
   - Completes training cycle

**Pipeline View Features:**
- Real-time progress bar showing overall completion
- Stage status indicators (⏳ Pending / ▶️ Running / ✅ Complete)
- Log output display in blue bordered view
- Automatic progression between stages
- Press `ESC` to return to menu at any time

The pipeline runs each program from `~/.local/share/hasher/bin/` in sequence, waiting for each to reach its quota before starting the next.

### Workflow Example

Typical workflow for setting up a new ASIC device:

```
Primary Menu:
  2. ASIC Config → Opens ASIC Configuration submenu

ASIC Configuration Menu:
  1. Discovery   → Finds ASIC devices on network
  2. Probe       → Gathers device information
  3. Protocol    → Detects communication protocol
  4. Provision   → Deploys hasher-server binary
  8. Test        → Verifies ASIC communication
  9. Back        → Return to Primary Menu

Primary Menu:
  3. Test Chat   → Starts inference with ASIC
```

### Real-time Monitoring

The CLI displays real-time information in the header and footer:

- **Header**: Server status, ASIC device IP
- **Footer**: CPU usage, RAM usage, Go version, device type
- **Blue Box**: Initialization logs, pipeline logs
- **Chat View**: Conversation history and server logs

### CLI Configuration

The CLI stores embedded binaries in:
- `~/.local/share/hasher/bin/` (Linux)
- `~/Library/Application Support/hasher/bin/` (macOS)
- `%LOCALAPPDATA%\hasher\bin\` (Windows)

Environment file (`.env`) is automatically copied to the bin directory if found in the project root.

---

## Programmatic Usage

### Creating and Using the Unified Hasher Engine

```go
package main

import (
    "fmt"
    "knirvhasher/pkg/hashing/transformer"
)

func main() {
    cfg := &transformer.UnifiedConfig{
        VocabSize:    100,
        EmbedDim:     32,
        NumHeads:     4,
        NumLayers:    2,
        ContextLen:   64,
        Hidden1:      16,
        Hidden2:      8,
        OutputSize:   4,
        FFNHiddenDim: 64,
        Activation:   "hash",
        Passes:       21,
        Jitter:       0.01,
    }

    seeds := transformer.BuildDefaultSeedStore(cfg)
    engine := transformer.NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, transformer.ModeTransformer)

    // Transformer mode: token IDs → pooled hidden vector
    hidden := engine.Forward([]int{1, 2, 3})

    // Switch to recursive mode without retraining
    engine.SetMode(transformer.ModeRecursive)
    result, err := engine.Infer([]byte("input data"))
    fmt.Printf("Consensus: %d (confidence: %.2f)\n", result.Consensus.Prediction, result.Consensus.Confidence)

    // Switch to feedforward mode
    engine.SetMode(transformer.ModeFeedforward)
    pred, conf, err := engine.Predict([]byte("input data"))
    fmt.Printf("Prediction: %d (confidence: %.2f)\n", pred, conf)
}
```

### Using the Hardware Router

```go
router := transformer.NewHardwareRouter(hashMethod, transformer.FallbackMixed)
out, err := router.Project(inputVector, seedMatrix)
scores := router.HashToVocab(hiddenState, outputSeed, vocabSize)
```

### Backward Compatibility with Legacy APIs

```go
// Legacy HasherTransformer automatically delegates to UnifiedHasherEngine
ht := transformer.NewHasherTransformer(config, hashMethod)
output := ht.ForwardWrapper(tokenIDs)

// Legacy RecursiveEngine supports mode dispatch
engine, _ := inference.NewRecursiveEngineWithHashMethod(network, hashMethod, 21, 0.01, false)
engine.SetMode(transformer.ModeTransformer)
result, _ := engine.Infer(input)
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

### ASIC Driver Files

- **cmd/driver/hasher-host/main.go**: Orchestrator for managing recursive inference
- **cmd/driver/hasher-server/main.go**: gRPC server running on ASIC device
- **internal/hasher/asic_client.go**: Client for communicating with ASIC server
- **internal/host/deployment.go**: Auto-deployment and management functionality

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
5. **Nonce-Mining LSH**: Uses mining hardware's natural state for deterministic bucket generation

### Architecture Principles

1. **Separation of Concerns**: Orchestrator handles logic, ASIC provides pure computation
2. **Simplicity**: Single-ASIC model minimizes complexity and failure points
3. **Observable Systems**: Exposes detailed metrics for monitoring and tracing
4. **Robustness**: Temporal ensemble provides inherent fault tolerance
5. **Hardware-Native**: Leverages ASIC's mining capabilities rather than fighting them

## ASIC Tools and Diagnostics

### ASIC Monitor with Integrated Diagnostics

The main monitoring tool (`cmd/monitor`) includes comprehensive diagnostic capabilities that run as Phase 0 before monitoring begins.

#### Features
- **Diagnostics**: System, device, process, protocol, and access testing
- **USB Communication**: Direct USB device communication with packet crafting
- **Real-time Monitoring**: Continuous status polling and logging
- **Multiple Output Formats**: Text or JSON diagnostic output
- **Flexible Deployment**: Support for both USB and character device modes

#### Usage Examples

```bash
# Run full diagnostics then monitor
./bin/monitor --diagnostics

# Run specific diagnostic phase only
./bin/monitor --diagnostics --diagnostic-phase system

# Run diagnostics with JSON output
./bin/monitor --diagnostics --json-diagnostics

# Simple device test (one RxStatus and exit)
./bin/monitor --simple-test

# Continuous status logging
./bin/monitor --dump-status --dump-interval 2

# Try character device instead of USB
./bin/monitor --try-char-dev

# Run interrupt endpoints (experimental)
./bin/monitor --try-interrupt
```

#### Build and Deployment

```bash
# Build monitor with USB support (requires CGO)
make build-monitor

# Build diagnostics-only version (MIPS compatible)
make build-monitor-diagnostics

# Deploy full monitor
make deploy-monitor

# Deploy diagnostics-only version
make deploy-monitor-diagnostics
```

#### Diagnostic Phases

1. **System Info**: CPU, memory, kernel, architecture, uptime
2. **Device Info**: USB devices, kernel modules, sysfs interface
3. **Process Info**: CGMiner/BMMiner status, running processes
4. **Protocol Info**: Firmware version, CGMiner config, kernel messages
5. **Device Access Test**: Direct device file access testing

## Integration with ASIC Driver

The `hasher` package integrates with existing asic-driver architecture:

- **gRPC Communication**: Uses ComputeHash, ComputeBatch, and StreamCompute methods
- **Metrics Collection**: Retrieves performance data from GetMetrics API
- **Device Information**: Queries device capabilities via GetDeviceInfo
- **Auto-Deployment**: Automatically deploys hasher-server to ASIC devices
- **Auto-Recovery**: Monitors server logs and handles reboot scenarios
- **Connection Health**: Monitors ASIC connection with automatic reconnection

## Compatibility

- **Protocol**: gRPC over TCP/IP (primary) or direct `/dev/bitmain-asic` access (fallback)
- **Devices**: Antminer S2/S3 with hasher-driver installed
- **Dependencies**: Go 1.16+, standard library only (no external frameworks)
- **Network**: Supports automatic discovery of hasher-server instances

## MATHASHER: Deterministic Mathematical Verification

HASHER now includes the **MATHASHER** extension - a deterministic mathematical verification layer that addresses the "Broken arXiv" problem of LLM hallucination in long-form derivations.

### Key Features

#### 1. Math-Domain Bitmask Specification
The system uses specialized bitmasks for Slot 4 (Grammar) and Slot 10 (Domain) when operating in Math Mode:

| Slot 4 (Math POS) | Role | Description |
|:---:|:---:|:---|
| `0x01` | VARIABLE | Symbolic placeholders (x, y, θ, α) |
| `0x02` | OPERATOR | Arithmetic or logical actions (+, -, ∫, Σ) |
| `0x03` | INTEGER | Constant whole numbers (1, 2, 42) |
| `0x04` | DECIMAL | Floating-point values (3.14, 0.5) |
| `0x05` | FUNCTION | Named operations (sin, log, lim) |
| `0x06` | DELIMITER | Structural boundaries ( (, ), [ ] ) |
| `0x07` | RELATION | Comparative logic (=, <, ≈, ≡) |
| `0x08` | EXPONENT | Power indicators (^2, √) |

| Slot 10 (Domain) | Sub-Domain |
|:---:|:---|
| `0x2000` | Arithmetic |
| `0x2100` | Algebra |
| `0x2200` | Calculus |
| `0x2300` | Statistics |
| `0x2400` | Logic/Set |

#### 2. LaTeX-to-Tensor Mapper
Converts mathematical expressions into the 12-slot neural frame:

```go
mapper := math.NewLaTeXMapper(math.SUB_calculus)
slots := mapper.MapLaTeXToTensor("\\int_0^1 x^2 dx", math.SUB_calculus)
// Returns [12]uint32 with domain locked to 0x2200
```

#### 3. Inference Watchdog
Validates mathematical derivation steps against bitmask constraints:

```go
watchdog := math.NewInferenceWatchdog(math.SUB_algebra)
result := watchdog.ValidateMathStep(0, slots)
// Returns ValidationResult with LogicIntegrity score
```

#### 4. HashNet Plugin API
REST endpoint for mathematical verification:

```json
POST /v1/verify/math
{
  "context": "Integrating f(x) = x^2 from 0 to 3",
  "proposition": "\\int_0^3 x^2 dx = 9",
  "subdomain": "0x2200"
}
```

Response:
```json
{
  "status": "VERIFIED",
  "nonce": "0xABC12345",
  "logic_integrity": 0.99,
  "latency_ms": 2.4
}
```

#### 5. Math-Logic Dataset Miner
Ingest mathematical proofs from Lean, OpenWebMath, or arXiv:

```go
mined, err := api.MineMathDataset("input.json", "output.parquet", math.SUB_calculus)
```

### Usage

```bash
# Verify a mathematical expression
go run cmd/cli/main.go math-verify "x + y = z" --subdomain algebra

# Run tests
go test -v ./pkg/hashing/math/...

# Build with math-logic support
make cli MATH=true
```

### Why MATHASHER?

Traditional LLMs treat math as "likely sequence of characters" - they drift into probabilistic guessing. MATHASHER treats math as a cryptographically signed path where every "=" requires a verified Golden Nonce.

| Feature | Standard LLM | MATHASHER |
|:---:|:---:|:---:|
| Math Execution | Probabilistic | **Cryptographic** |
| Logic Guardrails | Self-correction fails | **Hard-coded Bitmasks** |
| Verification | Black box | **Golden Nonce Proof** |

### Formal Proof Verification (Neuro-Symbolic Enhancement)

MATHASHER has been enhanced with a **neuro-symbolic formal proof system** that separates structural prechecking from formal theorem verification. The system now uses a pinned formal proof checker (initially Lean) as the sole authority for declaring a theorem formally verified.

#### Verification Status Model

The public status model distinguishes four independent claims that must not be conflated:

| Status | Authority | Meaning |
|:---|:---|:---|
| `STRUCTURALLY_VALID` | MATHASHER precheck | Input conforms to syntax/domain constraints |
| `PROOF_PENDING` | API orchestration | Valid candidate accepted for asynchronous checking |
| `FORMALLY_VERIFIED` | Lean kernel | Complete proof term type-checks against theorem |
| `FORMALLY_REJECTED` | Lean kernel | Proof failed formal checking |
| `CHECKER_UNAVAILABLE` | API orchestration | Formal checker not reachable or misconfigured |
| `HARDWARE_ATTESTED` | ASIC PoW | Optional hardware attestation after formal verification |

**Key principle:** Only a successful formal-checker receipt can produce `FORMALLY_VERIFIED`. The MATHASHER nonce is deterministic input-derived metadata; it is explicitly NOT a proof term or witness of theorem validity.

#### Proof Asset Model

Proof assets are content-addressed artifacts that bind a theorem source, proof source, and environment identity:

```go
asset := &proofasset.ProofAsset{
    SchemaVersion:        1,
    ProofSystem:          proofasset.ProofSystemLean,
    ToolchainDigest:      "lean-4.3.0",
    DependencyLockDigest: "lean-lock-abc123",
    TheoremSource:        []byte("theorem t : True := by trivial"),
    ProofSource:          []byte("theorem t : True := by trivial"),
    Imports: []proofasset.ArtifactRef{
        {Name: "Mathlib.Data.Real.Basic", Digest: "sha256:def456"},
    },
}
assetID, _ := proofasset.ComputeProofAssetID(asset)
```

#### Formal Verification API

When `--formal-verifier` is enabled, the following endpoints are available:

```bash
# Submit a proof asset for formal verification
curl -X POST http://localhost:8080/v1/verify/proof \
  -H "Content-Type: application/json" \
  -d '{"canonical_proof_asset": {...}, "request_hardware_attestation": false}'

# Poll proof status
curl http://localhost:8080/v1/verify/proof/{proof_asset_id}

# Retrieve stored proof asset
curl http://localhost:8080/v1/verify/proof/{proof_asset_id}/asset
```

#### Proof Ledger

Verified proofs are recorded in an append-only `proof_writes.jsonl` ledger. The ledger only accepts `FORMALLY_VERIFIED` entries when the receipt validates against the exact stored artifact IDs.

#### Running with Formal Verification

```bash
# Start hasher-host with MATHASHER schema and formal verifier enabled
./bin/hasher-host \
  --schema pipeline/2_DATA_ENCODER/config/math_schema.yaml \
  --formal-verifier \
  --lean-binary lean \
  --lean-max-seconds 15 \
  --frames-dir /data/hasher/frames
```

#### Schema Configuration

The `math_schema.yaml` now includes a `formal_verification` policy section:

```yaml
formal_verification:
  default_proof_system: lean
  allowed_imports:
    - Mathlib.Algebra.Group.Basic
    - Mathlib.Data.Real.Basic
  max_source_bytes: 65536
  max_checker_seconds: 15
```

This is an admission-control policy only; it does not represent proof rules and does not replace the proof assistant environment lock.

#### Trust Boundary

The following components are **untrusted** with respect to theorem correctness:
- LLMs, HASHER transformer modes, embeddings, retrieval, and evolutionary search
- Candidate parsers, proof repair logic, and canonicalization implementation
- Proof artifact ledger transport and ASIC hardware claims

The **trusted correctness boundary** is the selected proof assistant's kernel plus the pinned trusted environment it requires.

---

## Future Enhancements

1. **Z3 Integration**: Full integration with Z3 theorem prover for advanced logical reasoning
2. **Dynamic Learning**: Online learning from ground truth comparisons
3. **Adaptive Pass Count**: Adjust number of passes based on confidence levels
4. **Model Pruning**: Optimize network structure for specific tasks
5. **Header Verification**: Flash modified `cgminer` to verify Go-built headers are accepted by BM1382
6. **Nonce Stability Test**: Run same vector through multiple Antminers to verify consistent "First Nonce"

---

### ⚖️ License

**Hasher** is licensed under the [GNU General Public License v3.0](https://github.com/guiperry/HASHER/blob/main/LICENSE).

#### Why GPLv3?

We believe that the transformation of "obsolete" hardware into cutting-edge AI primitives should benefit everyone. The GPLv3 ensures that:

* **Transparency:** Anyone can inspect, modify, and learn from the code that bridges SHA-256 ASICs with neural network operations.
* **Reciprocity:** Any derivatives or improvements made to the Hasher core must also be released under the same open-source license.
* **Patent Protection:** The license provides an explicit grant of patent rights from contributors to users, protecting the community as we explore novel computational methods.

---

## Authors

- Guillermo Perry
- Hasher Architecture Team
- Claude Code, Gemini, & Deep Seek assisted with this implementation.
