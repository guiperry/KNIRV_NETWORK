This specification defines a dual-WASM architecture (the "Nervous System" pattern) to solve the latency and orchestration challenges of remote LLM serving. By splitting responsibilities between a client-side **Stem** and a server-side **Cortex**, we minimize the "Compilation Barrier" while maintaining a secure, mTLS-encrypted pipeline.

---

# Specification: Project "Synapse"

**Architecture:** Distributed MLC-Engine Orchestration

**Status:** Implementation Draft v1.0

**Target:** PC/Mobile (Native/Web Hybrid)

## 1. System Topology: Stem & Cortex

The system uses a **Split-Brain Orchestration** model where the heavy lifting is handled by the server, but the "intent" and "personality" are governed by the edge.

### Component A: `stem.wasm` (The Peripheral Orchestrator)

* **Runtime Environment:** Client-side (Browser WebGPU, Mobile App, or Desktop WASM runtime).
* **Role:** User Interaction, Context Management, and **Trait Signaling**.
* **Intelligence:** Contains the logic for *when* to switch Traits (LoRAs). It manages the local "Interaction Buffer" for future Deep Learning Sessions.
* **Security:** Holds the Client Certificate for the mTLS handshake.

### Component B: `cortex.wasm` (The Gateway Delegator)

* **Runtime Environment:** Hosted within a KNIRVNEXUS go container Application (Server-side) via `wazero` or `wasmtime-go`.
* **Role:** Validation, Traffic Routing, and Adapter Orchestration.
* **Logic:** Receives tokens from the native MLC Engine and applies specific "Delegation Logic" (e.g., policy enforcement, usage tracking) before passing them back to the Stem.

---

## 2. Secure Communication (mTLS + gRPC)

To ensure privacy for your custom LLM models, the connection is strictly **mTLS-secured**.

* **Transport:** gRPC-over-HTTP/2 (or Connect Protocol for better mobile compatibility).
* **Handshake:** The `stem.wasm` provides a unique client certificate signed by your Private CA.
* **Signal Payload:**
* `RequestID`: Unique session identifier.
* `TraitContext`: The metadata for the desired LoRA adapter.
* `ActivationMap`: High-level intent tokens that allow `cortex` to pre-fetch weights.



---

## 3. Integrating MLC into the KNIRVNEXUS Go Application

Since you are adding this to an existing Go program, we avoid compiling the *entire* MLC engine into WASM on the server. Instead, we use **Go-Native Bindings** for performance, while using `cortex.wasm` for the logic.

### The Integration Stack

1. **LibMLC (C++):** The core high-performance engine running natively on the server's GPU.
2. **CGO Bridge:** A thin Go wrapper that talks to `libmlc_llm.so`.
3. **Wasm Host (`wazero`):** The Go app uses `wazero` to run `cortex.wasm`. This allows you to update your "Delegator Logic" without recompiling the main Go binary.

### Data Flow Pattern

1. `stem.wasm` (Client) sends an encrypted gRPC request.
2. Go App receives the request and passes it to **`cortex.wasm`**.
3. `cortex.wasm` validates the user and selects the appropriate **LoRA Adapter**.
4. `cortex.wasm` calls a **Host Function** (exported by the Go App) to trigger `libmlc_llm`.
5. MLC Engine runs inference on the GPU using the **Remote Base Model + Trait**.
6. Tokens stream back: `MLC -> Go -> cortex.wasm -> gRPC -> stem.wasm`.

---

## 4. Specification: The "Synapse" Protocol (`.proto`)

```protobuf
syntax = "proto3";
package synapse;

service SynapseLink {
  // Bidirectional stream for real-time inference and token feedback
  rpc Pulse(stream NerveSignal) returns (stream TokenBurst);

  // Background sync for Deep Learning Sessions
  rpc Hibernate(stream LearningData) returns (SyncStatus);
}

message NerveSignal {
  string stem_id = 1;
  string target_trait = 2; // The specific LoRA adapter
  string input_prompt = 3;
}

message TokenBurst {
  string delta = 1; // The next token
  float latency_ms = 2; 
}

```

