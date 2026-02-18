# KNIRV-NEXUS: Deterministic Validation & Active Memory Fabric

KNIRV-NEXUS is the central orchestration and memory layer of the KNIRV Network. It provides a high-performance, secure, and semantic environment for AI agents to operate within a **Deterministic Validation Environment (DVE)**. By merging five specialized Go submodules, NEXUS delivers a unified "Markdown Fabric" for reasoning persistence and verifiable solution execution.

## 🏗️ Technical Architecture

KNIRV-NEXUS leverages a hybrid architecture combining high-throughput memory streaming with post-quantum secured persistence.

### 1. The Markdown Fabric (Active Memory Layer)
NEXUS transforms all intelligence artifacts into human-readable, machine-executable **Markdown (.md)** files.
*   **Reasoning Traces (KNIRVGRAPH)**: Stored as `.md` context records, documenting the agent's thought process through Network Resolution Vectors (NRV).
*   **Solution Vault (KNIRVCHAIN)**: Stores ErrorNodes and SolutionNodes. Solutions contain executable code blocks (interpreted Go/Shell) secured by PQC signatures.
*   **Persistence (KNIRVBASE)**: All `.md` nodes are transparently encrypted at rest using **Kyber-768** and signed with **Dilithium-3**.

### 2. Living Memory Projection (Apache Arrow Flight)
While data is persisted as Markdown, it is projected in real-time into the **Memory Fabric** using **Apache Arrow**.
*   **Sub-millisecond Streaming**: Context records are saved into Arrow buffers for immediate delivery to agents.
*   **Binary Portability**: Cross-language memory access for agents written in Python, Rust, or Go.
*   **Tick Data Streaming**: Phase 6 implements high-performance financial tick data streaming with Arrow IPC format.

### 3. Runtime Security & Verification (eBPF Guardian)
NEXUS implements a hardware-assisted security model to perform **Key Neural Intelligence Reasoning Validation**.
*   **Syscall Monitoring**: Uses eBPF to trace every system call made by an agent process.
*   **Intent Correlation**: Correlates the agent's stated "Intent" (from its reasoning trace) with its "Observed Action" (from eBPF).
*   **Virtual Containers**: Provides isolated namespaces and cgroup-based resource limits for agent execution.

### 4. Secure Transport (KNIRVROUTER)
Integrated **TURN Server** logic facilitates reliable P2P synchronization of the Markdown Fabric across restrictive NAT environments.

---

## 🚀 FinTech Validator: Financial AI Agent Validation

KNIRV-NEXUS includes a comprehensive **FinTech Validator** for deterministic validation of financial AI agents. This system transforms AI failures into collective knowledge through interconnected sovereign layers.

### Phase 1: Evidence Packs & Financial Ontologies
- **Evidence Packs**: Immutable audit trails capturing validation results, compliance checks, and execution traces
- **Financial Ontologies**: Regulatory frameworks (AML, KYC, SEC, Basel III) as machine-readable rules
- **PQC Signing**: All evidence packs signed with post-quantum cryptography (ML-DSA-65)

### Phase 2: Scenario Injection & Compliance Engine
- **Regulatory Scenarios**: Pre-built test scenarios for regulatory compliance testing
- **Compliance Engine**: Automated validation against financial regulations
- **Certificate of Correctness**: PQC-signed certificates attesting to agent validation results

### Phase 3: Deterministic Replay (eBPF Trajectory Capture)
- **eBPF Trajectory Capture**: Syscall-level execution tracing of financial AI agents
- **Execution Replay**: Deterministic replay of captured trajectories for verification
- **Trajectory Comparison**: Compare execution runs to detect non-deterministic behavior

### Phase 4: NRV Financial Semantic Engine & Fidelity Scoring
- **NRV (Network Resolution Vector) Traces**: Semantic reasoning traces capturing agent decision-making
- **Fidelity Scoring**: Quantify how closely agent reasoning matches expected financial logic
- **Semantic Distance**: Calculate regulatory alignment distance
- **Risk Detection**: KYC bypass detection, position limit violation monitoring

### Phase 5: Financial Compliance Dashboard (Frontend)
- **React/TypeScript Dashboard**: Real-time visualization of validation results
- **Fidelity Score Display**: Risk assessment with confidence indicators
- **Compliance Status**: Multi-regulation compliance tracking (AML, KYC, SEC, Basel)
- **Trajectory Visualization**: Execution path visualization
- **Evidence Pack Management**: View, export, and verify evidence bundles

### Phase 6: Arrow Flight Tick Data Streaming
- **Apache Arrow IPC**: High-performance binary data format for financial tick data
- **Tick Data Streams**: Real-time streaming of trade/quote data with buffering
- **Bar Data Aggregation**: OHLCV bar generation from tick streams
- **Query & Filter**: Time-range, price-range, and volume-based filtering
- **Export**: Export tick data as Arrow IPC files

