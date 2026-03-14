# KNIRV-SERVER Gap Analysis: Frontend to Backend Functionality

This document outlines the current discrepancies and missing functionality between the KNIRV-SERVER frontend dashboards and the backend implementation.

## 1. Problem: Terminal and SSH Integration
*   **Current State:** Both the `NetworkAccessModal` (NAP) and `ConsolePanel` (CDE-Panel) feature terminal interfaces. However, input handling is either localized (echoed back) or simulated via `setTimeout`.
*   **Missing Functionality:** 
    *   No real-time bi-directional communication (WebSockets) for terminal sessions.
    *   Missing backend SSH gateway to proxy commands from the web terminal to the TEE containers.
    *   Terminal output is largely hardcoded or limited to historical log fetching.

    ## 1.2. Solution: KNIRVCLI Integration
*   **Current State:** KNIRVCLI exists as a standalone command-line tool in `~/KNIRV_NETWORK/KNIRVCLI/` with the following capabilities:
    *   Multi-command CLI with subcommands for wallet, network, agent, MCP, economics, forge, graphchain, system, and WASM operations
    *   API clients for KNIRVGATEWAY, KNIRVORACLE, KNIRVROOT, KNIRVCHAIN, and KNIRVGRAPH
    *   XION wallet management and NRN token management
    *   MCP server and procedure management
    *   TUI (Terminal User Interface) for interactive CLI usage
    *   Event bus and WebSocket client for real-time updates
*   **Integration Requirement:**
    *   Seamlessly integrate KNIRVCLI as the backend communication layer for KNIRVNEXUS
    *   Replace direct REST/HTTP calls from frontend to backend services with KNIRVCLI's unified API client abstraction
    *   Leverage KNIRVCLI's existing clients (knirvgateway_client, knirvoracled_client, knirvroot_client, knirvgraph_client, knirvnexus_client) for cross-component communication
    *   Utilize KNIRVCLI's wallet and token management for FinTech functionality within the Nexus
    *   Integrate KNIRVCLI's event bus and WebSocket manager for real-time streaming to frontend
    *   Expose KNIRVCLI's MCP procedure execution capabilities through KNIRVNEXUS for capability invocation
    *   Maintain KNIRVCLI as a standalone CLI while embedding its core library in KNIRVNEXUS backend

## 2. Dynamic Policy Enforcement
*   **Current State:** The `PolicyEditor` provides a sophisticated UI for configuring network whitelists, eBPF forensics, and TEE attestation.
*   **Missing Functionality:**
    *   The "Commit Policy to Blockchain" button is a placeholder.
    *   Backend lacks endpoints to receive, validate, and persist these policy updates.
    *   No mechanism to propagate updated policies to active DVE nodes/containers in real-time.

## 3. DVE Monitoring and Task Tracking
*   **Current State:** `MonitorPanel` attempts to fetch per-node tasks and metrics using `/api/dve/${nodeId}/tasks` and `/api/dve/${nodeId}/metrics`.
*   **Missing Functionality:**
    *   The backend `DVEHandlers` and `DVEManager` do not implement these per-node endpoints.
    *   Global metrics exist, but granular task-level tracking for individual DVE nodes is not exposed via the REST API.
    *   Task history in the UI is currently populated by demo data when live fetch fails.

## 4. Workflow and Solver Automation
*   **Current State:** The `NAP` includes workflow templates (e.g., "Validation Setup", "Fabric Deployment") and a DVE Solver.
*   **Missing Functionality:**
    *   Workflow execution is simulated with `setTimeout` and hardcoded console output.
    *   The DVE Solver is mostly a UI shell; it does not trigger real validation recovery or consensus reconciliation on the backend.
    *   No backend service exists to orchestrate multi-step command sequences defined in the frontend workflow templates.

## 5. Network Topology and Connections
*   **Current State:** `NAP` displays a "Global Active Connections" list.
*   **Missing Functionality:**
    *   The connections list is hardcoded in the frontend.
    *   The backend `P2PManager` has the data, but there is no API endpoint to stream the real-time network topology or active peer list to the frontend.

## 6. Fintech Functionality (Plugin Refactoring)
*   **Current State:** The FinTech Validator service is tightly integrated into the core `backend_server` and `main.go` initialization.
*   **Refactoring Requirement:**
    *   The FinTech logic (`backend/internal/fintech`) and its handlers should be decoupled from the core server.
    *   Move towards a Plugin/Module architecture where FinTech functionality can be enabled/disabled or added as a separate service without modifying the core Nexus Memory Fabric.

## 7. API Routing Inconsistency
*   **Current State:** 
    *   `DVEManager` registers routes under `/dve/`.
    *   `DVEHandlers` registers routes under `/api/dve-nodes/`.
*   **Issue:** This redundancy and naming inconsistency causes the frontend to frequently target non-existent or misaligned endpoints (e.g., the frontend expects `/api/dve/...` but the backend provides `/dve/...`).

## 8. Real-time Event Streaming
*   **Current State:** While a WebSocket service exists, many UI updates still rely on polling or manual refreshes.
*   **Missing Functionality:**
    *   Comprehensive event mapping between backend state changes (node status, task completion, policy violation) and frontend Redux/Context updates via WebSockets.




# KNIRV-SERVER: Production Implementation Plan

This document outlines the roadmap to transition KNIRVNEXUS from its current development/demo state to a production-ready **Deterministic Validation & Active Memory Fabric**.

---

