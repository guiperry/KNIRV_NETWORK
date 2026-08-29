# **KNIRV-ENGINE: The Verification Layer for Software and AI Agents**

### **Abstract**

Software that runs unexamined is software that runs untrusted. The **KNIRV-ENGINE** is the verification layer of the KNIRV Network: a desktop platform where code, compiled binaries, and autonomous workflows are placed in an isolated sandbox and exercised against a full suite of static analysis, dynamic instrumentation, fuzzing, reverse engineering, network inspection, and authentication-flow auditing tools. The same techniques a security review would use, run as a matter of course before anything is trusted. Where the rest of the KNIRV Network concerns itself with running agents under guardrails, KNIRV-ENGINE concerns itself with the step that has to happen first: proving that a piece of software behaves the way its author claims it does. The evidence it produces — decompiled structure, fuzz results, intercepted traffic, verified secrets, tampered-and-retested auth tokens — is what KNIRVORACLE checks before a report is registered or an agentic workflow is minted onto the network.

### **1. Introduction**

#### **1.1 The Problem: Trust Without Evidence**

Agent networks, plugin marketplaces, and report registries all face the same gap: a submission looks correct, but "looks correct" is not "was verified." A binary can claim a capability it doesn't have, mishandle a token it claims to validate, or fail silently under adversarial input — and none of that is visible from source review alone. Verifying it properly means running it: in isolation, under instrumentation, against a hostile network peer, with a fuzzer generating the inputs its author didn't think to test.

Doing that by hand, tool by tool, host by host, is exactly the friction that causes teams to skip it. KNIRV-ENGINE exists to remove that friction — one sandboxed target, one operator console, the full verification toolchain wired directly against it.

#### **1.2 The Vision: Verification as a Gate, Not an Afterthought**

KNIRV-ENGINE treats verification as a precondition, not a courtesy. A target is loaded, a sandbox is provisioned around it, and only then does any tool in the suite get to touch it — nothing in KNIRV-ENGINE runs against a live, unsandboxed system. The output of that process — the findings, the corpora, the traffic captures — becomes the evidence a report carries into KNIRVORACLE's registration flow and into a minted workflow's provenance. Trust on the network is downstream of verification, not a separate claim made alongside it.

#### **1.3 Core Concepts**

1. **Sandboxed ground truth.** Every verification tool operates against the same live, isolated namespace — one target, one sandbox, one consistent set of findings, never a simulation running alongside the real thing.
2. **Composable tradecraft, not a monolith.** Each verification category wraps a focused, purpose-built open-source engine (Ghidra, Frida, Semgrep, AFL++, Wireshark, and others) rather than reimplementing their functionality — KNIRV-ENGINE is the orchestration and evidence layer above them.
3. **Evidence as the unit of trust.** What KNIRVORACLE registers is not a self-report from the submitter; it's the artifact produced when the submission was actually run.

### **2. Core Architecture & Components**

#### **2.1 The Sandbox Layer**

At the center of KNIRV-ENGINE is an unprivileged isolation boundary built on **Bubblewrap (`bwrap`)**: read-only binds for the base system, an isolated `tmpfs`, and namespace controls (`unshare-all` with an optional single shared network path) around the target process. Where the target is GUI-driven, an ephemeral **Xvfb** display backs it, streamed to the operator through a Go-native VNC bridge and docked directly into the console via **noVNC**. A dependency-check pass reports which underlying tool binaries are present on the host and can trigger installation for what's missing, so a sandbox never silently degrades into a partial verification.

Every sandbox session carries a stable identity — session id, status, target label, namespace handle, display, VNC path — that every downstream tool consumes through one narrow contract. No tool derives which namespace it's operating on from a filename or a label; it asks the sandbox session directly. This is what makes it safe to run eight independent tool categories against one target without any of them drifting onto the wrong process.

#### **2.2 The Verification Toolbench**

Each category below is a thin, purpose-built operator surface over a real engine, not a simulated approximation of one:

| Category | Tools | What it proves |
| --- | --- | --- |
| **Sandbox** | Bubblewrap, noVNC | The target is actually isolated, and its display is observable live. |
| **Proxy** | Interception/replay engine | What the target actually sends and receives, not what its documentation claims. |
| **Instrumentation** | Frida, proxychains-ng, bpftrace | Runtime behavior at the function-hook, socket, and kernel-trace level. |
| **Reversing** | Ghidra, Cutter, ILSpy, JADX | The target's actual compiled structure across native, .NET, and Android binaries. |
| **Fuzzing** | LibAFL, AFL++ | Resilience against inputs the target's author didn't anticipate. |
| **Static Analysis** | Semgrep, Tree-sitter, TruffleHog | Pattern-level vulnerability classes, syntax-tree structure, and embedded secrets. |
| **Packet Capture** | Wireshark (TShark), Zeek | Protocol-level ground truth for anything the target puts on the wire. |
| **Auth Audit** | jwt_tool, SAML Raider | Whether the target's authentication flow actually resists tampering, not just whether it implements one. |

#### **2.3 The AI Error Inference Engine**

Verification runs generate failures — a fuzzer stalls, a sandbox fails to provision, a hook fails to attach — and those failures are themselves diagnostic signal. The Error Inference Engine treats them that way: an LLM-backed analysis layer (Cerebras, Gemini, or DeepSeek, with automatic fallback across providers) classifies each failure by type, severity, and probable root cause, surfaces it through a severity-aware notification system, and offers an interactive chat interface for step-by-step remediation. A rule-based fallback path keeps this working even with every provider unreachable, and a retry-with-backoff strategy resolves the transient cases automatically — so a verification run's own instrumentation doesn't become the next thing an operator has to debug by hand.

### **3. Verification Workflow**

KNIRV-ENGINE enforces one sequence, not by convention but by construction — the tool views themselves refuse to activate outside it:

1. **Load.** The Dashboard opens a local project or file — the target under test — through the browser's File System Access API, building a navigable file tree.
2. **Provision.** The Sandbox view launches a Bubblewrap namespace scoped to that target; status moves from `provisioning` to `running` before anything else is allowed to proceed.
3. **Verify.** Every tool category — Proxy, Instrumentation, Reversing, Fuzzing, Static Analysis, Packet Capture, Auth Audit — operates against that one running namespace, in whatever combination the target calls for: decompile a binary while fuzzing it, intercept its traffic while hooking its TLS calls, audit its tokens while tracing its syscalls.
4. **Register.** The evidence produced — function lists, crash corpora, flagged secrets, tampered-token results, captured flows — travels with the report or workflow submission into KNIRVORACLE's registration path.

### **4. Integration with the KNIRV Network**

KNIRV-ENGINE is not a standalone security scanner; it is the entry point through which software earns standing on the network.

* **KNIRVORACLE** — report registration and workflow-minting requests carry the verification evidence generated during a KNIRV-ENGINE session; wallet balance and transaction calls are proxied through the same authenticated client.
* **KNIRVCONTROLLER** — a linked wallet connection surfaces NRN balance directly in the engine, so a verified report can flow into the controller's identity and vault layer without a second authentication step.
* **Role-based access control** — every top-level tool category and every individual sub-tool is gated per user, so which parts of the verification surface an operator can reach is itself a governed, auditable decision.

### **5. Security and Isolation Guarantees**

* **Sandboxed by construction, not by convention.** `RequireSandbox` blocks every tool view until the underlying namespace is confirmed `running` — there is no path to point a tool at an unsandboxed target.
* **Namespace isolation.** Bubblewrap's unprivileged user namespaces, combined with read-only system binds and a `tmpfs` working directory, keep the target from touching the host filesystem outside what it's explicitly given.
* **Single source of sandbox identity.** The `useSandboxSession` contract is the only path any tool has to a running namespace, preventing a tool from silently operating against the wrong target.
* **JWT-authenticated, role-gated access.** All API access requires an authenticated session, and page/sub-page visibility is enforced against the authenticated user's role on both the frontend and the underlying permission checks.

### **6. Conclusion**

KNIRV-ENGINE makes the case that trust in software — and in the AI workflows built on top of it — should rest on evidence produced by actually running the thing, not on a submission's own account of itself. By making sandboxed, instrumented verification the default first step rather than an optional extra, and by feeding what that verification produces directly into KNIRVORACLE's registration path, KNIRV-ENGINE turns "this was checked" from a claim into a matter of record. It is the layer that lets the rest of the KNIRV Network run agentic workflowss under guardrails with some confidence that what's inside those guardrails was actually examined first.