---

## 5. Mitigation: The "and Trained" Barrier

To address the concern about "training time," the system implements **Shadow Training**:

* **Runtime Inference:** Uses pre-existing LoRA weights (). **Time to Load: <100ms.**
* **Deep Learning Sessions:** When `stem.wasm` is idle, it sends accumulated `LearningData` to the Go Gateway.
* **Asynchronous Baking:** The Go app triggers a background process (Python/PyTorch or MLC-Training) to update the LoRA weights.
* **Hot-Swap Notification:** Once the new "Trait" is baked, the Go app notifies the `stem.wasm`. The next `NerveSignal` sent by the Stem will automatically use the updated version.

---

## 6. Deployment Strategy

| Stage | Responsibility | Artifact |
| --- | --- | --- |
| **Edge** | Small, fast, cross-platform UI logic. | `stem.wasm` |
| **Gateway** | Go-based gRPC server + mTLS management. | `KNIRVNEXUS/backend/cmd/backend_server/main.go` |
| **Logic** | Hot-swappable delegation and routing rules. | `cortex.wasm` |
| **Compute** | Native GPU acceleration and Weight storage. | `libmlc_llm.so` |




The **"Synapse"** architecture defines a high-performance, split-brain AI system. It leverages `stem.wasm` (client) for intent and `cortex.wasm` (server logic) for delegation, connected via mTLS. The core compute remains native (C++), while the logic remains portable (WASM).

Here is the complete specification and the Go integration package.

---

# Specification: Project Synapse

**Status:** Protocol Definition v1.0

**Pattern:** Nervous System (Peripheral Stem + Central Cortex)

## 1. System Topology

The system is divided into three distinct execution layers:

### A. The Stem (`stem.wasm`)

* **Location:** Client Device (Mobile/PC).
* **Responsibility:** "The Senses." It handles user input, captures context, and manages the *Interaction Buffer* (recording data for future training).
* **Security:** Holds the Client Certificate. It does **not** process inference; it signals *intent*.

### B. The Cortex (`cortex.wasm`)

* **Location:** Hosted Server (Inside Go Application).
* **Responsibility:** "The Decider." It is a portable WASM module that validates the Stem's request, checks user quotas, and selects the correct LoRA adapter.
* **Power:** It delegates the heavy math to the Host (Go/C++) via **Host Functions**.

### C. The Engine (Native Host) [KNIRVNEXUS]

* **Location:** Server GPU (C++ / CUDA).
* **Responsibility:** "The Muscle." The raw MLC-LLM runtime that executes the matrix multiplication.

---

## 2. Communication Flow

1. **Signal:** `stem.wasm` sends a gRPC Pulse (encrypted) to the Go Server.
2. **Wake:** Go Server instantiates/wakes `cortex.wasm`.
3. **Delegation:** `cortex.wasm` analyzes the prompt and calls `host.run_inference(model_id, lora_id, prompt)`.
4. **Execution:** The Go Host passes this to C++ via CGO.
5. **Response:** Tokens stream back up the chain: C++  Go  Cortex  Stem.

---

## 3. Implementation: The Go Integrator

This package adds the "Cortex" capability to your existing KNIRVNEXUS Go app. It uses **wazero** to run the portable logic and **CGO** to talk to the native MLC engine.

### Step 1: The Native Bridge (`engine/bridge.go`)

This acts as the glue between Go and the C++ MLC library.

