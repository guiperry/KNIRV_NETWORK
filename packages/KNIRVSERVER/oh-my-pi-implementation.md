# KNIRVSERVER: Unification and Agentic Runtime Implementation Plan

This document details the strategy for unifying redundant functionality within the KNIRVSERVER codebase and provides a comprehensive implementation plan for integrating the `oh-my-pi` agentic runtime into the Deterministic Validation Environment (DVE) ecosystem.

---

## Part 1: Codebase Unification Report

Analysis of the current codebase reveals several overlapping systems that should be consolidated to improve security, maintainability, and architectural clarity.

### 1. Unified Runtime Orchestration
**Current State:**
- `backend/internal/services/container`: Manages Docker/Podman/Native runtimes specifically for DVE rentals.
- `backend/internal/runtime`: Contains the `UnifiedContainerManager` (UCM), a modern abstraction for DVE, TEE, Kata, and NOC (Nested Object Containers).

**Unification Strategy:**
- **Centralize in `runtime`**: Deprecate the orchestration logic in `services/container` and migrate it to the `UnifiedContainerManager`.
- **Merge Specifications**: Consolidate `ContainerSpec` and `ResourceLimits` into a single shared definition in `runtime/types.go`.
- **Integrated Provisioning**: Move `SSHProvisioner` and `PortAllocator` into the `runtime` package to ensure all container types benefit from secure provisioning.

### 2. Node Lifecycle Management
**Current State:**
- `backend/internal/services/dvemanager`: Tracks node health and reputation.
- `backend/internal/services/dvecreation`: Handles registration and session establishment.
- `backend/internal/services/dverental`: Manages the economic layer and triggers provisioning.

**Unification Strategy:**
- **Consolidate Node Logic**: Merge `dvecreation` into `dvemanager`. Node registration (local/blockchain) and session heartbeat management should live in one service.
- **Service Decoupling**: Refactor `dverental` to strictly manage the financial/billing state, delegating all technical node and container operations to `dvemanager` and `UnifiedContainerManager` respectively.

### 3. API & Handler Standardization
**Current State:**
- Handlers are split between `internal/services/*/handlers.go` and `internal/web/*.go`.

**Unification Strategy:**
- **Standardize on `internal/web`**: Move all HTTP/transport-specific logic to the `web` package.
- **Pure Services**: Ensure `internal/services` contains only business logic, making it easier to expose functionality via CLI, P2P, or WebSockets without duplicating code.

---

## Part 2: Agentic Runtime Integration (`oh-my-pi`)

The integration of `oh-my-pi` will provide every DVE with a "batteries-included" agentic capability for autonomous coding, research, and validation.

### Implementation Phases

### Phase 1: Core Definitions
- **ObjectType Extension**: Add `ObjectTypeAgent` to the runtime constants.
- **Capability Advertising**: Update the P2P discovery layer to allow nodes to advertise `agentic-runtime-support`.

### Phase 2: Container Environment
- **Base Image**: Develop `knirv-agent-oh-my-pi`, a specialized OCI image including:
    - The `oh-my-pi` Rust-based native engine.
    - Python/IPython persistent kernels.
    - LSP servers for 40+ languages.
    - Headless browser for autonomous web research.
- **Spec Customization**: Update `buildSpecForObjectType` in `runtime/manager.go` to automatically inject the required environment variables and mount the "Active Memory" (Markdown Fabric) workspace.

### Phase 3: Secure Execution & Monitoring
- **Viewport Proxying**: Connect the `oh-my-pi` terminal interface to the SERVER `ViewportProxy`, allowing users to interact with the agent via the web dashboard.
- **eBPF Security Profiles**: Implement specific eBPF `SyscallMonitor` profiles for `oh-my-pi` to monitor its autonomous tool usage (e.g., `git`, `python`, `curl`) and ensure it doesn't violate DVE isolation.

### Phase 4: Integration with "Markdown Fabric"
- **Reasoning Persistence**: Configure `oh-my-pi` to output its thoughts and task results directly into the DVE's persistent Markdown storage.
- **Autonomous Feedback Loop**: Enable the SERVER `CognitiveEngine` to read the agent's output and adjust DVE parameters (e.g., resource allocation) based on the agent's task complexity.

### Phase 5: User Experience
- **SERVER Dashboard**: Add an "Agent Command Center" to the frontend DVE view.
- **API Endpoints**: Expose subagent management via `/api/dve/{id}/agent/tasks`.

---

## Conclusion
By unifying the container runtimes and node management, we create a stable foundation for the agentic layer. Integrating `oh-my-pi` transforms the DVE from a passive execution environment into a proactive, intelligent workspace capable of autonomous solution discovery.
