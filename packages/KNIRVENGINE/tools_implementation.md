# Feature: Real Tool Execution for the 13 Mock Tool Consoles

This plan spans two codebases — the GUI (`packages/KNIRVENGINE/gui`, React/TS/Vite) and the Go backend (`packages/KNIRVENGINE`, module `KNIRVENGINE/desktop-client`) — treat them as one implementation unit; they ship together.

## Feature Description

Bubblewrap + noVNC (see `sandbox_implementation.md`, already implemented) turned the Sandbox into a real Go-orchestrated namespace + framebuffer, gated every other tool route behind a running session, and bundled `bwrap`/`Xvfb`/`x11vnc` into the shipped binary so the operator never installs system packages by hand. **Proxy** followed the same real-execution pattern independently: `api/sandbox_manager.go` runs an actual embedded HTTP/CONNECT proxy and streams real captured flows to `Proxy.tsx`.

The remaining tool consoles were still exactly what `sandbox_implementation.md` explicitly scoped *out*: local-`useState` consoles rendering hardcoded arrays (`seedFlows`-style fixtures), with a "Run"/"Load"/"Attach" button that mutates local state after a `setTimeout`. Nothing shells out, nothing reads the real sandboxed target. This plan is the follow-up that was deliberately deferred — it wires each remaining tool to a real, bundled binary (or, for two of them, real in-process Go code) operating against the real `SandboxSession`, the same way Bubblewrap/noVNC/Proxy already do.

**This revision folds in a duplicate-function audit** (below): of the original 16 mock consoles, 3 were found to duplicate another tool's real-world function and are cut, leaving **13 tools to build**. It also folds in a recommended build order and three previously-open design questions the user has since answered.

## Current State Audit (verified in this session)

| Tool page | File | Real backend today? |
|---|---|---|
| Bubblewrap | `gui/src/components/tools/sandbox/Bubblewrap.tsx` | ✅ real (`SandboxManager`) |
| noVNC | `gui/src/components/tools/sandbox/NoVnc.tsx` | ✅ real (Go-native WS↔TCP bridge) |
| Proxy | `gui/src/components/tools/Proxy.tsx` | ✅ real HTTP/CONNECT capture (`api/sandbox_manager.go:558-816`); intercept/hold/drop/replay buttons are **still local-only** — flagged as a Phase 3-adjacent gap below |
| Frida | `instrumentation/Frida.tsx` | ❌ mock (`seedLog`, hardcoded `processes[]`) |
| bpftrace | `instrumentation/Bpftrace.tsx` | ❌ mock |
| proxychains-ng | `instrumentation/ProxychainsNg.tsx` | ❌ mock |
| ~~Ghidra~~ | ~~`reversing/Ghidra.tsx`~~ | **cut — duplicate of Cutter, see Consolidation Audit** |
| Cutter | `reversing/Cutter.tsx` | ❌ mock — **kept, becomes the sole native-binary reversing tool** |
| ILSpy | `reversing/ILSpy.tsx` | ❌ mock |
| Jadx | `reversing/Jadx.tsx` | ❌ mock |
| AFL++ | `fuzzing/AflPlusPlus.tsx` | ❌ mock — **kept, sole fuzzing tool** |
| ~~LibAFL~~ | ~~`fuzzing/LibAFL.tsx`~~ | **cut — duplicate of AFL++, see Consolidation Audit** |
| Semgrep | `staticanalysis/Semgrep.tsx` | ❌ mock (hardcoded `findings[]`, `runScan` just toggles a spinner) — **kept, sole static-analysis findings tool** |
| Tree-sitter | `staticanalysis/TreeSitter.tsx` | ❌ mock |
| ~~TruffleHog~~ | ~~`staticanalysis/TruffleHog.tsx`~~ | **cut — subset of Semgrep's `p/secrets` ruleset, see Consolidation Audit** |
| Wireshark | `packetcapture/Wireshark.tsx` | ❌ mock |
| Zeek | `packetcapture/Zeek.tsx` | ❌ mock |
| jwt_tool | `authaudit/JwtTool.tsx` | ❌ mock |
| SAML Raider | `authaudit/SamlRaider.tsx` | ❌ mock |

`grep -rl` across `api/*.go` for every tool/binary name returns **zero hits** — confirmed none of this exists in the backend yet, for any of the 16 originally-scoped consoles.

