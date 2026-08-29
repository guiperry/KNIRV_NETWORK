# KNIRVENGINE

## Overview

KNIRVENGINE is the verification layer of the KNIRV Network — the desktop platform where code, compiled binaries, and autonomous workflows get proven out before anyone trusts them. Where the rest of the network is concerned with running agentic workflows under guardrails, KNIRVENGINE is concerned with the step before that: taking a piece of software a developer is about to ship, dropping it into an isolated sandbox, and putting it through the same static analysis, dynamic instrumentation, fuzzing, and traffic inspection a security review would use — so that what eventually gets registered as a report or minted as an workflow on the network has actually been exercised, not just reviewed by eye.

It ships as a single Go binary with an embedded React/TypeScript frontend, launched either as a browser client (`--browser`) or as a native desktop application. A local API server handles authentication, sandbox orchestration, and the AI-assisted error inference engine; the frontend is the operator console for every verification tool in the suite.

## The Verification Workflow

KNIRVENGINE's tools are gated behind one deliberate sequence, enforced by the frontend itself (`RequireSandbox`):

1. **Load a target.** The Dashboard opens a local project directory (via the File System Access API) or a single file, builds a file tree, and lets you pick the binary, script, or repository under test.
2. **Provision a sandbox.** The Sandbox view launches an unprivileged `bubblewrap` namespace around the target — read-only binds for the base system, a `tmpfs`, and (optionally) full namespace isolation with a single shared network path back out through the instrumentation layer. An `Xvfb` virtual display backs any GUI the target needs, streamed to the operator over a Go-native VNC bridge.
3. **Verify.** With the sandbox `running`, every tool in the suite operates against that same namespace and the same live target — nothing runs against production, and nothing runs unsandboxed.
4. **Register the result.** Findings, decompiled structure, fuzz corpora, and traffic captures produced during verification become the evidence trail that KNIRVORACLE checks before a report is registered or an workflow is minted onto the network.

Every verification view shows the active sandbox target inline, so an operator always knows which running namespace a given finding came from.

## The Verification Toolbench

Eight tool categories, each wired to real open-source engines rather than a simulated approximation:

### Sandbox
* **Bubblewrap (`bwrap`)** — unprivileged user-namespace isolation for the target process: bind mounts, network namespace control, `Xvfb`-backed display.
* **noVNC** — HTML5 RFB client docked into the operator UI, streaming the sandboxed target's display live.

### Proxy
* Traffic interception and replay for the sandboxed target's egress — flow list with method/host/status/TLS columns, request/response inspection, and manual intercept-and-edit for in-flight requests.

### Instrumentation
* **Frida** — attach to the sandboxed process by PID, inject a JS agent, hook exports and Java/ObjC methods at runtime.
* **proxychains-ng** — force a target's socket connections through the proxy layer without modifying the binary.
* **bpftrace** — kernel-level tracing of the sandboxed process via eBPF probes.

### Reversing
* **Cutter** — radare2 GUI: graph view, ESIL evaluation, and an `r2` command console.
* **ILSpy** — .NET decompiler: assembly tree to C#, IL, and MSIL disassembly.
* **JADX** — APK/DEX to Java, with smali fallback, resource browser, and deobfuscation.

### Fuzzing
* **AFL++** — coverage-guided fuzzing campaigns with live crash/hang/coverage monitoring.

### Static Analysis
* **Semgrep** — pattern-based AST analysis with rule packs for OWASP classes, secrets, and custom patterns.
* **Tree-sitter** — incremental parsing with a live syntax tree and S-expression queries.

### Packet Capture
* **Wireshark (TShark)** — live packet list with display filters, protocol columns, and stream following.
* **Zeek** — protocol-level network security monitoring and connection logging.

### Auth Audit
* **jwt_tool** — decode, tamper, and re-sign JWTs: `alg:none`, algorithm confusion, and known-key attacks.
* **SAML Raider** — SSO assertion tampering for federated auth flow testing.

## AI Error Inference Engine

KNIRVENGINE turns its own error handling into part of the verification story. When a tool run, sandbox launch, or API call fails, the inference engine:

* **Diagnoses automatically** — an LLM-backed analysis (Cerebras, Gemini, or DeepSeek, with automatic fallback between providers) classifies the error by type, severity, and probable root cause, with a confidence score.
* **Notifies in real time** — a header-mounted bell surfaces error counts and severity, auto-triggering analysis for critical and high-severity failures.
* **Explains interactively** — a chat interface answers follow-up questions about a specific error and walks through remediation step by step.
* **Recovers where it can** — retry logic with exponential backoff, plus a rule-based fallback analysis path when no LLM provider is reachable.
* **Falls back safely** — if every provider is unreachable, the rule-based path still returns a categorized result instead of leaving the operator with nothing.