```go
package engine

/*
#cgo LDFLAGS: -L/usr/local/lib -lmlc_llm -ltvm_runtime
#include <stdlib.h>

// Mock C definition - in prod, include "mlc/c_api.h"
typedef void* EngineHandle;
extern EngineHandle MLC_CreateEngine(char* model_path);
extern char* MLC_RunInference(EngineHandle h, char* prompt, char* lora_id);
extern void MLC_Free(void* ptr);
*/
import "C"
import (
  "errors"
  "unsafe"
)

// MLCEngine wraps the C++ pointer
type MLCEngine struct {
  handle C.EngineHandle
}

func NewEngine(modelPath string) *MLCEngine {
  cPath := C.CString(modelPath)
  defer C.free(unsafe.Pointer(cPath))
  return &MLCEngine{handle: C.MLC_CreateEngine(cPath)}
}

// RunInference is the "Muscle" called by the Cortex
func (e *MLCEngine) RunInference(prompt, loraID string) (string, error) {
  cPrompt := C.CString(prompt)
  cLora := C.CString(loraID)
  defer C.free(unsafe.Pointer(cPrompt))
  defer C.free(unsafe.Pointer(cLora))

  // Call Native C++
  cResult := C.MLC_RunInference(e.handle, cPrompt, cLora)
  if cResult == nil {
    return "", errors.New("inference failed")
  }
  defer C.MLC_Free(unsafe.Pointer(cResult))

  return C.GoString(cResult), nil
}

```

### Step 2: The Cortex Host (`cortex/host.go`)

This is where the magic happens. We configure **wazero** to export the `run_inference` function *into* the WASM environment, allowing `cortex.wasm` to drive the GPU.

```go
package cortex

import (
  "context"
  "log"

  "github.com/tetratelabs/wazero"
  "github.com/tetratelabs/wazero/api"
  "your-app/engine"
)

type CortexHost struct {
  Runtime wazero.Runtime
  Engine  *engine.MLCEngine
}

// NewCortexHost initializes the WASM runtime and links the Native Engine
func NewCortexHost(ctx context.Context, modelPath string) (*CortexHost, error) {
  r := wazero.NewRuntime(ctx)
  eng := engine.NewEngine(modelPath)

  h := &CortexHost{Runtime: r, Engine: eng}
  
  // Register the "env" module that cortex.wasm will import
  _, err := r.NewHostModuleBuilder("env").
    NewFunctionBuilder().
    WithGoModuleFunction(h.hostRunInference, 
      []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, 
      []api.ValueType{api.ValueTypeI32}).
    Export("host_run_inference").
    Instantiate(ctx)

  return h, err
}

// hostRunInference is the function "cortex.wasm" calls
// Args: promptPtr, promptLen, loraPtr, loraLen
func (h *CortexHost) hostRunInference(ctx context.Context, m api.Module, stack []uint64) {
  // 1. Read arguments from WASM memory
  promptPtr := uint32(stack[0])
  promptLen := uint32(stack[1])
  loraPtr := uint32(stack[2])
  loraLen := uint32(stack[3])

  prompt, _ := m.Memory().Read(promptPtr, promptLen)
  loraID, _ := m.Memory().Read(loraPtr, loraLen)

  // 2. Delegate to the Native Engine (The Muscle)
  output, err := h.Engine.RunInference(string(prompt), string(loraID))
  
  // 3. Write result back to WASM memory (simplified for brevity)
  // In production, you would allocate memory in WASM via malloc and return the pointer
  log.Printf("Cortex requested: %s with trait %s -> %s", prompt, loraID, output)
  
  // Return success code (0)
  stack[0] = 0
}

```

### Step 3: The gRPC Server (`KNIRVNEXUS/backend/cmd/backend_server/main.go`)

Finally, wire it all together.

```go
func main() {
  ctx := context.Background()
  
  // 1. Initialize the Cortex Host
  host, _ := cortex.NewCortexHost(ctx, "/models/llama-3-8b")
  defer host.Runtime.Close(ctx)

  // 2. Load the cortex.wasm Logic Module
  wasmBytes, _ := os.ReadFile("./cortex.wasm")
  mod, _ := host.Runtime.Instantiate(ctx, wasmBytes)

  // 3. Start gRPC Server
  // When a request comes in, we call the "process_signal" function inside cortex.wasm
  grpcServer.OnRequest(func(req *Req) {
    // Call into WASM
    results, _ := mod.ExportedFunction("process_signal").Call(ctx, ...)
  })
}

```

