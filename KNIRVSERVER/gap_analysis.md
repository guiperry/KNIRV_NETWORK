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