## Tool Execution Architecture

KNIRVENGINE runs verification tools through six execution lanes, each optimized for a different interaction pattern:

### Lane 1 — Batch Scan
Tools that run to completion and return structured output. The generic handler spawns a subprocess, captures stdout/stderr, and parses the result. Overlap is prevented per session+tool via a running lock.
- **Tools:** Semgrep, JADX, ILSpy, jwt_tool
- **Endpoint:** `POST /api/v1/sandboxes/{id}/tools/{tool}/run`

### Lane 2 — Streaming Daemon
Tools that produce continuous output. The handler starts the process, reads stdout/stderr line-by-line, and fans events out to WebSocket clients. Historical events are replayed to new connections.
- **Tools:** bpftrace, tshark, zeek, afl-fuzz
- **Endpoints:** `POST /api/v1/sandboxes/{id}/tools/{tool}/start`, `POST /api/v1/sandboxes/{id}/tools/{tool}/stop`, `GET /api/v1/sandboxes/{id}/tools/{tool}/ws`

### Lane 3 — RPC Attach
Tools that attach to a running process via a bridge subprocess. The handler starts the bridge, reads its output, and forwards commands from the frontend over WebSocket.
- **Tools:** Frida
- **Endpoints:** `POST /api/v1/sandboxes/{id}/tools/{tool}/attach`, `POST /api/v1/sandboxes/{id}/tools/{tool}/detach`, `GET /api/v1/sandboxes/{id}/tools/{tool}/attach/ws`

### Lane 4 — Launch Modifier
Tools that modify the sandbox launch configuration rather than running as a separate process.
- **Tools:** proxychains-ng
- **Endpoint:** `POST /api/v1/sandboxes/{id}/tools/proxychains/configure`

### Lane 5 — Headless Native UI
Tools that run headless but produce structured data for KNIRVENGINE's own UI. The handler spawns the tool, parses its output, and returns function lists, decompiled code, or listings.
- **Tools:** Cutter
- **Endpoint:** `POST /api/v1/sandboxes/{id}/tools/{tool}/analyze`

### Lane 6 — Native Go
Tools implemented directly in Go, running in-process without subprocess spawning. They receive the session and raw JSON arguments, returning structured results.
- **Tools:** Tree-sitter, SAML Raider
- **Endpoint:** `POST /api/v1/sandboxes/{id}/tools/{tool}/native`

### Namespace Join (Phase 0)
All tool execution lanes that need sandbox access use `nsenter -t <InnerPid> -m -p -i -u -n -C -- <cmd>` to join the sandbox's namespaces. The inner PID is resolved from bwrap's child process at sandbox start time.

## Network Integration

KNIRVENGINE doesn't verify in isolation:

* **KNIRVORACLE** — report registration and workflow minting requests carry the verification evidence produced by the toolbench; wallet balance and transaction endpoints are proxied through the same client.
* **KNIRVCONTROLLER** — a linked wallet connection surfaces NRN balance and lets verified capabilities flow into the controller's identity/vault layer without re-authenticating.
* **Role-based access** — every top-level view and every sub-tool is gated per-user (`canAccessPage` / `canAccessSubPage`), so an operator's role determines which parts of the toolbench they can reach.

## Architecture

### Backend
* **API Server** — Go, Gorilla Mux, JWT-authenticated, with a dedicated security middleware chain (CORS, rate limiting, request validation).
* **Sandbox Manager** (`internal/api/sandbox_manager.go`) — owns the `bubblewrap`/`Xvfb` lifecycle, a dependency-check/install path for the underlying tool binaries, and a status + VNC WebSocket pair per session.
* **Inference Services** — provider-agnostic client layer over Cerebras, Gemini, and DeepSeek with fallback ordering.
* **Database** — SQLite-backed persistence for users, sessions, and sandbox history.

### Frontend
* **React 18 + TypeScript**, Tailwind CSS, React Router.
* **SandboxContext / `useSandboxSession`** — the single seam every tool integration consumes: session id, status, target label, namespace handle, display, and VNC path. No tool derives sandbox identity any other way.
* **RequireSandbox** — the gate component that blocks tool views until a sandbox is actually `running`.