All 7 category pages already call `useSandboxSession()` and show a `session.targetLabel` chip (that part of `sandbox_implementation.md`'s Phase 6 *is* done), and all 7 routes are already gated by `RequireSandbox` in `App.tsx:262-268`. The route/session/gating skeleton this plan builds on is real and does not need to be rebuilt.

## Tool Consolidation Audit

Before designing backend integrations for 16 tools, each was checked against the others for functional overlap — building two real integrations that answer the same question wastes bundling budget and gives the operator two consoles to reconcile instead of one to trust. **This is a recommendation, applied to this plan's scope going forward; it's reversible** — nothing below assumes any of the 3 cut tools can never come back as a 14th/15th/16th console later.

### Confirmed duplicates — one kept per pair, chosen for size / bundling ease / architecture fit (per explicit standing criteria)

| Pair | Overlap | Keep | Cut | Why |
|---|---|---|---|---|
| **Ghidra vs. Cutter** | Both decompile/disassemble native binaries (ELF/PE/Mach-O) | **Cutter** (rizin-backed) | Ghidra | Ghidra is a multi-hundred-MB Java app with no package-manager or single-binary distribution (this plan's own Phase 1 originally had to flag it "manual install only"). Cutter's engine, rizin, is a small native binary, scriptable headlessly with JSON output, and fits the Lane 1/5 bundling model directly — smaller, easier to bundle, and closer to how every other tool in this plan is acquired. |
| **AFL++ vs. LibAFL** | Both are coverage-guided fuzzers | **AFL++** | LibAFL | AFL++ is a ready-to-run CLI (`afl-fuzz`) with a stable machine-readable `fuzzer_stats` file. LibAFL is a Rust *fuzzer-construction framework*, not a tool — using it means writing and compiling a bespoke Rust harness per target binary, which breaks the "operator points the tool at a target, it runs" pattern every other console in this plan assumes. |
| **Semgrep vs. TruffleHog** | TruffleHog's entire function (secret detection) is a subset of what Semgrep's own `p/secrets` ruleset already covers — visible directly in `Semgrep.tsx`'s existing mock data (`detected-private-key`) and its default ruleset string (`p/owasp-top-ten p/secrets p/golang`) | **Semgrep** | TruffleHog | Semgrep is strictly broader (general SAST + secrets), and folding secret-scanning into its existing ruleset removes an entire tool plus a whole separate pip-vs-Go acquisition strategy. TruffleHog's one real edge — live credential verification (is this key still active?) — is a genuine capability loss, named here in case it matters enough later to justify a 14th console. |

### Reviewed and confirmed NOT duplicates (kept as-is)

- **Jadx / ILSpy / Cutter** — despite all three being "decompilers," each targets a disjoint bytecode format: Android DEX, .NET IL, and native machine code respectively.
- **Frida vs. bpftrace** — Frida does userspace/managed-runtime hooking *with modification* (JS-injected, bridges Java/ObjC, can change return values); bpftrace is kernel/syscall-level *read-only* tracing with near-zero overhead. Complementary layers, a standard real-world pairing (app-layer instrumentation + kernel-layer observability), not a duplicate.
- **Wireshark vs. Zeek** — Wireshark/tshark is packet-by-packet manual inspection; Zeek is automated behavioral/session-log triage (conn.log, dns.log, etc.). Routinely run side-by-side in real DFIR work. Kept both; Wireshark/tshark is the lighter single-tool fallback if Packet Capture is ever cut to one.
- **Proxy vs. proxychains-ng** — Proxy *observes/intercepts* traffic already reaching it; proxychains-ng *forces* a target's sockets through an external chain (e.g. Tor) via LD_PRELOAD, working even on binaries that ignore proxy env vars. Kept, but noted as the most marginal survivor — some of its value may become redundant once Proxy's netns-transparent redirect is real; revisit then.
- **jwt_tool vs. SAML Raider** — different token formats entirely (JWT vs. SAML/XML), no overlap.
- **Tree-sitter vs. Semgrep** — different output types (AST vs. security findings). Semgrep is *built on* tree-sitter parsers internally, but the operator-facing consoles serve different purposes (code-structure navigation vs. vulnerability findings).

**Net effect: 16 mock consoles → 13 to build.** `Ghidra.tsx`, `LibAFL.tsx`, and `TruffleHog.tsx` are removed from the app entirely as part of this plan (route, nav card, and file — see Phases 3, 5, and 7) rather than left permanently mock.

## Problem Statement

1. No backend module exists for any of the 13 remaining tools — there is nothing to call.
2. The 13 tools are not homogeneous. Wireshark and Semgrep have almost nothing in common operationally: one is a long-lived packet stream, the other is a one-shot scan against files already sitting in a bind mount. Treating them identically would force either a leaky abstraction or 13 bespoke, un-reusable implementations. **The core design problem this plan solves is picking the right number of reusable execution patterns** (this doc calls them "lanes") so 13 tools become 6 backend shapes, not 13.
3. `sandbox_deps.go`'s dependency model only knows one acquisition strategy: "resolve via system package manager, `microdnf`/`apt-get`/etc." Several of the 13 tools don't ship that way — Semgrep/jwt_tool are pip packages, AFL++ may need a source build depending on distro, Cutter's rizin engine is best acquired via package manager or a GitHub release rather than `microdnf`. The dependency + bundling infra (`sandbox_deps.go`, `sandbox_tools.go`, `scripts/bundle-sandbox-tools.sh`, `Containerfile`) has to grow new acquisition strategies before any of these tools can be "just there" the way `bwrap` is. Two tools (Tree-sitter, SAML Raider) sidestep this problem entirely by not being external binaries at all — see the Consolidation Audit's Lane 6.
4. **The unresolved architectural question**: tools that need to operate *inside* the running sandbox (Frida attaching to the target's PID, bpftrace tracing its syscalls, tshark/Zeek capturing its network namespace's traffic, proxychains-ng wrapping its socket calls) must join the *same* Linux namespaces `bwrap` already created for `SandboxSession.Pid` — a brand-new subprocess spawned by the Go backend does **not** automatically land inside those namespaces just because a session is "running." This plan's Phase 0 is dedicated entirely to closing that gap before any Lane 2/3/4 tool is attempted, because every other phase depends on the answer.
5. Frida specifically has no first-party Go binding (`frida-core`'s official bindings are Python/Node/Swift/.NET/Rust) — Go can only reach it via a subprocess speaking Frida's own RPC, which is a materially different integration shape than "spawn a CLI, parse stdout."

## Solution Statement

- **Group the 13 tools into 6 execution lanes** by how their real binary (or, for two tools, real Go code) actually behaves, and design one reusable backend+frontend pattern per lane instead of 13 one-offs.
- **Phase 0**: resolve the namespace-join question empirically (does `nsenter -t <SandboxSession.Pid> -a` land a new process inside the running sandbox?) before committing to Lane 2/3/4 designs.
- **Extend, don't replace**, the existing `api/sandbox_deps.go` / `api/sandbox_tools.go` / `scripts/bundle-sandbox-tools.sh` / `Containerfile` infra with additional acquisition strategies (pip, GitHub-release binary), reusing the exact `DependencyStatus` reporting shape the Bubblewrap UI already renders (`missingDeps` banner in `Bubblewrap.tsx:161-192`) — that UI is lane-agnostic and needs zero changes to also show "Semgrep missing, pip install semgrep."
- **Extend `SandboxManager`**, not a new competing manager — every one of these tools only makes sense in the context of one running `SandboxSession`, so tool processes are children of that session (spawned via the existing `spawn()`/`handleOutput()`/`watchProcess()` helpers at `api/sandbox_manager.go:443-479`, reused verbatim), not a parallel lifecycle.
- **One new generic REST/WS route shape** — `/api/v1/sandboxes/{id}/tools/{tool}/...` — reused by all 13, differing only in payload shape per lane, not per tool.
- **One new frontend hook per lane family** (`useToolScan` serves both Lane 1 and Lane 6, `useToolStream` serves Lane 2, `useToolAttach` serves Lane 3) that each of the corresponding tool pages imports, replacing local mock state with real data while keeping the existing, already-well-built UI shells untouched.
- **Prove each lane on one pilot tool before rolling out the rest of that lane** — this plan gives full file-level tasks for the generic infra plus 5 pilots (Semgrep, proxychains-ng, bpftrace, Frida, Cutter) and rolls out the remaining 8 tools against the same proven patterns, per the Recommended Build Order below.

## Out of Scope / Non-Goals

- **Not re-litigating Bubblewrap/noVNC/Proxy** — those are done; this plan only touches the 13 tools kept after the Consolidation Audit, plus the one flagged Proxy gap (intercept/hold/drop/replay).
- **Not building a second orchestration daemon.** Per `sandbox_implementation.md`'s own rejected-alternative note, everything stays inside the existing single-language Go backend, extending `SandboxManager`.
- **Not attempting true multi-session tool isolation** — like the sandbox itself, this plan assumes one active `SandboxSession` and tools operate against it. A future multi-session sandbox would need this plan's `{id}` route segment anyway, so nothing here blocks that later.
- **Cutter's full GUI is not embedded in noVNC in this plan's default path** — see Lane 5 design decision in Phase 7; headless-with-native-UI is the confirmed default (user sign-off below), GUI-in-VNC is called out as the fallback if headless coverage proves insufficient.
- **SAML Raider has no meaningful standalone CLI** (it originated as a Burp Suite extension) — **confirmed**: re-scoped to a small Go-native SAML assertion mutator (mirroring how Proxy is a Go-native HTTP proxy, not a wrapped mitmproxy binary), built in Phase 3 alongside the other tools that don't need namespace-join.
- **Ghidra, LibAFL, and TruffleHog are explicitly out of scope** — see Consolidation Audit. Their existing mock pages are removed, not left in a permanently-unbuilt state.

## Feature Metadata

**Feature Type**: New Capability (backend process orchestration + frontend data wiring, ×13, plus 3 mock-console removals)
**Estimated Complexity**: Very High (mostly from breadth — 13 distinct real-world tools across 6 lanes — not from any single piece being individually hard)
**Primary Systems Affected**: `packages/KNIRVENGINE/api` (extends `sandbox_manager.go`, `sandbox_deps.go`, `sandbox_tools.go`; new per-lane files), `packages/KNIRVENGINE/gui/src` (13 tool pages + 3 removed pages, new hooks, `sandboxService.ts`), `packages/KNIRVENGINE/scripts/bundle-sandbox-tools.sh`, `packages/KNIRVENGINE/Containerfile`, `go.mod` (Tree-sitter cgo bindings)
**Dependencies**: see the per-tool acquisition table in Phase 1. No new Go module deps expected for Lanes 1/2/4 (`os/exec`, `gorilla/websocket`, `encoding/json` all already present per `go.mod`); Lane 3 (Frida) needs a subprocess bridge, not a Go module; Lane 6 needs one new cgo-based Go module (Tree-sitter bindings) plus a C toolchain at build time.

## Related Work

**Implements**: user request, this session ("draft a roadmap for tool bundling and real execution like Bubble Wrap and noVNC"), extended by a same-session follow-up ("ensure no duplicate functioning tools; recommend an execution order").

**Back-references**:
- `sandbox_implementation.md` — the plan this one continues. Its Phase 6 ("Downstream Tool Contract") explicitly deferred all remaining tools' real backend work to "separate, future, per-tool work" — this is that work.
- `api/sandbox_manager.go`'s Proxy implementation (lines 558-816) — the only prior art in this repo for "a real tool, not a wrapped CLI" (a Go-native reimplementation rather than shelling out to mitmproxy). Referenced directly in the SAML Raider design decision, and now shared with Tree-sitter's cgo-binding approach as Lane 6.

**Forward-references**: (append future per-tool refinement plans here as they're created)

---

## CONTEXT REFERENCES

### Relevant Codebase Files — READ THESE BEFORE IMPLEMENTING

**Backend:**
- `api/sandbox_manager.go` (full file, 1086 lines) — Why: `SandboxSession.spawn()`/`handleOutput()`/`watchProcess()`/`toolEnv()` (lines 433-479) are reused verbatim by every lane that spawns a process. `SandboxSession.Pid` (line 68, set at line 363) is bwrap's own PID — the namespace-join target for Phase 0. `buildBwrapArgs` (lines 398-428) is the one function Lane 4 (proxychains-ng) modifies. The Proxy handlers (558-816) are the pattern to copy for any tool needing a live event feed rather than one-shot output, and also the flow source Phase 3's SAML Raider composes with.
- `api/sandbox_deps.go` (full file, 329 lines) — Why: `requiredTool`/`DependencyStatus`/`EnsureSandboxDependencies`/`installPackages` — the exact model Phase 1 extends with non-package-manager acquisition strategies. `binaryExists` (line 269) already prefers a bundled copy over PATH — new tools get this for free once added to `sandboxToolsDir()`-resolvable locations. Not consulted at all for Tree-sitter (Lane 6, compiled in) or SAML Raider (Lane 6, pure Go).
- `api/sandbox_tools.go` (full file, 89 lines) — Why: `resolveSandboxTool`/`sandboxToolsDir`/`isBundledTool`/`bundledToolsLibDir` — lane-agnostic; no changes needed here, only more binaries land in the directory it already resolves.
- `scripts/bundle-sandbox-tools.sh` (full file) — Why: `TOOLS=("bwrap" "Xvfb" "x11vnc")` array + `ldd`-walking dependency closure — Phase 1 extends this array and adds non-`ldd`-walkable acquisition (pip venvs) as separate copy steps.
- `Containerfile` (lines 28-42, the `microdnf install` block) — Why: exact insertion point for new system packages; EPEL-enable precedent (line 38) for tools that also need a non-default repo. Also the insertion point for the *build*-stage C toolchain (`gcc`) Tree-sitter's cgo binding needs — a separate concern from the runtime-stage package list.
- `gui/src/components/SandboxContext.tsx` (full file, 254 lines) — Why: `proxyFlows` state (line 77) + the WS message handler (around line 104-117) is the exact pattern `useToolStream` generalizes. This is where new lane state (`toolResults`/`toolEvents`/`toolAttach`) gets added.
- `gui/src/hooks/useSandboxSession.ts` (full file, 43 lines) — Why: the documented "narrow seam" every tool page already imports; new lane hooks (`useToolScan`, `useToolStream`, `useToolAttach`) live beside this file and follow its doc-comment-block convention.
- `gui/src/services/sandboxService.ts` — Why: REST+WS client to extend with `runTool`, `getToolResult`, `createToolWebSocket` functions, mirroring the existing session functions' shape.
- `gui/src/components/tools/staticanalysis/Semgrep.tsx` (full file, 143 lines) — Why: Phase 3 pilot target. `runScan` (line 37) is the mock to replace; `findings[]`/`Finding` interface (lines 6-23) is the real shape the backend's JSON response should match so the rest of the component needs zero changes.
- `gui/src/components/tools/staticanalysis/TreeSitter.tsx` — Why: Phase 3 Lane 6 rollout target; read its existing AST-node mock shape before wiring the cgo binding's output to it.
- `gui/src/components/tools/authaudit/SamlRaider.tsx` — Why: Phase 3 Lane 6 rollout target; read alongside `Proxy.tsx`'s flow list, since real SAML Raider reads captured flows rather than running standalone.
- `gui/src/components/tools/instrumentation/ProxychainsNg.tsx` (full file) — Why: Phase 4 pilot target; read alongside `Bubblewrap.tsx`'s `command` preview pattern (`Bubblewrap.tsx:55-67`) since proxychains-ng's real integration is a modification to that same generated bwrap command, not a separate process.
- `gui/src/components/tools/instrumentation/Frida.tsx` (full file, 149 lines) — Why: Phase 6 (Lane 3) target; `script`/`log`/`attachedPid` state (lines 44-46) map directly onto the RPC bridge's expected inputs/outputs.
- `gui/src/components/tools/reversing/Cutter.tsx` — Why: Phase 7 (Lane 5) target — inherits the same `functions[]`/`decompiled{}`/`listing{}`-shaped mock data `Ghidra.tsx` used to define (both mock UIs converged on the same shape independently, which is itself a small piece of evidence the two tools were genuinely redundant). `Ghidra.tsx` is read one last time in Phase 7 only to confirm the shape before it's deleted.

### New Files to Create

**Backend:**
- `api/sandbox_tool_acquire.go` — extended dependency-acquisition strategies (pip, GitHub release) alongside the existing package-manager one in `sandbox_deps.go`.
- `api/sandbox_tool_scan.go` — Lane 1 generic handler (`RunScan`, one-shot spawn + JSON/text capture + REST response).
- `api/sandbox_tool_native.go` — Lane 6 generic handler: identical REST shape to `sandbox_tool_scan.go`'s `/run` endpoint, but calls an in-process Go function instead of spawning a subprocess. Used by Tree-sitter and SAML Raider.
- `api/sandbox_tool_stream.go` — Lane 2 generic handler (start/stop long-lived subprocess, fan out lines over a per-tool WebSocket, mirrors `handleOutput`/`broadcastToClients`).
- `api/sandbox_tool_nsjoin.go` — Phase 0's namespace-join helper. Confirmed shape: `SandboxSession.spawnJoined(name string, args ...string)`, building `tools/nsenter -t <InnerPid> -m -p -i -u -n -C -- name args...`; packaging grants that helper its restricted capability set. Used by every Lane 2/3/4 spawn.
- `api/sandbox_tool_semgrep.go`, `api/sandbox_tool_jadx.go`, `api/sandbox_tool_ilspy.go`, `api/sandbox_tool_jwttool.go` — Lane 1 tools (thin: argv-building + output-shape parsing only, all logic reused from `sandbox_tool_scan.go`).
- `api/sandbox_tool_treesitter.go`, `api/sandbox_tool_samlraider.go` — Lane 6 tools (thin: call into a Go package/cgo binding directly, all REST plumbing reused from `sandbox_tool_native.go`).
- `api/sandbox_tool_proxychains.go` — Lane 4 (`buildBwrapArgs` wrapper).
- `api/sandbox_tool_bpftrace.go`, `api/sandbox_tool_tshark.go`, `api/sandbox_tool_zeek.go`, `api/sandbox_tool_afl.go` — Lane 2 tools.
- `api/sandbox_tool_frida.go` — Lane 3 (RPC bridge subprocess management).
- `api/sandbox_tool_cutter.go` — Lane 5 (rizin headless invocation + JSON export parsing).
- Corresponding `*_test.go` for each, mirroring `api/sandbox_manager_test.go`'s style.

**Frontend:**
- `gui/src/hooks/useToolScan.ts` — Lane 1 **and** Lane 6 hook (`{ result, running, run() }`) — both lanes are "request in, structured result back," differing only in whether the backend spawned a process or called a function, which the frontend never needs to know.
- `gui/src/hooks/useToolStream.ts` — Lane 2 hook (`{ events, running, start(), stop() }`).
- `gui/src/hooks/useToolAttach.ts` — Lane 3 hook (`{ attached, log, send(msg) }`), Frida-specific but written generically in case a second RPC-style tool ever lands.
- `gui/src/services/sandboxToolService.ts` — REST+WS client for the `/api/v1/sandboxes/{id}/tools/*` surface (kept separate from `sandboxService.ts`, which stays session-lifecycle-only).

### Patterns to Follow

**Generic tool REST shape** (mirrors `sandbox_manager.go`'s `RegisterHandlers`):
```go
router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/run", handler).Methods("POST")   // Lane 1 & Lane 6
router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/start", handler).Methods("POST") // Lane 2/3
router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/stop", handler).Methods("POST")  // Lane 2/3
router.HandleFunc("/api/v1/sandboxes/{id}/tools/{tool}/ws", handler)                    // Lane 2/3 event stream
```

**Namespace-joined spawn — CONFIRMED working shape (Phase 0, executed twice live, second run root-verified by the user).** Two fixes versus the original draft: (1) join the **inner** bwrap PID (`SandboxSession.InnerPid`, resolved once at `Start()` time via `ps --ppid s.Pid`), not `s.Pid` itself — `s.Pid` is bwrap's outer setup process and stays in the host's namespaces; (2) run the bundled `nsenter` helper directly, skipping `-U`/`--user` entirely. Packaging grants that helper only `CAP_SYS_ADMIN`, `CAP_SYS_PTRACE`, and `CAP_SYS_CHROOT`, which is sufficient to `setns()` into every namespace the sandbox owns (mnt/pid/ipc/uts/cgroup all confirmed; `net`/`time` are shared with the host anyway, joining them is a no-op). The Electron host and long-lived engine remain the logged-in user.
```go
func (s *SandboxSession) spawnJoined(name string, args ...string) (*exec.Cmd, error) {
    full := append([]string{"-t", strconv.Itoa(s.InnerPid), "-m", "-p", "-i", "-u", "-n", "-C", "--", name}, args...)
    return s.spawn(resolveSandboxTool("nsenter"), full...)
}
```
Verified live: `sudo nsenter -t <innerPid> -m -p -i -u -n -C -- sh -c 'id; ls /; ps -ef'` returned `uid=0` (root's real identity — no `-U` means no identity confusion), `ls /` showing only the sandboxed rootfs (`bin dev lib lib64 proc tmp usr`), and `ps -ef` showing exactly the sandbox's own process tree (`bwrap` as pid 1, the target as pid 2, the joined shell itself appearing as a new pid 3 inside that same pid namespace) — including `cgroup`, which had failed even in every unprivileged attempt. **Implementation decision — settled:** KNIRVENGINE itself, including Electron and the long-lived Go backend, always runs as the logged-in user. `spawnJoined` invokes the bundled, capability-carrying `nsenter` helper directly; only package-manager acquisition uses `sudo -n --` as a rare setup operation.

**Lane 6 in-process call** (Phase 2/3, used by Tree-sitter and SAML Raider instead of `spawn()`):
```go
// Same REST envelope as Lane 1's /run, no subprocess involved.
func (m *SandboxManager) handleNativeRun(tool string, run func(s *SandboxSession, args json.RawMessage) (interface{}, error)) http.HandlerFunc {
    // decode {id}, look up session, call run(s, body), RespondWithSuccess(w, result, "")
}
```

**Dependency acquisition strategy interface** (Phase 1, extending `requiredTool` in `sandbox_deps.go`):
```go
type acquireStrategy interface {
    present(binary string) bool
    install(binary string, runner commandRunner) error
    manualCommand(binary string) string
}
// packageManagerStrategy{} (existing behavior, refactored out of EnsureSandboxDependencies)
// pipStrategy{}
// githubReleaseStrategy{repo, assetPattern string}
```

**Response envelope**: every new handler uses `api.RespondWithSuccess`/`RespondWithList` from `api/standard_response.go`, matching `sandbox_manager.go`'s own handlers — no raw `json.NewEncoder`.

**Naming**: `api/sandbox_tool_<toolname>.go`, one file per tool, each exporting exactly one `register<ToolName>Routes(router *mux.Router, m *SandboxManager)`-style function called from `sandbox_manager.go`'s `RegisterHandlers`.

---

## ARCHITECTURE

### The 6 execution lanes

This is the plan's central design decision — everything else follows from correctly bucketing each tool. (Lane 6 is new in this revision, formed by merging Tree-sitter's confirmed cgo-binding approach with SAML Raider's confirmed native-Go approach — both are "no external process at all," just one links a C library via cgo and the other is pure Go.)

| Lane | Behavior | Tools | Namespace join needed? |
|---|---|---|---|
| **1 — Batch scan (subprocess)** | Spawn once, read the mounted target's files, produce a JSON/text report, exit. Only needs the target's files, already available via the existing `binds` bind-mounts. | Semgrep, Jadx, ILSpy, jwt_tool | **No** — operates on files, not the live process |
| **2 — Streaming daemon** | Spawn once, stays alive, continuously emits structured events (packets, probe hits, fuzzer stats) fanned out over a WS, stopped explicitly. | bpftrace, tshark (Wireshark), Zeek, AFL++ | **Yes** — must observe the live sandboxed process/netns |
| **3 — Attach + RPC** | Spawn once, stays alive, but also accepts *commands* from the frontend (load script, evaluate expression) and returns *responses*, not just a one-way log — bidirectional. | Frida | **Yes**, plus no native Go binding (subprocess bridge) |
| **4 — Launch modifier** | Doesn't run as a separate process at all — changes the argv `bwrap` execs so the *target itself* runs wrapped. | proxychains-ng | N/A — baked into the existing bwrap launch, no new process |
| **5 — Headless-with-native-UI** | GUI tool with a documented headless/CLI mode that can export structured data; KNIRVENGINE renders that data in its own React UI (already built) instead of showing the tool's real window. | Cutter (the `rizin` binary itself, run headlessly — see Phase 7's rz-ghidra note; not `rz-bin`, a different, narrower rizin-suite utility) | **No** for the headless path (files only, like Lane 1) |
| **6 — Native Go (in-process, no external binary)** | No subprocess at all — the logic runs compiled into the KNIRVENGINE backend itself. Same one-shot request/response shape as Lane 1 from the frontend's perspective. | Tree-sitter (cgo bindings to the C parser library), SAML Raider (pure Go XML/SAML mutation, composes with Proxy's captured flows) | **No** |

Frontend consequence: **3 hooks, not 13.** `useToolScan('semgrep', args)` (also used for every Lane 6 tool) / `useToolStream('bpftrace', script)` / `useToolAttach('frida', pid)` are the only three data-fetching shapes any of the 13 pages ever need.

### Phase 0: the namespace-join question — RESOLVED (executed live, this revision)

**Executed against a real `SandboxSession`** (real `bwrap`/`Xvfb`/`x11vnc`, via `SandboxManager.CreateSession`/`Start()` directly — not a hand-rolled `bwrap` invocation) on the actual dev host (Kali, kernel 5.15, util-linux 2.37.2, unprivileged user, no passwordless sudo), across two independent session launches for reproducibility. Findings:

**1. `SandboxSession.Pid` is NOT the sandboxed target — it's the outer setup process, still in the HOST's namespaces.** `bwrap` forks a child that re-execs itself ("inner bwrap") to actually perform `--unshare-all` and become PID 1 of the new pid namespace; the real target (`sleep 300` in the spike) is *that* process's child. Confirmed twice:
```
outer bwrap (session.Pid)  — mnt/net/pid/user namespaces IDENTICAL to the host's (unshared nothing)
  └─ inner bwrap (session.Pid's only child) — NEW mnt/pid/ipc/uts/user/cgroup namespaces, PID 1 of the new pidns
       └─ target command (e.g. sleep 300) — NS-local PID 2
```
**Consequence for `spawnJoined()`**: it must resolve `ps --ppid <SandboxSession.Pid>` (or read `/proc/<Pid>/task/.../children`) to find the inner bwrap PID at join time — `session.Pid` itself is useless as an `nsenter` target. `SandboxManager` should track this child PID on the session going forward (a new field, e.g. `SandboxSession.InnerPid`) rather than re-resolving it on every tool launch.

**2. `--share-net` genuinely shares the host's network namespace — confirmed, not hypothetical.** The inner bwrap's `net` namespace inode was identical to the host's own (`net:[4026531840]`) on both runs. **This independently resolves the Phase 5 open question about Wireshark/Zeek's capture scope**: there is no dedicated per-session veth today; a `tshark`/`Zeek` capture attached to this session would see all host traffic, not target-scoped traffic. The networking redesign flagged as an open question in Phase 5 is now a confirmed requirement, not a maybe.

**3. Unprivileged `nsenter` can join the sandbox's `user` namespace alone (we own it — bwrap identity-maps the real UID, `uid_map` showed `1000 1000 1`) — but cannot join any of mnt/pid/ipc/uts/cgroup, with or without `-U --preserve-credentials`, and regardless of ordering:**
```
nsenter -t <innerPid> -m -p --                        → Operation not permitted (fails at the first requested ns)
nsenter -t <innerPid> -U --preserve-credentials -- id  → SUCCEEDS (uid=1000(gperry) ...)
nsenter -t <innerPid> -U --preserve-credentials -m -p -i -u -- ...   → still fails (at whichever ns util-linux attempts first)
nsenter -t <innerPid> -U --preserve-credentials -- nsenter -t <innerPid> -m -p -- ...  → still fails (two-stage workaround also fails)
```
This reproduced identically across both sessions and multiple namespace-flag combinations. It is a real util-linux/kernel-ordering limitation, not a fluke: joining the owning user namespace via `setns()` does not reliably carry forward the resulting in-namespace capabilities to a *subsequent* `setns()` call for another namespace type owned by it, at least not through `nsenter`'s call sequence on this kernel/util-linux version. **Plain unprivileged `nsenter` — the mechanism this plan originally assumed `spawnJoined()` would use — does not work.**

**Decision gate outcome — RESOLVED: Option A, confirmed live.** Two spikes were run: the first (unprivileged) established that plain `nsenter` categorically fails on mnt/pid/ipc/uts/cgroup; the second re-ran the identical join **as root via `sudo`**, and it succeeded cleanly on every namespace, including `cgroup` (the one that had failed even in combination attempts unprivileged). Evidence:
```
$ sudo nsenter -t <innerPid> -m -p -i -u -n -C -- sh -c 'id; ls /; ps -ef; hostname'
id: uid=0 gid=0 groups=0,108,143
--- / ---
bin  dev  lib  lib64  proc  tmp  usr                    # the sandbox's rootfs, not the host's
--- ps -ef ---
UID   PID  PPID  ...  CMD
1000    1     0  ...  bwrap --ro-bind /usr /usr --ro-bind ...   # inner bwrap, pid 1 of the sandbox's pidns
1000    2     1  ...  sleep 300                                # the real sandboxed target
   0    3     0  ...  sh -c ...                                # our own joined shell — now inside the sandbox's pidns too
   0    6     3  ...  ps -ef
--- hostname ---
cloud-eq
```
This requires root capability only on the namespace-entry helper, not for KNIRVENGINE as a whole. The Electron host and its backend must remain unprivileged because Electron refuses to run as root with its sandbox enabled.

**Options B and C are no longer needed and are dropped from further consideration** — they were fallbacks for a scenario (root escalation unavailable or unwanted) that doesn't apply here. `spawnJoined()`'s confirmed shape is documented in "Patterns to Follow" above. The packaged `nsenter` capability set supplies routine attach privileges; the engine must never run entirely as root because that would make Electron refuse to start.

### Extending dependency acquisition (Phase 1)

`sandbox_deps.go` today hardcodes "package manager only." The 13 remaining tools split across four acquisition strategies. The table below is the complete, per-tool bundling specification — every tool that needs a runtime binary has one named row; the four strategy categories from earlier revisions of this plan are still the right grouping, but grouping alone wasn't a complete implementation guide, since two tools (ILSpy, Jadx) don't cleanly fit any of the three subprocess-based strategies as originally scoped, and two (Cutter, Frida) actually need *two* different strategies each for their different binaries. This section replaces the earlier grouped-only table.

**System package manager strategy** (mirrors `requiredTool.packages` already used for bwrap/Xvfb/x11vnc — `binary → {apt-get, dnf, microdnf, yum, pacman, zypper, apk}`):

| Tool | Binary | apt-get | dnf/microdnf/yum | pacman | zypper | apk | Notes |
|---|---|---|---|---|---|---|---|
| proxychains-ng | `proxychains4` | `proxychains4` | `proxychains-ng` | `proxychains4` | `proxychains-ng` | `proxychains-ng` | package name diverges from the binary name on most managers — don't assume `proxychains4` package resolves everywhere |
| bpftrace | `bpftrace` | `bpftrace` | `bpftrace` | `bpftrace` | `bpftrace` | `bpftrace` | same name everywhere; also needs `CAP_SYS_ADMIN`/BPF at *runtime*, already covered by the Phase 0 root/`sudo` decision, not a separate acquisition concern |
| Wireshark | `tshark` | `tshark` | `wireshark-cli` | `wireshark-cli` | `wireshark` | `tshark` | package name is NOT `tshark` on RHEL/Fedora/Arch — it's bundled inside the `wireshark-cli`/`wireshark` package there |
| Zeek | `zeek` | `zeek` | `zeek` | `zeek` | `zeek` | `zeek` | **not in default apt/dnf repos on most distros** — needs the Zeek project's own repo added first (`download.opensuse.org/repositories/security:zeek`), same shape as the existing EPEL-enable precedent for `x11vnc` (`enableExtraRepos` in `sandbox_deps.go`) — extend that function, don't write a new one |
| AFL++ | `afl-fuzz` | `afl++` | `aflplusplus` | `aflplusplus` | `aflplusplus` | `afl++` | package name also diverges from the binary name |
| Cutter (engine) | `rizin` | `rizin` | `rizin` | `rizin` | `rizin` | `rizin` | this is the binary Phase 7's headless commands (`aflj`, `pdgj`) actually run against — **not** `rz-bin`, which is a narrower binary-metadata utility (imports/exports/sections, no scripting/analysis commands). If a build environment's bundling step only stages `rz-bin`, Phase 7 will not work — confirm `rizin` itself is what's staged, `rz-bin` at most as a secondary convenience |

**pip strategy** (managed venv under `<sandboxToolsDir()>/pyenv/`, not system Python — Phase 1 task):

| Tool | Binary | pip module | Notes |
|---|---|---|---|
| Semgrep | `semgrep` | `semgrep` | binary name == module name |
| jwt_tool | `jwt_tool.py` | `jwt_tool` (PyPI) | PyPI package is a thin wrapper; the actively-maintained tool is `ticarpi/jwt_tool` on GitHub — if the PyPI package lags, fall back to a `githubReleaseStrategy`-style `git clone` into the venv instead (flag as a fallback path, not the default) |
| Frida (CLI half only) | `frida` | `frida-tools` | **note the module/binary name mismatch** — `pip install frida-tools` provides the `frida` command; installing a `frida` module by that literal name is a different, unrelated package |

**GitHub release strategy** (`githubReleaseStrategy{repo, assetPattern}`, downloaded at build time into `sandboxToolsDir()`):

| Tool | Binary | Repo | Asset pattern (illustrative) | Notes |
|---|---|---|---|---|
| Jadx | `jadx` | `skylot/jadx` | `jadx-\d+\.\d+\.\d+\.zip` | **Jadx needs a JRE at runtime — this is a gap, not yet addressed anywhere in this plan.** Either (a) require a host-provided JRE and add it to the missing-deps check same as any package-manager tool (`java` binary, `default-jre`/`java-*-openjdk` across managers), or (b) bundle a minimal JRE alongside the JAR (adds real size to the shipped bundle — a Ghidra-sized tradeoff for one tool, worth avoiding). **Recommendation: (a)**, treat `java` as its own package-manager-strategy dependency row, since it's a much smaller, more commonly-already-present dependency than a bundled JRE. |
| Frida (agent half only) | `frida-server` | `frida/frida` | `frida-server-[\d.]+-linux-x86_64\.xz` (arch-specific — also need an `arm64` pattern if shipping for that arch) | **This is a second, distinct acquisition path from the `frida-tools` pip install above** — `frida-server` is the in-target agent binary that must run *inside* the sandbox's namespaces (per Phase 6), not a Python package. Conflating the two in a single "Frida" dependency-status row would hide a real failure mode (CLI present, agent missing, or vice versa) — give them separate `DependencyStatus` rows. |

**dotnet global tool strategy — NOT YET A REAL STRATEGY, this is the one true gap in the acquisition-strategy design itself:**

| Tool | Binary | Notes |
|---|---|---|
| ILSpy | `ilspycmd` | Acquired via `dotnet tool install --tool-path <dir> ilspycmd`, not `apt`/`pip`/a GitHub release download — none of the three `acquireStrategy` implementations designed so far (`packageManagerStrategy`, `pipStrategy`, `githubReleaseStrategy`) model "install via a language-specific global-tool command into a managed directory," even though that's structurally almost identical to `pipStrategy`'s shape (managed dir, one install command, resolve the binary inside it after). **Add a fourth `dotnetToolStrategy` to Phase 1**, mirroring `pipStrategy` exactly (`dotnet tool install --tool-path <sandboxToolsDir()>/dotnettools ilspycmd` instead of `pip install`) — this also needs the .NET runtime itself present on the build/runtime host, which is its own package-manager-strategy row (`dotnet-runtime-8.0` or equivalent, varies more by distro than most packages here). |

**Compiled in — no runtime acquisition at all** (Lane 6):

| Tool | Mechanism | Notes |
|---|---|---|
| Tree-sitter | `go-tree-sitter` cgo binding, `go.mod` dependency | Only requirement is a C toolchain (`gcc`) in the **build** environment (dev machine, CI, `Containerfile`'s *build* stage) — never appears in `DependencyStatus`/the missing-deps banner, since there's nothing to check at runtime |
| SAML Raider | Pure Go (`encoding/xml`), no external dependency | Same as Tree-sitter — no acquisition step of any kind |

Every subprocess-based strategy still reports through the same `DependencyStatus` struct (`sandbox_deps.go:15-21`) so `Bubblewrap.tsx`'s existing missing-deps banner needs **zero frontend changes** for any row above — this is why Phase 1 is worth doing generically before any Lane work starts. Lane 6 tools never appear in that banner at all.

### Frontend data flow (generalizing `proxyFlows`)

```
SandboxContext (extends existing session WS handler)
  │
  ├─ toolResults: Record<string, ToolScanResult>     (Lane 1 & Lane 6 — set once per run() call, via REST, not WS)
  ├─ toolEvents:  Record<string, ToolEvent[]>         (Lane 2/3 — appended by a per-tool WS, same pattern as proxyFlows)
  └─ toolAttach:  Record<string, ToolAttachState>     (Lane 3 — request/response correlation over the same WS)
        │
        ▼
useToolScan / useToolStream / useToolAttach  (gui/src/hooks/)
        │
        ▼
Semgrep.tsx / TreeSitter.tsx / Bpftrace.tsx / Frida.tsx / ... (existing UI shells, only the data source changes)
```

### Recommended Build Order

Below noVNC, keeping Dashboard/Sandbox fixed. Ordered by **dependency and risk**, not raw nav position: items 0–7 need nothing new architecturally (they reuse Lane 1/6's batch/native plumbing or are a pure argv change); everything from item 8 onward is gated on Phase 0's namespace-join spike being resolved first. This table is the authoritative *sequencing* layer — the numbered Phases below group work by *technique* (useful when implementing a given lane), while this table is the actual recommended order to work through the tools, cutting across phases. Building Phase 3's SAML Raider before Phase 4's proxychains-ng (as this table does) is expected and fine.

| # | Tool | Category | Lane | Phase | Why here |
|---|---|---|---|---|---|
| 0 | **Proxy** (finish intercept/hold/drop/replay) | Proxy | — | (gap-fill) | Already ~90% real — cheapest possible win, closes out an existing tool before opening 13 new ones. |
| 1 | **Semgrep** | Static Analysis | 1 | 3 | Pilot — its mock `Finding` shape already matches Semgrep's real JSON almost field-for-field, so it proves the generic scan plumbing without also fighting an output-format design problem. |
| 2 | **Tree-sitter** | Static Analysis | 6 | 3 | Trivial once #1's REST envelope exists; cgo binding confirmed over a CLI subprocess — no acquisition step, no bundling. |
| 3 | **Jadx** | Reversing | 1 | 3 | Single JAR, own format (Android DEX) — no dependency on anything else in Reversing. |
| 4 | **ILSpy** | Reversing | 1 | 3 | Same pattern, dotnet-tool acquisition, own format (.NET IL). |
| 5 | **jwt_tool** | Auth Audit | 1 | 3 | Small; naturally consumes tokens already visible in Proxy's captured flows — validates that Proxy and Auth Audit compose. |
| 6 | **SAML Raider** (native Go rebuild) | Auth Audit | 6 | 3 | No process spawning at all — reimplemented like Proxy itself, and also consumes Proxy's captured flows. Doesn't need Phase 0, so it goes early even though it's new code, not a wrapped binary. Moved here from the plan's earlier standalone "Phase 8" — see Amendments. |
| 7 | **proxychains-ng** | Instrumentation | 4 | 4 | A `bwrap` launch-argv change, not a new process. Last "no namespace-join needed" tool; closes out that whole tier. |
| — | *Phase 0 gate* | — | — | — | **Nothing below this line starts until the `nsenter`-into-the-running-sandbox question is empirically confirmed** on the real container. |
| 8 | **bpftrace** | Instrumentation | 2 | 5 | Pilot — first tool to prove namespace-join + live WS streaming actually works end-to-end. |
| 9 | **Wireshark (tshark)** | Packet Capture | 2 | 5 | Reuses bpftrace's proven join+stream pattern directly. |
| 10 | **Zeek** | Packet Capture | 2 | 5 | Same capture source as #9 — do them back-to-back since Zeek's netns-capture-scope question is identical to tshark's. |
| 11 | **AFL++** | Fuzzing | 2 | 5 | A variant (polled `fuzzer_stats` file instead of streamed stdout) — do after simpler Lane 2 tools are proven. |
| 12 | **Cutter (rizin)** | Reversing | 5 | 7 | Headless, replacing Ghidra — closer to Lane 1 in spirit but benefits from Lane 2's process-output maturity for larger binaries. |
| 13 | **Frida** | Instrumentation | 3 | 6 | Hardest — no Go binding, needs an RPC bridge subprocess, and `frida-server` itself must run inside the sandbox's namespaces. Last for a reason. |

---

## IMPLEMENTATION PLAN

### Phase 0: Namespace-Join Spike — RESOLVED this revision (Option A confirmed)

**Tasks:**
- ~~Launch a real `SandboxSession` in dev, run the `ps`/`nsenter` probe commands from ARCHITECTURE above against the live `bwrap` PID.~~ **Done** — executed three times, live, via `SandboxManager.CreateSession`/`Start()` directly (real `bwrap`/`Xvfb`/`x11vnc`, not stubbed), on the dev host (Kali, kernel 5.15, util-linux 2.37.2): once establishing the outer-vs-inner-PID and `--share-net` findings, once (unprivileged) proving plain `nsenter` fails on mnt/pid/ipc/uts/cgroup, and once **as root via `sudo`, which succeeded cleanly on every namespace** — user-run and pasted back with full output. See ARCHITECTURE's Phase 0 write-up for the complete evidence trail.
- **Decision gate — RESOLVED: capability-based Option A.** The live root spike proves the required kernel permissions; the packaged `nsenter` helper carries precisely those permissions, while Electron and the engine remain unprivileged.
- Write the confirmed mechanism into `api/sandbox_tool_nsjoin.go`'s doc comment when that file is created in Phase 2 — the shape is documented in "Patterns to Follow" above (`spawnJoined()` via the capability-carrying bundled `nsenter -t <InnerPid> -m -p -i -u -n -C -- ...`, no `-U`).
- Add `SandboxSession.InnerPid int` (resolved once, right after `bwrapCmd.Process.Pid` is captured, via `ps --ppid <s.Pid>`) — every Lane 2/3/4 tool needs this, not `s.Pid`.
- **Still outstanding, non-blocking**: confirm on the target production distro (UBI9-minimal container, per `Containerfile`) before Phase 2 ships there — this spike ran on a dev-machine Kali install; container `nsenter`/file-capability posture (and whether the container runtime must grant `SYS_ADMIN`/`SYS_PTRACE` in addition to the in-image file capability) can differ. Re-run the same spike there before trusting the dev-host result in production.

### Phase 1: Generalized Dependency Acquisition

**Depends on:** nothing (parallel to Phase 0).

**Tasks:**
- Refactor `EnsureSandboxDependencies` (`sandbox_deps.go:215-265`) to dispatch through an `acquireStrategy` interface instead of assuming package-manager-only; the existing package-manager code becomes `packageManagerStrategy{}`, behavior-identical for bwrap/Xvfb/x11vnc (regression risk here — this refactor must not change Bubblewrap's existing dependency banner behavior).
- Add `pipStrategy{}` — installs into a dedicated venv at `<sandboxToolsDir()>/pyenv/`, not the system Python, so it doesn't require root and doesn't pollute the host.
- Add `githubReleaseStrategy{repo, assetPattern}` — downloads a release tarball/binary matching the current OS/arch, extracts into `sandboxToolsDir()`, chmod +x, same as a bundled binary today. Two tools need this: Jadx (`skylot/jadx`) and `frida-server` (`frida/frida`, arch-specific asset pattern — see the per-tool table above).
- Add `dotnetToolStrategy{}` — the fourth strategy the per-tool table above identified as missing. Mirrors `pipStrategy` exactly: `dotnet tool install --tool-path <sandboxToolsDir()>/dotnettools ilspycmd`, then resolve the binary from that managed directory. Only tool that needs it today: ILSpy (`ilspycmd`).
- Add a `java` (JRE) dependency row via the existing package-manager strategy — Jadx needs a JRE present at runtime; per the per-tool table above, requiring a host JRE (small, commonly already present) is recommended over bundling one (large, Ghidra-sized tradeoff for a single tool).
- Give `frida-server` its own `DependencyStatus` row, separate from the `frida`/`frida-tools` pip-installed CLI — they're two different binaries acquired two different ways (pip vs. GitHub release), and collapsing them into one status would hide which half is actually missing.
- Extend `enableExtraRepos` (`sandbox_deps.go`, the existing EPEL-for-x11vnc precedent) to also add the Zeek project's own repo before attempting to install `zeek` — it is not in default apt/dnf repos on most distros, same class of problem `x11vnc` already had.
- Keep the acquisition-strategy interface open to a future `manualOnly` marker (not needed by any tool in this revision's final 13, now that Ghidra is cut, but worth leaving the extension point in place rather than removing it — some future tool may still need it).
- Add `github.com/tree-sitter/go-tree-sitter` (or equivalent) to `go.mod` for Tree-sitter — this is a **build-time** `go.mod` change, not a `sandbox_deps.go` entry; confirm `gcc` (or equivalent C toolchain) is available in dev, CI, and the `Containerfile`'s **build** stage specifically (the runtime stage doesn't need it — the cgo code is compiled into the final binary).
- Extend `scripts/bundle-sandbox-tools.sh`'s `TOOLS` array and add non-`ldd` copy steps (venv directory copy, `dotnettools` directory copy) for build-time bundling into the shipped `dist/tools/`. Confirm the array stages `rizin` itself, not only `rz-bin` (see the per-tool table above — `rz-bin` alone cannot run Phase 7's headless analysis commands).
- `Containerfile`: add the system-package-manager-strategy tools (including `rizin`, `java`/JRE, and Zeek's repo-enable step) to the existing `microdnf install` block (same pattern as bwrap/Xvfb/x11vnc, lines 38-42); confirm the build stage already has a C toolchain for Tree-sitter's cgo build, add one if not; confirm a .NET runtime is available for `dotnetToolStrategy`'s `dotnet tool install` step to even run.
- **VALIDATE**: `go test ./api/... -run TestEnsureSandboxDependencies -v` (extend `sandbox_deps_test.go` with cases per new strategy, stubbing network/filesystem the same way the existing tests stub `commandRunner`).

### Phase 2: Generic Tool Backend (Lane 1, Lane 2 & Lane 6 plumbing)

**Depends on:** Phase 0 (for Lane 2's `spawnJoined`), Phase 1 (for dependency reporting).

**Tasks:**
- `api/sandbox_tool_nsjoin.go`: `SandboxSession.spawnJoined(name string, args ...string)` per Phase 0's confirmed answer (capability-carrying `tools/nsenter -t s.InnerPid -m -p -i -u -n -C -- name args...`); also add `SandboxSession.InnerPid int`, resolved once at `Start()` time via `ps --ppid s.Pid`.
- `api/sandbox_tool_scan.go`: generic Lane 1 handler — `POST /api/v1/sandboxes/{id}/tools/{tool}/run`, spawns via the existing `s.spawn()` (no namespace join needed per ARCHITECTURE), captures combined stdout, returns via `RespondWithSuccess`. Tool-specific argv building and output parsing are injected per tool (small per-file adapter, not copy-pasted plumbing).
- `api/sandbox_tool_native.go`: generic Lane 6 handler — same `POST .../run` REST shape as Lane 1, but the per-tool adapter is a Go function call, not a spawned command. Shares response envelope and error handling with `sandbox_tool_scan.go` so the frontend truly cannot tell Lane 1 and Lane 6 apart.
- `api/sandbox_tool_stream.go`: generic Lane 2 handler — `POST .../start` / `POST .../stop` / `GET .../ws`, using `spawnJoined()` + a per-tool-per-session `clients map[*websocket.Conn]bool` (mirror `SandboxSession`'s own client map for status WS) fanning out parsed event lines.
- Wire all three generic handlers' route registration into `SandboxManager.RegisterHandlers` (`sandbox_manager.go`'s existing registration block).
- `gui/src/services/sandboxToolService.ts`: `runTool(sessionId, tool, args)`, `startTool`, `stopTool`, `createToolWebSocket(sessionId, tool)`.
- `gui/src/hooks/useToolScan.ts`, `gui/src/hooks/useToolStream.ts`.
- **VALIDATE**: `go build ./... && go test ./api/... -run TestSandboxTool -v`; `cd gui && npx tsc --noEmit`.

### Phase 3: Lane 1 & Lane 6 Rollout — Semgrep, then Tree-sitter, Jadx, ILSpy, jwt_tool, SAML Raider

**Depends on:** Phase 2.

**Pilot tasks (Semgrep, full detail):**
- `api/sandbox_tool_semgrep.go`: argv = `semgrep --config <ruleset> --json <mountedTargetDir>`; parse Semgrep's JSON `results[]` into the `Finding` shape `Semgrep.tsx:6-13` already defines (ruleId/severity/file/line/message) — **the frontend interface was designed against the real tool's output shape already**, so this is a direct mapping, not a redesign.
- `Semgrep.tsx`: replace `runScan`'s `setTimeout` (line 37-40) with `useToolScan('semgrep', { ruleset })`; delete the hardcoded `findings[]` array; everything else in the component (severity filter, selected-finding detail pane) is unchanged since it already operates on a `Finding[]`.
- **VALIDATE**: manual — run Semgrep against a real mounted project with a known MD5 usage, confirm the finding appears with the real file/line, not the seeded `f1`.

**Rollout tasks, per the Recommended Build Order (one `api/sandbox_tool_<name>.go` + one hook call site each):**
- **Tree-sitter** (Lane 6, cgo binding, confirmed): call the `go-tree-sitter` binding directly against the target file's content already available via the mounted bind; parse the resulting tree into `TreeSitter.tsx`'s existing AST node shape. No argv, no subprocess, no `DependencyStatus` entry.
- Jadx: argv `jadx --export-gradle -d <outDir> <apk>`, or `-r` for a quick class listing depending on what `Jadx.tsx`'s mock UI actually renders — read the component first, output parsing must match its existing shape.
- ILSpy: argv `ilspycmd <assembly.dll> -o <outDir>`.
- jwt_tool: argv `jwt_tool.py <token> -M at` (or the specific mode `JwtTool.tsx`'s mock buttons imply) — read the component's existing action buttons before choosing modes, since jwt_tool has many.
- **SAML Raider** (Lane 6, native Go, confirmed — moved into this phase from the plan's earlier "Phase 8"): `api/sandbox_tool_samlraider.go` parses/mutates/re-encodes SAML `AuthnRequest`/`Response` XML natively (`encoding/xml` + a signature-mutation helper), operating on flows the already-real Proxy capture surfaces (SAML traffic is just HTTP through the existing embedded proxy) — the one tool in the whole plan that composes with Proxy rather than needing its own process at all. `SamlRaider.tsx` wires to real captured SAML flows instead of a fixture.
- **Cleanup**: delete `TruffleHog.tsx` and its route/nav card in `StaticAnalysis.tsx` — its function is folded into Semgrep's `p/secrets` ruleset per the Consolidation Audit.
- **VALIDATE per tool**: `go test ./api/... -run TestSandboxToolLane1 -v` (and `TestSandboxToolLane6` for Tree-sitter/SAML Raider); manual run against a real sample.

### Phase 4: Lane 4 — proxychains-ng

**Depends on:** Phase 1 only (no namespace-join, no new process type).

**Tasks:**
- `api/sandbox_tool_proxychains.go`: generates a `proxychains.conf` from the chain list `ProxychainsNg.tsx` already collects in local state, writes it into the session's mount, and — the actual integration point — modifies `buildBwrapArgs` (`sandbox_manager.go:398-428`) to prepend `proxychains4 -f <conf-path>` to `s.TargetCommand` **only when a proxychains config is attached to the session**, gated behind a new `SandboxSession.proxychainsConf string` field.
- **GOTCHA**: this must be applied at launch time (before `bwrap` starts), not as a post-launch action — proxychains-ng only redirects sockets of the process it directly wraps and its children, so it has to be baked into the original exec, unlike every other Lane which attaches after the fact. This is *why* it's its own lane.
- `ProxychainsNg.tsx`: wire the existing chain-editing UI to set `sandbox.proxychainsConf` in `SandboxContext` *before* `launch()` is called (likely surfaced as an option on the Bubblewrap launch form itself, not a separate "run" action, since there's nothing to run once the target is already launched wrapped).
- **VALIDATE**: manual — launch a target with a local SOCKS proxy in the chain, confirm its outbound connections egress through that proxy (observable via the already-real Proxy tool, nicely closing the loop).

### Phase 5: Lane 2 — bpftrace, then roll out Wireshark/Zeek/AFL++

**Depends on:** Phase 0 (namespace join confirmed and working), Phase 2.

**Pilot tasks (bpftrace, full detail):**
- `api/sandbox_tool_bpftrace.go`: `spawnJoined("bpftrace", "-e", script)`, parse each stdout line (bpftrace's default text output, or `-f json` if the installed version supports it — check version pinned in Phase 1) into a `{ probe, args, timestamp }` event, push onto the tool's WS.
- `Bpftrace.tsx`: replace whatever mock log/state it has with `useToolStream('bpftrace', script)`; Start/Stop buttons call `start()`/`stop()`.
- **GOTCHA**: bpftrace attaching to kernel uprobes/tracepoints typically needs `CAP_SYS_ADMIN`/root regardless of namespace-join specifics — this is a second, independent privilege requirement on top of Phase 0's nsenter question. Confirm both before this pilot is considered done, not just the join.
- **VALIDATE**: manual — trace a real syscall the sandboxed target makes, confirm events stream live in the UI.

**Rollout tasks (same generic Lane 2 pattern):**
- Wireshark (tshark): `spawnJoined("tshark", "-i", sandboxVethName, "-T", "ek")` (JSON-per-line mode) → feeds `Wireshark.tsx`'s packet list. Requires knowing the sandbox's veth interface name — likely a new field on `SandboxSession` populated when `--share-net`/netns setup happens; audit `buildBwrapArgs`'s current networking flags (`sandbox_manager.go:412-413`) to confirm a per-session veth actually exists to capture on, versus the sandbox currently just sharing the host's network namespace (`--share-net` in the current default binds — **if `--share-net` means literally sharing the host netns rather than a dedicated veth, tshark/Zeek would capture host-wide traffic, not target-scoped traffic — this needs verifying before this rollout, it's a second Phase-0-shaped spike, still open, see Open Questions**).
- Zeek: same capture source as tshark, feeding its log-tail output instead of raw packets.
- AFL++: `spawnJoined("afl-fuzz", "-i", corpusDir, "-o", outDir, "--", targetBinary)`, parse `outDir/default/fuzzer_stats` on a poll interval (not stdout — AFL's TUI isn't machine-parseable, `fuzzer_stats` is the documented machine-readable output) into `AflPlusPlus.tsx`'s stats shape.
- **Cleanup**: delete `LibAFL.tsx` and its route/nav card in `Fuzzing.tsx` — AFL++ is the sole fuzzing console per the Consolidation Audit.
- **VALIDATE per tool**: manual, live stream observed against a real running target.

### Phase 6: Lane 3 — Frida

**Depends on:** Phase 0, Phase 2, Phase 1 (frida-tools via pip strategy).

**Tasks:**
- Confirm `frida-server` (the in-target agent) can run inside the sandbox's namespaces — this is a *third* variant of the namespace question: unlike bpftrace/tshark which observe from outside, `frida-server` needs to run *as a process inside* the same PID namespace as the target to attach to it, likely meaning it must be launched via `spawnJoined` at session-start time (alongside `Xvfb`/`x11vnc` in `sandbox_manager.go`'s `Start()`), not on-demand when the operator opens the Frida page.
- `api/sandbox_tool_frida.go`: since Go has no `frida-core` binding, spawn `frida` (the Python CLI, via the pip-strategy venv from Phase 1) in a scriptable mode that accepts commands on stdin and emits JSON on stdout (Frida's own `frida-tools` REPL supports non-interactive scripting; alternatively write a small ~30-line Python bridge script using `frida-python` that reads JSON commands from stdin and writes JSON events to stdout — bundle this script, don't hand-roll protocol parsing against the human-oriented CLI output). This Python bridge process is then treated exactly like a Lane 2 stream from Go's perspective, except Go also writes to its stdin (extending `spawnJoined`/`spawn` to expose `cmd.StdinPipe()`, which today's `spawn()` doesn't need for any other lane).
- `Frida.tsx`: `useToolAttach('frida', pid)` — `attach(pid)`, `loadScript(js)`, `log` fed from the bridge's JSON events.
- **VALIDATE**: manual — attach to the real sandboxed target's PID (not a hardcoded `4821`), load a real hook script, confirm `send()` calls from the script appear in the console.

### Phase 7: Lane 5 — Cutter (rizin headless), replacing Ghidra

**Depends on:** Phase 1 (rizin acquisition), Phase 2's Lane 1 machinery (headless mode is operationally Lane 1 — batch in, JSON out).

**Tasks:**
- `api/sandbox_tool_cutter.go`: spawn `rizin` itself (headlessly, e.g. `rizin -q -c '<cmd>' <binary>`) — **not** `rz-bin`, which is a separate, narrower rizin-suite utility for reading binary metadata (imports/exports/sections) and cannot run scripting/analysis commands — with JSON-emitting analysis commands, e.g. `aflj` for a function-list JSON, and a decompilation command for the selected function; parse into the same `functions[]`/`decompiled{}`/`listing{}` shape `Cutter.tsx` inherits from the retired `Ghidra.tsx` mock.
- **Nice-to-have, not required for v1**: rizin's `rz-ghidra` plugin can optionally use Ghidra's own decompiler engine as a headless library, without needing the full Ghidra GUI/JRE application — worth evaluating if rizin's native decompiler output quality is insufficient, since it gets near-Ghidra decompilation quality without reintroducing the multi-hundred-MB manual-install burden the Consolidation Audit cut Ghidra for.
- `Cutter.tsx`: `useToolScan('cutter', { binary })`, replacing the three hardcoded objects it inherited from the old Ghidra mock shape.
- **Cleanup**: delete `Ghidra.tsx` and its route/nav card in `Reversing.tsx` — Cutter is the sole native-binary reversing console.
- **Design decision — confirmed**: headless-with-native-UI ("Implement the headless [tool] in our own UI, just as the other tool conventions follow"). This answer was given while the plan still named Ghidra as the Lane 5 tool; it is being carried forward to apply to Cutter/rizin since Cutter replaced Ghidra in this same revision. **Flagged for one more explicit confirmation** in Open Questions, since the transfer from "headless Ghidra" to "headless Cutter" is this document's inference, not a restated instruction.
- Headless mode cannot do *interactive* re-decompilation of an arbitrary newly-selected function the way a live GUI can — the original mock UI's fallback text ("no disassembly cached... headless re-analysis required") already anticipated this limitation, and it applies identically to headless rizin.
- **VALIDATE**: manual — headless-analyze a real small binary, confirm real function names/addresses (not `verify_license_token`/`0x00101f10`) appear.

### Phase 8: Packaging & Docs

**Depends on:** all prior phases landed.

**Tasks:**
- Final `Containerfile` pass — confirm every system-package-strategy tool (including `rizin`) resolves in the actual UBI9-minimal build (same validation approach `sandbox_implementation.md` used for `x11vnc`/EPEL), and confirm the build stage's C toolchain covers Tree-sitter's cgo requirement.
- `scripts/bundle-sandbox-tools.sh` — full dry run producing a `dist/tools/` that contains all bundlable binaries; document (in this file's Notes, not a new doc) the final list of what's bundled vs. compiled-in (Lane 6) vs. manual-install-required (none, as of this revision).
- Update `useSandboxSession.ts`'s doc-comment block (Phase 6 pattern from `sandbox_implementation.md`) to also describe `toolResults`/`toolEvents`/`toolAttach` for future contributors.

---

## PER-TOOL ROLLOUT TABLE (reference for Phases 3/5)

Full bundling specifics (package names per manager, pip modules, GitHub repos/asset patterns) are in "Extending dependency acquisition (Phase 1)" — this table is the quick-reference index into that detail, not a duplicate of it.

| Tool | Lane | Real binary / mechanism | Acquisition | Backend file |
|---|---|---|---|---|
| Semgrep | 1 | `semgrep` | pip (`semgrep`) | `sandbox_tool_semgrep.go` |
| Tree-sitter | 6 | `go-tree-sitter` cgo binding | compiled in (`go.mod` + C toolchain) | `sandbox_tool_treesitter.go` |
| Jadx | 1 | `jadx` | GitHub release (`skylot/jadx`) **+ a host JRE** (own dependency row — see Phase 1) | `sandbox_tool_jadx.go` |
| ILSpy | 1 | `ilspycmd` | **dotnet tool** (`dotnetToolStrategy`, new — see Phase 1) **+ a host .NET runtime** | `sandbox_tool_ilspy.go` |
| jwt_tool | 1 | `jwt_tool.py` | pip (`jwt_tool`), git-clone fallback if the PyPI package lags | `sandbox_tool_jwttool.go` |
| SAML Raider | 6 | native Go (`encoding/xml`) | none — reimplemented, no acquisition | `sandbox_tool_samlraider.go` |
| proxychains-ng | 4 | `proxychains4` | package manager (package name diverges from binary name on most managers — see Phase 1) | `sandbox_tool_proxychains.go` |
| bpftrace | 2 | `bpftrace` | package manager | `sandbox_tool_bpftrace.go` |
| Wireshark | 2 | `tshark` | package manager (bundled inside `wireshark-cli`/`wireshark` on RHEL/Fedora/Arch) | `sandbox_tool_tshark.go` |
| Zeek | 2 | `zeek` | package manager, **own repo required first** (not in default apt/dnf repos — extend the existing EPEL-for-x11vnc precedent) | `sandbox_tool_zeek.go` |
| AFL++ | 2 | `afl-fuzz` | package manager or source build | `sandbox_tool_afl.go` |
| Cutter | 5 | `rizin` (**not** `rz-bin` — that binary can't run the headless `aflj`/`pdgj` analysis commands Phase 7 needs) | package manager, GitHub release fallback (`rizinorg/rizin`) | `sandbox_tool_cutter.go` |
| Frida | 3 | `frida` (CLI) **and separately** `frida-server` (in-target agent) — two different binaries, two different acquisitions | pip (`frida-tools`) for the CLI; GitHub release (`frida/frida`, arch-specific asset) for `frida-server` — give each its own `DependencyStatus` row | `sandbox_tool_frida.go` |

**Cut (Consolidation Audit, not built):** ~~Ghidra~~ (duplicate of Cutter), ~~LibAFL~~ (duplicate of AFL++), ~~TruffleHog~~ (subset of Semgrep).

---

## TESTING STRATEGY

### Unit Tests
- Go: one `*_test.go` per new `api/sandbox_tool_*.go`, stubbing `commandRunner`/`spawn` the same way `sandbox_manager_test.go`/`sandbox_deps_test.go` already stub process execution — none of these tests should require the real tool binary installed in CI, matching the existing bwrap-optional pattern. Lane 6 tests (Tree-sitter, SAML Raider) call the in-process function directly with no stubbing needed at all, since there's no subprocess boundary.
- Frontend: one test per new hook (`useToolScan.test.ts` etc.) mocking `sandboxToolService`, plus a smoke test per rolled-out tool page confirming it renders real-shaped data instead of importing its old fixture array (which should be deleted, not left dead in the file).

### Integration Tests
- Extend `integration-tests/` (real services, no mocks, per root `CLAUDE.md`) with a Semgrep end-to-end case (Phase 3) as the reference example. Static analysis (Lane 1) and Lane 6 (Tree-sitter, SAML Raider) are the only tools with no privilege/namespace prerequisites, making them the cheapest to run unconditionally in CI — prefer growing this set first. Gate every Lane 2/3/5 integration test behind `t.Skip()` when the relevant binary or capability isn't available, matching how `sandbox_implementation.md` already gates `bwrap`-dependent tests.

### Edge Cases
- Tool binary missing entirely: `DependencyStatus.Error` surfaces in the existing missing-deps UI; `run()`/`start()` must fail fast with that same message, not a raw `exec: "semgrep": executable file not found in $PATH`. Not applicable to Lane 6.
- Lane 2 tool started with no session running, or session stops mid-stream: WS must close cleanly, frontend `useToolStream` must reflect `running: false`, not hang.
- Lane 3 (Frida) bridge process crashes mid-session (script caused a segfault in the target): `useToolAttach` must surface `attached: false` and the crash reason, not silently stop updating.
- Two Lane 1/6 scans requested back-to-back for the same tool: second request should either queue or reject with a clear "scan already running" error — decide per Phase 2's generic handler, applied uniformly to all Lane 1/6 tools rather than each pilot inventing its own answer.

---

## VALIDATION COMMANDS

### Level 1: Syntax & Style
```bash
cd packages/KNIRVENGINE/gui && npx tsc --noEmit
cd packages/KNIRVENGINE/gui && npm run lint
cd packages/KNIRVENGINE && go vet ./...
cd packages/KNIRVENGINE && gofmt -l . | grep -v vendor
```

### Level 2: Unit Tests
```bash
cd packages/KNIRVENGINE && go test ./api/... -run 'TestSandboxTool' -v
cd packages/KNIRVENGINE/gui && npx jest --testPathPattern="useTool|sandboxTool"
```

### Level 3: Manual Validation (per phase, see each phase's own VALIDATE line above)

### Level 4: Container Build
```bash
podman build -f packages/KNIRVENGINE/Containerfile packages/KNIRVENGINE
```

---

## ACCEPTANCE CRITERIA

- [ ] Phase 0's namespace-join question is answered with evidence (real `nsenter` output against a real running session), not assumed.
- [ ] `sandbox_deps.go` supports pip / GitHub-release acquisition without changing `Bubblewrap.tsx`'s existing missing-deps UI.
- [ ] Semgrep (Lane 1 pilot) produces real findings against a real mounted project, with the existing `Semgrep.tsx` UI unchanged apart from its data source.
- [ ] Tree-sitter and SAML Raider (Lane 6) both round-trip through the same `useToolScan` hook Semgrep uses, with zero backend dependency-check entries.
- [ ] proxychains-ng (Lane 4 pilot) demonstrably redirects a real target's outbound connections, observable in the already-real Proxy tool.
- [ ] bpftrace (Lane 2 pilot) streams real kernel events from the real sandboxed target's syscalls, not seeded log lines.
- [ ] Frida (Lane 3) attaches to the real sandboxed target's real PID and both loads a script and reports its `send()` output live.
- [ ] Cutter headless analysis returns real function names/addresses for a real binary, not the `verify_license_token`/`0x00101f10` fixture.
- [ ] `Ghidra.tsx`, `LibAFL.tsx`, and `TruffleHog.tsx` are removed from the app (route, nav card, and file) — not merely left unbuilt.
- [ ] Every one of the 13 built tools' `seedFlows`/`findings`/`functions`-style hardcoded fixture arrays is deleted from its component once that tool's real backend lands (no dead mock data left behind "just in case").
- [ ] All Level 1–2 validation commands pass with zero errors after each phase.

---

## OPEN QUESTIONS / ASSUMPTIONS

**Still open:**
- **Open, follow-up to Phase 0's spike (not blocking Phase 2, blocking before production)** — the same spike must be re-run against the actual production container (UBI9-minimal, per `Containerfile`) before shipping there; this session's findings, including the Option A confirmation, are from a Kali dev host, and container `nsenter`/capability/`sudo` behavior can differ (a container may already run as root, may need `--cap-add=SYS_ADMIN` at the container-runtime level instead of in-container `sudo`, etc.).

Confirmation:

- ~~**Open, small implementation detail (not a re-open of the Option A decision)** — whether the engine invokes `sudo nsenter ...` per tool-launch, or the engine process runs as root throughout.~~ **Resolved:** the engine remains unprivileged and launches the bundled capability-carrying `nsenter` helper directly.

Confirmation: The engine process and Electron host run as the logged-in user; only the bundled `nsenter` helper has the required file capabilities.

- **Open, blocking for Phase 5's Wireshark/Zeek — now confirmed rather than hypothetical**: Phase 0's spike directly confirmed `--share-net` (`sandbox_manager.go:412-413`) shares the host's network namespace verbatim (identical namespace inode observed on the live sandbox, twice) — there is no dedicated per-session veth today. Packet capture needs the networking redesign (a per-session veth pair) called out in Phase 5 before Wireshark/Zeek can be scoped to "this target's traffic" rather than "all host traffic." What was previously a maybe is now a confirmed requirement; the open part is scheduling/designing the veth work, not whether it's needed.

**Resolved this revision (user-confirmed):**
- ~~Open — Tree-sitter's acquisition: CLI binary vs. embedding a WASM/cgo parser directly in the Go backend.~~ **Decided: official Go bindings via cgo.** Folded into Lane 6, Phase 1 (build-time C toolchain requirement), and Phase 3 (rollout task).
- ~~Open, needs user confirmation — SAML Raider's re-scope to a native Go implementation.~~ **Confirmed: proceed with the native Go implementation.** Folded into Lane 6 and moved from the plan's original standalone "Phase 8" into Phase 3 (see Recommended Build Order and Amendments).
- ~~Open, needs user confirmation — Ghidra's headless-with-native-UI default vs. full GUI-in-noVNC.~~ **Confirmed: implement headless, in KNIRVENGINE's own UI, per other tool conventions.** Folded into Phase 7.
- ~~Open, needs one more confirmation — whether the Ghidra headless-UI answer transfers to Cutter, since Ghidra was cut and Cutter took its slot.~~ **Confirmed: "Correct!"** — Phase 7 stands as written (headless rizin, KNIRVENGINE's own UI, no GUI-in-VNC).
- ~~Open, blocking — Phase 0's namespace-join mechanism choice (A vs. B vs. C).~~ **Confirmed: Option A (root/`sudo` escalation)**, matching the precedent already set for KNIRVSERVER's cgroup control. Verified live twice — once establishing the constraints (wrong-PID issue, unprivileged `nsenter` failing on mnt/pid/ipc/uts/cgroup, confirmed host-shared `net` namespace), once with the user reproducing the exact join **as root**, which succeeded cleanly on every namespace including `cgroup`. Options B and C are dropped. See ARCHITECTURE's Phase 0 write-up and "Patterns to Follow" for the confirmed `spawnJoined()` shape and the full command transcript.

**Assumptions carried forward:**
- **Assumed** — this plan's phase order (0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8) is priority-ordered by risk-resolution-first (Phase 0) and reuse-value (Lane 1/6's generic machinery serves 6 of 13 tools), refined by the Recommended Build Order's dependency-tier sequencing. If a specific tool is more urgent than this order implies (e.g., Frida before Semgrep), phases 3-7 can be reordered without touching Phases 0-2's foundations.
- **Assumed** — single active `SandboxSession` scope carries over from `sandbox_implementation.md` unchanged; no tool in this plan needs multi-session support.
- **Assumed** — the Consolidation Audit's 3 cuts (Ghidra, LibAFL, TruffleHog) are applied to this plan's scope; if the user wants any of them reinstated, they re-enter as their own lane/phase without disturbing the 13 kept tools' design.

## NOTES (open canvas)

**Why 6 lanes and not "one handler per tool"?** 13 bespoke implementations would mean 13 places to get subprocess lifecycle, WS fan-out, and error handling right (or wrong). The existing codebase already proves the value of shared plumbing — `terminal_manager.go` → `sandbox_manager.go`'s structural mirroring, and `sandbox_manager.go`'s own `spawn()`/`handleOutput()`/`watchProcess()` already serving 3 different subprocesses (Xvfb/bwrap/x11vnc) uniformly. This plan extends that same instinct one level further: lanes are the abstraction the *tools themselves* impose (batch vs. streaming vs. RPC vs. launch-modifier vs. headless-GUI vs. in-process), not an arbitrary grouping.

**Why is Semgrep the Lane 1 pilot instead of, say, jwt_tool?** Semgrep already has a stable, well-documented `--json` output format and `Semgrep.tsx`'s mock `Finding` interface (`ruleId`/`severity`/`file`/`line`/`message`) already matches Semgrep's real JSON shape almost field-for-field — meaning the pilot proves the Lane 1 *plumbing* without simultaneously fighting an output-format design problem. jwt_tool's real output is messier (interactive-oriented text, multiple attack modes) and is better attempted once the plumbing is already proven elsewhere.

**Why is proxychains-ng its own lane instead of Lane 2?** Every other "operate on the live target" tool attaches *after* the target is already running. proxychains-ng's LD_PRELOAD mechanism only redirects sockets of processes it directly execs (and their children) — it cannot retroactively wrap an already-running process. That single mechanical fact is what forces it to be a `bwrap` launch-argv change instead of a followup process, and is worth keeping as a structurally separate lane rather than a special case bolted onto Lane 2.

**Why does Cutter win over Ghidra given Ghidra is the more widely-recognized tool?** Recognition wasn't one of the standing criteria — size, bundling ease, and architecture fit were, and on all three Cutter's rizin engine wins clearly: a small native binary vs. a multi-hundred-MB JRE application with no clean single-binary distribution. The `rz-ghidra` plugin note in Phase 7 is the hedge — if rizin's native decompiler output proves materially worse than Ghidra's, that plugin gets most of Ghidra's decompilation quality without the bundling cost that got Ghidra cut in the first place.

**Why does AFL++ win over LibAFL?** Not a quality judgment — LibAFL can outperform AFL++ in the hands of someone writing a custom harness. The disqualifier is operational: every other tool in this plan (and the whole KNIRVENGINE UX) assumes "operator points the tool at an already-running or already-mounted target, tool runs." LibAFL has no such mode — it's a framework you compile a new fuzzer with per target, which doesn't fit a generic "Fuzzing" console at all.

**Why does Semgrep absorb TruffleHog instead of the reverse?** TruffleHog is deeper on the one thing it does (secret detection, including live-credential verification), but that's the entirety of what it does. Semgrep's `p/secrets` ruleset covers the same detection surface as one ruleset among several, so keeping Semgrep loses no *category* of coverage, only the live-verification depth within that one category — named explicitly in the Consolidation Audit as the traded-off capability.

## AMENDMENTS

- **2026-08-25** — Tool Consolidation Audit added: Ghidra, LibAFL, and TruffleHog cut as duplicates of Cutter, AFL++, and Semgrep respectively (16 → 13 tools). Recommended Build Order table added under ARCHITECTURE, cutting across the numbered Phases. New Lane 6 ("Native Go, in-process") introduced, combining Tree-sitter's confirmed cgo-binding approach and SAML Raider's confirmed native-Go approach — both reuse the `useToolScan` hook and a new `sandbox_tool_native.go` generic handler rather than getting bespoke frontend/backend shapes. SAML Raider's work was moved out of the original standalone "Phase 8" and merged into Phase 3's rollout (it has no namespace-join dependency, so there's no reason to schedule it last); the old Phase 9 ("Packaging & Docs") was renumbered to Phase 8 as a result. Three previously-open design questions were resolved per user answers: Tree-sitter → Go cgo bindings (not a CLI subprocess); SAML Raider → native Go reimplementation (not a bundled Burp extension); reversing tool → headless-with-native-UI (originally answered for Ghidra, carried forward to Cutter in this revision — user separately confirmed "Correct!" on that transfer).
- **2026-08-25 (same day, follow-up)** — Phase 0 executed live: a temporary spike test (`api/zz_phase0_spike_test.go`, deleted after use) drove `SandboxManager.CreateSession`/`Start()` directly to launch two real sandbox sessions (real `bwrap`/`Xvfb`/`x11vnc`) on the dev host and ran `ps`/`nsenter` probes against them. Findings, folded into ARCHITECTURE, the Phase 0 task list, the "Namespace-joined spawn" pattern snippet, and Open Questions: (1) `SandboxSession.Pid` is bwrap's outer setup process, not the sandboxed target — the real namespaces live on its child; `spawnJoined()` must resolve that child PID, not use `session.Pid` directly. (2) `--share-net` was confirmed to literally share the host's network namespace (no per-session veth exists today), which independently confirms the Phase 5 Wireshark/Zeek capture-scope concern was real, not speculative. (3) Unprivileged `nsenter` can join the sandbox's user namespace (since KNIRVENGINE's own process owns it) but categorically fails on mnt/pid/ipc/uts/cgroup — alone, combined with `-U --preserve-credentials`, or via a two-stage nested-`nsenter` workaround — reproduced identically across both live sessions. The original `spawnJoined()` design (plain `nsenter`) is now known not to work; three replacement mechanisms are documented (sudo/root escalation, an in-namespace resident stub reached over a socket — recommended default, or a `setcap` file capability), with the final choice left open for user sign-off since it's a security-posture decision, not an implementation detail. The dev-host result (Kali, kernel 5.15, util-linux 2.37.2) still needs re-confirming against the production UBI9-minimal container before Phase 2 is built there.
- **2026-08-25 (same day, second follow-up)** — Phase 0's mechanism choice resolved: the user independently reproduced the spike as root via `sudo` (recreating `api/zz_phase0_spike_test.go` from this doc's findings, per its own instructions), and the join succeeded cleanly on every namespace — including `cgroup`, the one namespace that had failed even in every unprivileged combination attempt. `ps -ef` inside the joined namespace showed exactly the sandbox's own process tree (inner `bwrap` as pid 1, the real target as pid 2), `ls /` showed only the sandboxed rootfs, and `id` stayed `uid=0` (root's real identity, confirming no `-U`/user-namespace join is needed or wanted). This matches a decision already made for KNIRVSERVER (always run with `sudo` for cgroup control) — no new security posture is introduced, just reuse of an existing one. Options B and C are dropped from the plan. `spawnJoined()`'s confirmed shape, `SandboxSession.InnerPid`, and the full command transcript are folded into ARCHITECTURE, "Patterns to Follow," the Phase 0 task list, and Open Questions (moved from "still open" to "resolved"). One incidental cleanup during this session: 9 leaked `knirv-engine --production` processes (and their orphaned `bwrap`/`knirvclient` sandbox children on `DISPLAY :99`) from earlier, unrelated testing were found running on the dev host and stopped — unrelated to this plan's tools, noted here only because it happened mid-investigation. The spike test file was deleted again after use.
- **2026-08-26** — Bundling-methods guide completed: "Extending dependency acquisition (Phase 1)" was previously grouped only by strategy category (system package manager / pip / GitHub release / compiled-in), which wasn't a complete per-tool implementation guide — two tools (ILSpy, Jadx) didn't cleanly fit any of the three subprocess-based strategies, and two (Cutter, Frida) each actually need two separate acquisitions for two different binaries. Replaced with five per-strategy tables giving every tool's exact binary name, package name per package manager (several diverge from the binary name, e.g. `proxychains4`→`proxychains-ng` on most managers, `afl-fuzz`→`afl++`/`aflplusplus`), pip module name, or GitHub repo + asset pattern. New gaps surfaced and folded into Phase 1's task list and the Per-Tool Rollout Table: Jadx needs a host JRE (its own dependency row, recommended over bundling a JRE); ILSpy needs a **new fourth acquisition strategy** (`dotnetToolStrategy`, mirroring `pipStrategy`) that didn't exist in the plan before; Zeek needs its own package repo enabled first (extending the existing EPEL-for-x11vnc precedent), same as Jadx/x11vnc-shaped problems already solved once; Frida's CLI (`frida-tools`, pip) and `frida-server` (the in-target agent, GitHub release, arch-specific) are two different binaries via two different strategies and need separate `DependencyStatus` rows, not one combined "Frida" row. Also corrected a binary-name error carried since an earlier revision: Cutter's headless engine is the `rizin` binary itself, not `rz-bin` (a different, narrower rizin-suite utility that cannot run the `aflj`/`pdgj` analysis commands Phase 7 depends on) — fixed in the lane table, Phase 7's task description, and the Per-Tool Rollout Table. Per explicit instruction this pass was scoped to the plan document only — a separate, unrelated implementation effort is already in progress against the actual codebase and was deliberately not audited or reconciled here.
