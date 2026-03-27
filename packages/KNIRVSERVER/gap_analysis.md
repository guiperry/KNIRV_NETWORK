# KNIRV-SERVER Gap Analysis: Frontend to Backend Functionality (Updated March 2026)

This document outlines the current discrepancies, missing functionality, and architectural misalignments between the KNIRV-SERVER frontend dashboards and the backend implementation.

## 1. Terminal and SSH Integration (Unified via KNIRVCLI)
*   **Current State:** 
    *   The `ConsolePanel` (CDE-Panel) and `NetworkAccessModal` (NAP) feature terminal interfaces.
*   **Gaps & Requirements:** 
    *   **Unified Backend Terminal:** `KNIRVCLI` must be implemented in the backend to serve as the provider for *all* terminal and console sessions, including those within each DVE workspace.
    *   `ConsolePanel` input must be proxied via `KNIRVCLI` to the target DVE or system shell.
    *   Missing backend SSH/WebSocket proxy to connect web terminals to actual TEE containers.
    *   Terminal output is limited to historical log fetching; it needs real-time bi-directional streaming via `KNIRVCLI`.

## 2. DVE Monitoring and Task Tracking
*   **Requirement:** The backend MUST utilize and implement the `/api/dve/${nodeID}/tasks` endpoint for all per-node task monitoring.
*   **Current State:** 
    *   `MonitorPanel` attempts to fetch per-node tasks and metrics via `/api/dve/${nodeId}/tasks`.
    *   Backend currently has fragmented endpoints like `/api/dve/{id}/agent/tasks`.
*   **Gaps:**
    *   Consolidation: All per-node task tracking must be migrated to the `/api/dve/${nodeID}/tasks` endpoint.
    *   No backend implementation for `/api/dve/${nodeId}/metrics`.
    *   Task history in the UI is currently populated by demo data when live fetch fails.

## 3. Dynamic Policy Enforcement
*   **Current State:** 
    *   `PolicyEditor` and `NetworkAccessModal`'s Policy section provide UIs for configuring network whitelists, eBPF forensics, and TEE attestation.
*   **Gaps:**
    *   The "Commit Policy to Blockchain" and "Save Policy" buttons are placeholders.
    *   Backend lacks endpoints to receive, validate, and persist these policy updates.
    *   No mechanism to propagate updated policies to active DVE nodes/containers in real-time.

## 4. Workflow and Solver Automation (Drafting)
*   **Goal:** Define a structured orchestration layer for multi-step sequences.
*   **Proposed Workflow Structure (Draft):**
    ```json
    {
      "workflow_id": "validation-setup-001",
      "steps": [
        {
          "step_id": 1,
          "name": "TEE Initialization",
          "command": "knirvcli tee init --node $NODE_ID",
          "expected_output": "ENCLAVE_READY",
          "retry_policy": { "count": 3, "interval": "5s" }
        },
        {
          "step_id": 2,
          "name": "Validation Configuration",
          "command": "knirvcli validation config --env prod",
          "dependency": [1]
        }
      ],
      "status": "pending",
      "logs": []
    }
    ```
*   **Gaps:**
    *   Workflow execution is simulated; it needs a backend engine to parse and execute the above structure.
    *   No backend service exists to orchestrate multi-step command sequences defined in the frontend workflow templates.

## 5. Network Topology and Connections
*   **Current State:** 
    *   `NetworkAccessModal` displays a "Global Active Connections" list.
*   **Gaps:**
    *   The connections list is hardcoded in the frontend.
    *   The backend `P2PManager` has the data, but there is no API endpoint to stream the real-time network topology or active peer list to the frontend.

## 6. API Pathing and Routing Inconsistency
*   **Current State:** 
    *   `DVEHandlers` registers routes under `/api/dve-nodes/`.
    *   `FabricManagementHandlers` registers routes under `/api/fabric-management/`.
    *   `AgentHandlers` registers routes under `/api/dve/{id}/agent/`.
    *   Frontend `MonitorPanel` expects `/api/dve/`.
*   **Issue:** Redundancy and naming inconsistency cause the frontend to target non-existent or misaligned endpoints. There is no unified API versioning or naming convention (e.g., `dve-nodes` vs `dve`).

## 7. FinTech Functionality (Plugin Refactoring)
*   **Current State:** 
    *   FinTech logic (`backend/internal/fintech`) is still integrated into the core server.
    *   `PaymentHandlers` contains many mock implementations.