### Deployment
* **Development** — hot-reloading Vite dev server against the Go API.
* **Production** — static assets embedded into the Go binary, served with `--production`.
* **Desktop** — Electron-wrapped native application; browser mode available via `--browser`.
* **Cross-platform** — Linux, macOS, Windows.

## Setup and Installation

### Prerequisites

* **Go** 1.21+ — [https://go.dev/doc/install](https://go.dev/doc/install)
* **Node.js and npm** 16+ — [https://nodejs.org/](https://nodejs.org/)
* **AI Provider Accounts** — API keys from Cerebras, Google (Gemini), or DeepSeek, used by the error inference engine
* A Linux host with `bubblewrap`, `Xvfb`, and the individual verification tool binaries (Frida, Ghidra, Semgrep, etc.) installed for sandbox provisioning — the Sandbox view's dependency check reports what's missing and can trigger install where supported

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/guiperry/KNIRV-Engine.git
   cd KNIRV-Engine
   ```

2. **Configure environment variables** — create a `.env` file in the project root:
   ```dotenv
   # AI providers (error inference engine)
   CEREBRAS_API_KEY=your_cerebras_api_key_here
   GEMINI_API_KEY=your_gemini_api_key_here
   DEEPSEEK_API_KEY=your_deepseek_api_key_here

   # Security
   JWT_SECRET=your_jwt_secret_key_here

   # Server configuration
   API_PORT=8081
   GUI_PORT=8080
   ```

3. **Build and run:**

   * **Quick start:**
     ```bash
     go build -o knirv-engine
     ./sync-env.sh
     ./knirv-engine
     ```

   * **Development mode:**
     ```bash
     go run main.go

     # in another terminal
     cd gui
     npm install
     npm run dev
     ```

* **Production mode:**
     ```bash
     go build -o knirv-engine

     cd gui
     npm install
     npm run build

     cd ..
     ./knirv-engine --production
     ```

### Container installer

The installer runs on Linux, macOS, and Windows. It provisions Docker/Docker
Desktop when required, downloads the published verification-tool bundle inside
the Alpine container, and starts the Linux KNIRVENGINE container:

```bash
go run ./cmd/installer
```

Release maintainers publish the archive with `make upload-tools`. This creates
`release_assets/tools.tar.gz` and uploads it to
`knirv/engine/tools/tools.tar.gz`, served as
`https://releases.knirv.com/engine/tools/tools.tar.gz`.

All KNIRVENGINE runtime state now uses the App Data root. On Linux this is
`~/.config/KNIRV-Engine` by default: databases and desktop TEE state live in
`data/`, tools in `data/bin/`, and plugins, logs, configuration, cache, and
migration reports live in their corresponding subdirectories. KNIRVENGINE no
longer creates runtime `data/`, `plugins/`, or migration-report files in the
working directory.

### Port Configuration

Configurable via `ports.config`:
- **API_PORT:** backend server port (default: 8081)
- **GUI_PORT:** frontend server port (default: 8080)

To change ports, edit `ports.config`, run `./sync-env.sh`, and restart. See [docs/PORT_CONFIGURATION.md](docs/PORT_CONFIGURATION.md).

## Usage Guide

### Authentication
1. Open `http://localhost:8080` (or your configured port).
2. Log in — a default admin user is created on first run.
3. Configure AI provider keys under Settings on first use.

### Running a Verification Pass
1. **Dashboard** — open the project directory or file under test.
2. **Sandbox → Bubblewrap** — launch a namespace targeting that project; wait for status `running`.
3. Move to any tool category (Proxy, Instrumentation, Reversing, Fuzzing, Static Analysis, Packet Capture, Auth Audit) — each one now operates against the live sandboxed target.
4. **Sandbox → noVNC** — watch the target's display directly if it's GUI-driven.
5. Review findings inline (functions decompiled, secrets found, flows intercepted, fuzz crashes) before deciding whether the target is fit to register as a network capability.

### Error Recovery
1. Watch the notification bell for auto-triggered analysis on critical/high-severity failures.
2. Open the chat assistant for step-by-step remediation on any specific error.
3. Let the retry/backoff path attempt automatic recovery before escalating manually.

## API Reference

### Authentication
- `POST /api/v1/auth/login` — authenticate, receive a JWT
- `POST /api/v1/auth/register` — create a user
- `POST /api/v1/auth/refresh` — refresh an existing token

### Sandbox
- `GET /api/v1/sandboxes/deps` — check verification-tool dependency status
- `POST /api/v1/sandboxes/deps/install` — install missing dependencies
- `GET /api/v1/sandboxes` — list active sandbox sessions
- `POST /api/v1/sandboxes` — provision a new sandbox for a target
- `GET /api/v1/sandboxes/{id}` — get session detail
- `DELETE /api/v1/sandboxes/{id}` — stop a session
- `GET /api/v1/sandboxes/{id}/ws` — status/log WebSocket
- `GET /api/v1/sandboxes/{id}/vnc` — VNC bridge WebSocket

### Tool Execution
- `POST /api/v1/sandboxes/{id}/tools/{tool}/run` — Lane 1 batch scan
- `POST /api/v1/sandboxes/{id}/tools/{tool}/native` — Lane 6 native Go tool
- `POST /api/v1/sandboxes/{id}/tools/{tool}/start` — Lane 2 start streaming
- `POST /api/v1/sandboxes/{id}/tools/{tool}/stop` — Lane 2 stop streaming
- `GET /api/v1/sandboxes/{id}/tools/{tool}/ws` — Lane 2 WebSocket events
- `POST /api/v1/sandboxes/{id}/tools/{tool}/attach` — Lane 3 RPC attach
- `POST /api/v1/sandboxes/{id}/tools/{tool}/detach` — Lane 3 detach
- `GET /api/v1/sandboxes/{id}/tools/{tool}/attach/ws` — Lane 3 WebSocket
- `POST /api/v1/sandboxes/{id}/tools/{tool}/analyze` — Lane 5 headless analysis
- `POST /api/v1/sandboxes/{id}/tools/proxychains/configure` — Lane 4 proxy config

### Health
- `GET /api/v1/health` — service health

## System Requirements

### Minimum
- **CPU:** 2+ cores
- **RAM:** 4GB+
- **Disk:** 1GB free space
- **OS:** Linux, macOS, or Windows 10+
- **Browser:** Chrome, Firefox, Safari, or Edge (latest)

### Recommended (for fuzzing/reversing workloads)
- **CPU:** 4+ cores
- **RAM:** 8GB+
- **Disk:** 5GB+ free space
- **Network:** stable connection for AI provider calls and network integration

## Testing

```bash
# Run all tests
make test

# Run specific test categories
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-frontend      # Frontend tests
make test-api           # API tests
make test-security      # Security tests
make test-performance    # Performance tests
make test-connectivity  # End-to-end connectivity tests

# Comprehensive suite
./scripts/run_comprehensive_tests.sh [mode]
# Available modes: unit, integration, frontend, api, cloud, desktop,
#                  security, performance, wasm, connectivity, ci, all, full
```

## Development and Extension

### Adding a Verification Tool
1. Add the tool component under `gui/src/components/tools/<category>/`.
2. Wire its route into the category's parent component (e.g. `Reversing.tsx`) and into `Sidebar.tsx`'s navigation tree.
3. Consume the sandbox contract exclusively via `useSandboxSession()` — never derive sandbox identity from any other source.
4. Extend `AuthContext`'s `canAccessSubPage` gating for the new sub-page.

### Frontend Customization
1. Components live under `gui/src/components/`.
2. Styling uses Tailwind CSS.
3. New top-level views extend the routing in `App.tsx` and the navigation array in `Sidebar.tsx`.

### Backend Extension
1. Add new services implementing the appropriate interfaces.
2. Register new API endpoints in `internal/api/simple_server.go` (general API) or `internal/api/sandbox_manager.go` (sandbox lifecycle).
3. For new verification tools, add a lane adapter in the appropriate `sandbox_tool_*.go` file (`sandbox_tool_scan.go` for Lane 1, `sandbox_tool_stream.go` for Lane 2, `sandbox_tool_attach.go` for Lane 3, `sandbox_tool_launchmod.go` for Lane 4, `sandbox_tool_headless.go` for Lane 5, `sandbox_tool_native.go` for Lane 6).
4. Extend database models in `internal/database/models/`.

## License

*MIT, Apache 2.0*

## Contributing

We welcome contributions! Please read our [contributing guidelines](CONTRIBUTING.md) before submitting pull requests.

## Documentation

Additional documentation is available in the `docs/` directory:
- [PORT_CONFIGURATION.md](docs/PORT_CONFIGURATION.md) — port configuration guide
- [MCP_INTEGRATION.md](docs/MCP_INTEGRATION.md) — MCP integration documentation
- [Testing.md](docs/Testing.md) — testing strategies and procedures
- [KNIRV-ENGINE_Whitepaper.md](KNIRV-ENGINE_Whitepaper.md) — architecture and vision