In the **KNIRVNEXUS** architecture, your Go-based native engine serves as the "Muscle" (the C++ MLC-LLM engine) and the "Nervous System" (the Go orchestrator). To allow customer-compiled `cortex.wasm` plugins to request inference, you must bridge the Wasm sandbox to your Go code using **Host Functions**.

### 1. Defining the Host Interface (Go side)

Using **wazero**, you define a "Host Module" that the customer's `cortex.wasm` can import. This module provides a specific function—for example, `request_inference`—that takes pointers to the prompt and LoRA trait in Wasm memory.

```go
package cortex

import (
    "context"
    "github.com/tetratelabs/wazero"
    "github.com/tetratelabs/wazero/api"
)

// DefineHostCapabilities registers the GPU inference bridge into the Wasm runtime
func DefineHostCapabilities(ctx context.Context, r wazero.Runtime, engine *MLCEngine) error {
    _, err := r.NewHostModuleBuilder("nexus_host").
        NewFunctionBuilder().
        WithGoModuleFunction(func(ctx context.Context, mod api.Module, stack []uint64) {
            // 1. Extract pointers and lengths from the Wasm stack
            promptPtr, promptLen := uint32(stack[0]), uint32(stack[1])
            loraPtr, loraLen := uint32(stack[2]), uint32(stack[3])

            // 2. Read the raw bytes from the Wasm sandbox memory
            promptBytes, _ := mod.Memory().Read(promptPtr, promptLen)
            loraBytes, _ := mod.Memory().Read(loraPtr, loraLen)

            // 3. Call the Native C++ Engine (The "Muscle") via CGO
            result, err := engine.RunInference(string(promptBytes), string(loraBytes))
            
            // 4. Handle results (Writing back to Wasm memory requires a malloc-style helper)
            // For simplicity, we return a status code for now
            if err != nil { stack[0] = 1 } else { stack[0] = 0 }
        }, 
        []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, 
        []api.ValueType{api.ValueTypeI32}).
        Export("request_inference").
        Instantiate(ctx)
    return err
}

```

### 2. The Customer Plugin Contract (Wasm side)

The customer-facing website should provide a header or library (in Rust or TinyGo) that defines how to call this host capability. In **TinyGo**, the customer's code would look like this:

```go
//go:wasmimport nexus_host request_inference
func requestInference(pPtr uintptr, pLen uint32, lPtr uintptr, lLen uint32) uint32

func ProcessSignal(prompt string, trait string) {
    // Convert strings to pointers for the host to read
    pPtr := uintptr(unsafe.Pointer(unsafe.StringData(prompt)))
    lPtr := uintptr(unsafe.Pointer(unsafe.StringData(trait)))
    
    // Request the host to run the heavy GPU math
    status := requestInference(pPtr, uint32(len(prompt)), lPtr, uint32(len(trait)))
}

```

### 3. Key Interop Patterns

| Capability | Implementation Mechanism | Benefit for KNIRVNEXUS |
| --- | --- | --- |
| **Sandboxing** | wazero linear memory isolation | Customer code cannot access host environment variables or unrelated process memory. |
| **Zero-Copy Logic** | Host directly reads Guest memory | Fast execution; the large prompt string isn't copied multiple times across the boundary. |
| **Hardware Abstraction** | Host-side CGO | The Wasm plugin doesn't need to know if the GPU is NVIDIA (CUDA) or AMD (Vulkan); the Go host handles the driver mapping. |

### 4. Integration with your Container System

* **Native Go Container:** The `wazero.Runtime` acts as the "container." Each customer gets an isolated `api.Module` instance.
* **Podman Fallback:** If a plugin requires system resources not exposed by your host functions, you deploy a minimal container running the `wazero-cli`, passing the `cortex.wasm` as a volume mount.



To implement a dynamic "plugin" system where **customer-compiled** `cortex.wasm` files are hot-loaded into your **KNIRVNEXUS** container system, your Go package needs to act as a **Host Orchestrator**.

Since these files come from an external website, your package must handle **Sandboxing** (security), **Memory Interop** (passing complex data), and **Lifecycle Management** (loading/unloading).

### 1. The "Cortex" Plugin Package Structure