## 🎯 Objective
To replace all mock implementations, hardcoded placeholders, and "test mode" fallbacks with robust, secure, and integrated production-grade services.

---

## 🏗️ Phase 1: Security & Core Hardening
**Goal:** Transition from simulated monitoring to enforced hardware-assisted security.

### 1.1 eBPF Policy & Telemetry (High Priority)
*   **Action:** Replace placeholder metrics in `DVEManager.GetSystemHealth` with real-time data from eBPF probes.
*   **Tasks:**
    *   Implement actual syscall monitoring for memory usage, CPU cycles, and network I/O in `internal/ebpf/syscall_monitor.go`.
    *   Develop a "Deny-by-Default" eBPF security profile for agent containers.
    *   Integrate `LSM` (Linux Security Modules) with eBPF to enforce file-system path restrictions.

### 1.2 PQC Lifecycle Management
*   **Action:** Solidify the `EncryptionManager` for long-term data persistence.
*   **Tasks:**
    *   Implement a Key Rotation protocol for the **Kyber-768** master keys.
    *   Add support for Hardware Security Module (HSM) or TEE-backed key storage.
    *   Enforce PQC signing (`Dilithium-3`) on every Solution Node before execution.

### 1.3 Guardrails & Onboarding Configuration
*   **Action:** Implement dynamic guardrails that are dynamically configured based on user-defined parameters.
*   **Specification:** Guardrails are **dependent upon** the ingested ontologies, policies, and preferences as configured during the onboarding process.
*   **Tasks:**
    *   Design an ontology ingestion pipeline that accepts domain-specific knowledge graphs during DVE node onboarding.
    *   Implement a policy engine that loads and enforces network whitelists, eBPF forensics rules, and TEE attestation policies as defined in user preferences.
    *   Create a preference manager that persists user-configured constraints for memory limits, execution timeouts, and resource quotas.
    *   Ensure guardrails are dynamically reloadable without requiring full DVE node restarts.
    *   Validate that all guardrail rules are consistent with the ingested ontologies before activation.

---

## 🔗 Phase 2: KNIRVCHAIN Fabric Integration
**Goal:** Establish exclusive communication channels between DVE nodes and the central KNIRVCHAIN.

### 2.1 Centralized Blockchain Operations
*   **Action:** Deprecate localized blockchain logic in SERVER and centralize all state transitions within the internal **KNIRVCHAIN**.
*   **Tasks:**
    *   Refactor `NRNClient` to act as a session-based gateway to the internal `KNIRVCHAIN` gRPC/mTLS bus.
    *   Migrate all transaction validation and block height verification to the central chain authority.
    *   Implement PQC-secured session handshake for DVE-to-Chain exclusive channels.

### 2.2 DVE Creation & Provisioning
*   **Action:** Deprecate the "Rental" model in favor of **DVE Creation** and sovereign node ownership.
*   **Tasks:**
    *   Rename `DVERentalService` to `DVECreationService` and refactor the database schema to support permanent node registration.
    *   **CDE Deprecation:** Completely remove all references to Cloud Development Environments (CDE) and associated services.
    *   Implement TEE-bound session keys for each DVE node to maintain an exclusive persistent channel with `KNIRVCHAIN`.

### 2.3 Automated NRV Trace Validation
*   **Action:** Use exclusive channels to commit Reasoning Trace headers to the blockchain.
*   **Tasks:**
    *   Automate the anchor of **Evidence Packs** as PQC-signed state updates on the `KNIRVCHAIN`.
    *   Implement real-time audit trails where `KNIRVCHAIN` validates the intent-action correlation reported by the DVE eBPF probes.

---

## 🌐 Phase 3: Infrastructure & Scalability
**Goal:** Ensure the system is resilient and dynamically discoverable within the chain ecosystem.

### 3.1 Chain-Native Node Discovery
*   **Action:** Replace `seedDemoDVENodesIfEmpty` with on-chain node registration.
*   **Tasks:**
    *   Implement node discovery via the `KNIRVCHAIN` registry.
    *   Deploy the `ReputationEngine` as an on-chain scoring system based on verifiable DVE performance data.

### 3.2 Integrated Secret Management
*   **Action:** Use DVE-specific session keys for secret retrieval.
*   **Tasks:**
    *   Replace `.env` file dependencies with a session-based secret retrieval system from the central authority.
    *   Implement dynamic session rotation for all DVE-to-Chain channels.

---

## 📊 Phase 4: Frontend & Observability
**Goal:** Provide professional visualization of the Memory Fabric and Chain state.

### 4.1 Unified Fabric & Chain Monitoring
*   **Action:** Optimize the React Dashboard for high-throughput chain events.
*   **Tasks:**
    *   Implement **Apache Arrow Flight** consumers for sub-millisecond visualization of both local DVE metrics and global chain updates.
    *   Add a "Chain Session Inspector" to monitor the health of exclusive DVE-to-Chain channels.

---

## ✅ Success Criteria
1.  **Zero Mock Code:** No occurrences of "mock", "dummy", or "placeholder" in non-test production paths.
2.  **Chain-Anchored:** All DVE operations are verified and recorded via exclusive sessions with the internal **KNIRVCHAIN**.
3.  **CDE Removed:** No remaining CDE service dependencies or UI components.
4.  **Creation Model:** Successfully transitioned from a rental-based system to a sovereign DVE creation model.
5.  **Audit Ready:** The system produces PQC-signed evidence packs anchored to the blockchain for every automated decision.

---

**KNIRV-SERVER Team**
*Saturday, February 21, 2026*