---

## 🛠️ Technical Stack

-   **Backend**: Go 1.24+ (Strict concurrency, ported PQC primitives).
-   **Frontend**: Next.js 15 (App Router, Tailwind CSS 4, shadcn/ui).
-   **Security**: eBPF (CO-RE), Linux Security Modules (LSM), TEE Support (SGX/TDX).
-   **Protocols**: Model Context Protocol (MCP), Arrow Flight, TURN (STUN).
-   **Storage**: BuntDB (Metadata) + Markdown Fabric (Encrypted Content).
-   **Data Format**: Apache Arrow IPC for high-performance streaming.

---

## 📡 API Endpoints

### FinTech Validator API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/fintech/status` | GET | Service status |
| `/api/fintech/validate` | POST | Run comprehensive validation |
| `/api/fintech/evidence` | GET | List evidence packs |
| `/api/fintech/evidence/{id}` | GET | Get evidence pack |
| `/api/fintech/evidence/{id}/export` | GET | Export evidence bundle |
| `/api/fintech/ontologies` | GET | List financial ontologies |
| `/api/fintech/scenarios` | GET | List regulatory scenarios |
| `/api/fintech/scenarios/validate` | POST | Run scenario validation |
| `/api/fintech/certificates/issue` | POST | Issue certificate of correctness |
| `/api/fintech/trajectories` | GET | List execution trajectories |
| `/api/fintech/trajectories/{id}/replay` | POST | Replay trajectory |
| `/api/fintech/nrv/traces` | GET | List NRV reasoning traces |
| `/api/fintech/nrv/traces/{id}/score` | POST | Calculate fidelity score |
| `/api/fintech/nrv/traces/{id}/distance` | POST | Calculate semantic distance |
| `/api/fintech/nrv/traces/{id}/detect/kyc-bypass` | GET | Detect KYC bypass |
| `/api/fintech/ticks/streams` | POST | Create tick data stream |
| `/api/fintech/ticks/{symbol}` | GET | Get ticks for symbol |
| `/api/fintech/ticks/{symbol}/add` | POST | Add tick to stream |
| `/api/fintech/ticks/query` | POST | Query ticks with filters |
| `/api/fintech/ticks/export` | GET | Export ticks as Arrow IPC |

### Model Context Protocol (MCP)
- `POST /api/memory/mcp/store`: Encrypts and registers an interaction trace.
- `POST /api/memory/mcp/execute/{id}`: Loads and runs a solution node from the PQC Vault.

### Arrow Flight Memory Stream
- **Endpoint**: `:50051`
- **Schema**: `timestamp (int64)`, `agent_id (string)`, `intent (string)`, `observed_action (string)`, `verified (bool)`.

---

## 🧪 Build & Deployment

### Environment Requirements
- **OS**: Linux (Kernel 5.10+ required for eBPF features). Kali Linux or Ubuntu 22.04+ recommended.
- **Tools**: Go 1.24, Node.js 18+, `build-essential`, `libseccomp-dev`.

### Unified Build Process
```bash
# 1. Install frontend dependencies
npm install

# 2. Build the Next.js frontend
npm run build

# 3. Build the Go backend
cd backend
go mod tidy
go build -o ../bin/backend_server ./main.go
cd ..

# 4. Compile the Unified Binary (Main Wrapper)
go build -o knirv-nexus main.go
```

### Configuration

Enable FinTech Validator phases in configuration:

```go
config := &fintech_validator.Config{
    // Phase 1: Evidence Packs & Ontologies
    EnableAMLChecks:       true,
    EnableKYCCheks:        true,
    EnableSECCheks:        true,
    EnableBaselCheks:      true,
    
    // Phase 2: Scenarios & Certification
    EnableScenarioTesting: true,
    EnableCertification:   true,
    
    // Phase 3: Trajectory Capture
    EnableTrajectoryCapture: true,
    EnableReplayEngine:      true,
    
    // Phase 4: NRV & Fidelity
    EnableNRVTracing:       true,
    EnableFidelityScoring:  true,
    
    // Phase 6: Tick Data Streaming
    EnableTickDataStreaming: true,
    TickDataServerPort:     "8819",
    MaxStreamBufferSize:    1000,
}
```

---

## 🔒 Security Constraints

- **Hardware Enclaves**: TEE features (SGX/TDX) require specific hardware support and BIOS configuration.
- **eBPF Capabilities**: Requires `CAP_SYS_ADMIN` or `CAP_BPF` privileges for program attachment.
- **PQC Keys**: Master key rotation requires manual re-encryption of the Markdown Fabric.

---

## 📄 License
Copyright 2026 KNIRV-NEXUS. Distributed under the **GPL-3.0-or-later** license.

---

**KNIRV-NEXUS**: *Where semantic reasoning meets verifiable execution.* 🚀