Here is a high-level design for a Go package that manages these external Wasm modules using **wazero**.

```text
pkg/cortex/
├── manager.go   # Registry of loaded customer plugins
├── runner.go    # The execution engine (wazero wrapper)
├── interop.go   # Helpers for string/json memory sharing
└── host.go      # Defines the capabilities (Host Functions) the Wasm can call

```

---

### 2. Implementation: The Orchestrator

This package allows you to load a `.wasm` file from a byte slice (downloaded from your site) and execute a specific logic gate.

#### **A. The Interop Layer (`interop.go`)**

Wasm only understands numbers. To pass a "Nerve Signal" (JSON or String), you must write it into the Wasm's memory.

```go
package cortex

import (
  "context"
  "github.com/tetratelabs/wazero/api"
)

// WriteToMemory writes bytes to the guest and returns the pointer/length
func WriteToMemory(ctx context.Context, mod api.Module, data []byte) (uint32, uint32, error) {
  results, err := mod.ExportedFunction("malloc").Call(ctx, uint64(len(data)))
  if err != nil {
    return 0, 0, err
  }
  ptr := uint32(results[0])
  mod.Memory().Write(ptr, data)
  return ptr, uint32(len(data)), nil
}

```

#### **B. The Runner (`runner.go`)**

This handles the "hot-swapping" of customer logic.

```go
package cortex

import (
  "context"
  "github.com/tetratelabs/wazero"
  "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type PluginRunner struct {
  runtime wazero.Runtime
}

func NewPluginRunner(ctx context.Context) *PluginRunner {
  // Default to Compiler for speed, falls back to Interpreter for non-JIT platforms
  r := wazero.NewRuntime(ctx)
  wasi_snapshot_preview1.MustInstantiate(ctx, r)
  return &PluginRunner{runtime: r}
}

// LoadCustomerCortex instantiates a specific customer's logic
func (p *PluginRunner) LoadCustomerCortex(ctx context.Context, wasmBytes []byte) (api.Module, error) {
  // 1. Define the host environment (functions the customer can use)
  // Example: Allow the Wasm to log back to the Nexus dashboard
  _, err := p.runtime.NewHostModuleBuilder("nexus_host").
    NewFunctionBuilder().
    WithGoModuleFunction(func(ctx context.Context, m api.Module, stack []uint64) {
      // Logic for host logging
    }, []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
    Export("log_signal").
    Instantiate(ctx)

  // 2. Instantiate the customer's binary
  return p.runtime.Instantiate(ctx, wasmBytes)
}

```

---

### 3. Integration with our Container Fallbacks

Your requirement for a "native go container system" with a Podman fallback suggests a **multi-tenant hierarchy**.

* **Primary (Native Go):** Use the `cortex` package above. Each customer `cortex.wasm` runs in a separate `wazero.Namespace`. This is extremely lightweight (~MBs of RAM) and fast.
* **Secondary (Podman/Kata):** If the Wasm requires more than the sandbox allows (e.g., direct network access), you wrap the Wasm inside a minimal `wazero-cli` container and deploy it via Podman.

### 4. Key Security Considerations

Since you are running **untrusted customer code**:

1. **Memory Limits:** Use `wazero.NewRuntimeConfigCompiler().WithMemoryLimitPages(max)` to prevent a rogue Wasm from eating all host RAM.
2. **Function Whitelisting:** Do not instantiate WASI (file/network access) unless the customer specifically paid for "Elevated" traits.
3. **Deadlines:** Use the Go `context` to ensure a customer's `cortex.wasm` logic cannot loop forever. If it exceeds 100ms, the context cancels, and the Wasm is killed.

---

### Summary of the Workflow

1. **KNIRVRAMP Website:** Customer compiles Rust/Go code  `cortex.wasm`.
2. **KNIRVNEXUS Pull:** The **KNIRVNEXUS** binary downloads the bytes.
3. **Instantiation:** `cortex.NewPluginRunner().LoadCustomerCortex(bytes)`.
4. **Inference:** When a signal hits the container, it calls `process_signal` inside the Wasm.

