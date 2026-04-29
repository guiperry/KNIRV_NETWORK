# Investigation Report: KNIRVHASHER WASM Integration with DVE eBPF Monitoring

## 1. Objective
To integrate generated `rules.wasm` and `resolution.wasm` (from KNIRVHASHER) into the I/O execution path of DVE containers, triggered by eBPF signals derived from DVE "Badges."

---

## 2. Current State Analysis

### 2.1 KNIRVHASHER WASM Pipeline
The `KNIRVHASHER` transformer pipeline generates TinyGo source code based on GPT-driven "logits" or templates.
*   **WASM Types:** Rule, Resolution, Patch.
*   **Key Exports:** 
    *   `resolution.wasm`: `resolveError() bool`, `ErrorClass() uint32`.
    *   `rules.wasm`: `GuardrailClass() uint32`.
*   **WASM Runtime:** Uses `wazero` (via `WazeroGate`) for execution within the Go backend.

### 2.2 KNIRVSERVER eBPF Infrastructure
The `ebpf` package in KNIRVSERVER provides:
*   **Syscall Monitoring:** Captures entry into syscalls like `read`, `write`, `open`.
*   **LSM (Linux Security Modules):** Restricts file opens and network connects based on policies.
*   **VirtualContainerManager:** Tracks container IDs and PIDs in eBPF maps.
*   **Signals:** Currently, eBPF signals are handled as "Events" (via `ringbuf`) that are consumed by the `EventCollector` and published to the `GuardrailEngine`.

### 2.3 I/O Execution Path
Current I/O for DVE containers is handled via:
*   **Direct Runtime (Docker/Podman):** Standard container I/O.
*   **ViewportProxy:** A userspace HTTP/WebSocket reverse proxy (`httputil.ReverseProxy`) that wraps container endpoints.

---

## 3. Integration Challenges
1.  **Transparent Interception:** eBPF syscall tracing is non-blocking (observe-only). To modify I/O data (e.g., in a `read` syscall), we would need `bpf_override_return` (limited to specific kprobes) or a FUSE-based approach.
2.  **Execution Context:** WASM execution (`wazero`) happens in userspace. Triggering it from eBPF requires a userspace "switchboard" that listens to eBPF signals and then invokes the WASM `resolve()` function.

---

## 4. Proposed Fulfilment Options

### Option A: Userspace Interception (Proxy-Driven) — Exclusive to `rules.wasm`
Integrate Guardrail enforcement into the `ViewportProxy` middleware for high-level policy validation.
*   **Target:** `rules.wasm` (Guardrail Class validation).
*   **Mechanism:** Modify `ViewportProxy.HandleHTTP` to intercept incoming requests. Before proxying to the container, the proxy invokes the `GuardrailClass()` function in `rules.wasm` via `WazeroGate`.
*   **Trigger:** Active "Badges" on the DVE provide the ontology tags that select which `rules.wasm` to load.
*   **Pros:** Low latency for web-facing agents; full context of HTTP headers/body for policy checks.
*   **Cons:** Limited to traffic passing through the proxy.

### Option B: Kernel-to-Userspace Signaling (eBPF-Driven) — Exclusive to `resolution.wasm`
Use eBPF to detect deep-level system anomalies and trigger automated resolution logic.
*   **Target:** `resolution.wasm` (Error Resolution and Recovery).
*   **Mechanism:** Use eBPF LSM or syscall probes to monitor for specific error patterns (e.g., EPERM, ENOENT, or specific I/O corruption). On detection, eBPF pushes an event containing the error context to a userspace `ResolutionWorker`.
*   **Execution:** The worker invokes `resolveError()` in the `resolution.wasm` associated with the DVE's Badge. If `true`, the worker attempts a `RemediationAction` (e.g., `applyPatch` or `restart_service`).
*   **Pros:** Deep transparency; handles non-networked I/O errors and kernel-level anomalies.
*   **Cons:** Higher complexity in synchronizing kernel signals with userspace WASM execution.

---

## 5. Implementation Roadmap

1.  **Badge-to-WASM Mapping:** 
    *   Map "Guardrail" tags in Badges to `rules.wasm`.
    *   Map "Resolution/Recovery" tags in Badges to `resolution.wasm`.
2.  **Viewport Guardrail Hook (Option A):** Implement a `GuardrailMiddleware` in `ViewportProxy` that executes `rules.wasm` for every session initialization.
3.  **eBPF Resolution Trigger (Option B):** 
    *   Update `ebpf.EventCollector` to recognize "Resolution Signals."
    *   Create a `ResolutionService` that maintains a pool of `WazeroGate` instances specifically for `resolution.wasm`.
4.  **Remediation Link:** Ensure `resolution.wasm` results are piped directly into the `GuardrailEngine`'s remediation pipeline for automated recovery.

---

## 6. Conclusion
The most viable path for "operational fulfilment" is a **Hybrid Approach**: Use eBPF for low-level signal detection based on Badge ontology, and use the `WazeroGate` in userspace to execute the generated `resolution.wasm` as a "logic gate" for remediation.