*   **Gap:** 
    *   The requirement to decouple FinTech into a Plugin/Module architecture remains unfulfilled.
    *   Backend needs real integration with Stripe/PayPal/Blockchain instead of mock responses.

## 8. Real-time Event Streaming
*   **Current State:** 
    *   A WebSocket service exists (`webSocketService.ts` on frontend, `backend/internal/services/websocket` on backend).
*   **Gaps:**
    *   Many UI updates still rely on polling or manual refreshes.
    *   Comprehensive event mapping between backend state changes (node status, task completion, policy violation) and frontend Redux/Context updates via WebSockets is missing for most components.

## 9. AetherDashboard vs KNIRVSERVER Integration
*   **Current State:** 
    *   **AetherDashboard has now been implemented in the frontend.** It exists as a specialized view for autonomous agent interaction.
*   **Requirement:** 
    *   Ensure backend services operate with `AetherDashboard` as needed.
    *   Integrate Aether's autonomous reasoning engine (`GeminiAgentService`) as a capability within the KNIRVSERVER Agent Command Center, allowing the Nexus to leverage Aether's planning and execution logic.

## 10. KNIRVCLI Backend Integration
*   **Current State:** 
    *   `KNIRVCLI` exists as a powerful standalone tool.
*   **Requirement:**
    *   Integrate `KNIRVCLI` as the backend communication layer for `KNIRVSERVER`.
    *   Replace direct REST/HTTP calls with `KNIRVCLI`'s unified API client abstraction.
    *   Utilize `KNIRVCLI`'s wallet and token management for FinTech functionality.
    *   **Unified Interface:** `KNIRVCLI` is the primary interface for all terminal interactions within the Nexus and its DVE instances.

## 11. Intelligent Onboarding and Validation Guardrails
*   **Requirement:** The onboarding process must ingest and process structured organizational intelligence to create validation guardrails that enforce these parameters across all DVE and Agentic operations.
*   **Value System Ingestion:**
    *   Guidelines
    *   Customs
    *   Etiquette
    *   Mission Statement
    *   Stated Values
    *   Goals & Objectives
    *   Insights
    *   Risk Appetite & Tolerance Levels
    *   Cultural Context & Regional Nuances
*   **Ontology Ingestion:**
    *   Trade Secrets
    *   Business Logic
    *   User Data
    *   Rules
    *   Regulations
    *   Procedures
    *   Policies
    *   FAQs
    *   Customer Service Bullets
    *   Industry-Specific Jargon & Taxonomy
    *   Stakeholder Hierarchy & Decision Rights
*   **Gaps:**
    *   No current mechanism to "ingest" these semantic structures and transform them into active, enforceable guardrails.
    *   The `PolicyEditor` is currently limited to technical parameters (IPs, Ports, TEE types) and does not support qualitative/semantic guardrails.
    *   Missing integration between the onboarding flow and the `CognitiveEngine` to ensure these values are respected in autonomous reasoning and intent-action correlation.

## 12. Neural Desktop and Cognitive Engine Integration
*   **Requirement:** The **Neural Desktop** (Aether component) and the **Cognitive Engine** must be fully integrated with the current **Inference Engine** in the backend.
*   **Current State:**
    *   Inference Engine (`backend/internal/services/inferencer`) handles low-level LLM calls.
    *   Cognitive Engine (`backend/internal/services/cognitiveengine`) manages learning and adaptation metrics.
    *   Neural Desktop (AetherDashboard components) interacts directly with external Gemini APIs.
*   **Integration Objectives:**
    *   **Unified Execution Pipeline:** All autonomous reasoning requests from the Neural Desktop must be routed through the backend Inference Engine to ensure policy enforcement and TEE security.
    *   **Closed-Loop Learning:** Insights generated by the Neural Desktop/Aether sessions must be fed back into the Cognitive Engine's adaptation metrics.
    *   **Contextual Resonance:** The Cognitive Engine should provide dynamic context weights to the Inference Engine based on the ingested Value System and Ontology (from onboarding).
*   **Gaps:**
    *   Neural Desktop currently bypasses the backend Inference Engine for Gemini-2.5-flash calls.
    *   No backend service exists to synchronize Aether's "Thoughts" and "Memories" with the Cognitive Engine's adaptation state.
    *   The Inference Engine does not currently receive real-time learning parameters from the Cognitive Engine.
