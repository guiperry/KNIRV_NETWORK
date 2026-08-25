# Feature: Sandbox-Anchored Tool Pipeline

The following plan should be complete, but validate documentation and codebase patterns and task sanity before you start implementing.

Pay special attention to naming of existing utils/types/models. Import from the right files. This plan spans two codebases — the GUI (`packages/KNIRVENGINE/gui`, React/TS/Vite) and the Go backend (`packages/KNIRVENGINE`) — treat them as one implementation unit; they ship together.

## Feature Description

KNIRVENGINE's tool suite (Proxy, Instrumentation, Reversing, Fuzzing, Static Analysis, Packet Capture, Auth Audit) currently renders as independent, disconnected consoles with local mock state — nothing actually runs, and nothing requires a target. This feature turns the Sandbox into the mandatory front door of the pipeline: the operator picks a target project on the Dashboard, launches it inside a real Bubblewrap namespace with a real Xvfb display, watches it live via a real noVNC stream **docked to the bottom of the content area** (persistent across navigation), and every other tool is locked until that session is running. The lock isn't cosmetic — it's backed by a real Go orchestration service (mirroring the existing `TerminalManager`) that actually spawns `bwrap`, tracks the child process, and streams status over WebSocket.

## User Story

As a security operator running KNIRVENGINE
I want to load a target project, launch it inside an isolated sandbox, and keep its live display pinned at the bottom of my screen while I work
So that every tool I use afterward is provably operating against a real, isolated, observable target — not a disconnected mock

## Problem Statement

1. The 8 tool consoles have no shared session concept — each is an island of local `useState`, so "using mitmproxy on the target" and "using Frida on the target" have no actual relationship to each other or to a real running process.
2. `App.tsx` re-mounts the entire layout (`Sidebar` + wrapper `div`s) on every route (9 near-identical copies of the same JSX block) — there is no persistent layout region, so nothing can survive a route change today, which blocks the "dock stays visible while navigating" requirement outright.
3. Nothing prevents visiting `/fuzzing` or `/proxy` before any target exists.
4. The Sandbox's own `Bubblewrap.tsx` and `NoVnc.tsx` pages are themselves mock — `Launch` just prints fake log lines, `noVNC` never connects to anything.

## Solution Statement

- Consolidate `App.tsx`'s duplicated per-route layout into one `AppLayout` (Sidebar + main content + a **dock region** that outlives route changes via `<Outlet/>`).
- Introduce a `SandboxContext` (mirrors the existing `OnboardingContext` pattern) holding one `SandboxSession` at a time, backed by a real Go service.
- Build a new Go `SandboxManager` (`api/sandbox_manager.go`), structurally mirroring the existing `TerminalManager` (`api/terminal_manager.go`) almost line-for-line: session CRUD + a WebSocket per session. Its `Start()` shells out to `bwrap` wrapping `Xvfb` + the target command, and a second subprocess (`x11vnc`) exposes the framebuffer, bridged to the browser over a **Go-native WebSocket-to-TCP proxy** (see Notes — no Python/websockify dependency needed).
- Swap the GUI's decorative `NoVnc.tsx` canvas for a real RFB client (`@novnc/novnc`) rendered inside the persistent dock, connected to the live session's WebSocket.
- Gate the 7 non-sandbox tool routes behind a `RequireSandbox` wrapper that renders a locked in-place screen when no session is `running`.
- Link the Dashboard's loaded project (name/label only — see Gotcha in Task list re: File System Access API path limits) into the session as `targetLabel`, pre-filling the Sandbox launch form.
- Define and document the `useSandboxSession()` contract every future tool integration must consume (session id, status, `netnsId`, `vncWsPath`) — this is the enforced seam, not a suggestion.

## Out of Scope / Non-Goals

- **Not building real mitmproxy/Frida/Ghidra/AFL++/Wireshark/jwt_tool/SAML Raider backend integration.** Those 7 tools stay local-mock consoles this phase; they only gain (a) a route gate and (b) a `useSandboxSession()` read of the live target label/status for display. Wiring each tool to actually operate through the sandbox's netns is separate, future, per-tool work — this plan defines the contract, not each tool's implementation.
- **Not adding multi-session support.** One active `SandboxSession` app-wide. The data model isn't hostile to multiple sessions later, but nothing in this plan builds a session switcher.
- **Not solving the Electron real-filesystem-path problem.** Dashboard's `showDirectoryPicker()` (File System Access API) never exposes an absolute host path, even in Electron's renderer — see Gotcha. The Bubblewrap launch form keeps a separate, explicitly-required host command/path field.
- **Not changing `Dashboard.tsx`'s file browser/editor behavior.** Only its `targetName` value gets threaded into the new `SandboxContext`.
- **Not touching KNIRVGATEWAY, KNIRVCHAIN, or any other `packages/KNIRV*` package.** This is entirely within `packages/KNIRVENGINE`.

## Feature Metadata

**Feature Type**: New Capability (frontend architecture + new backend service)
**Estimated Complexity**: High
**Primary Systems Affected**: `packages/KNIRVENGINE/gui` (App shell, routing, Sidebar, AuthContext, Sandbox tool pages), `packages/KNIRVENGINE/api` (new Go service), `packages/KNIRVENGINE/main.go`, `packages/KNIRVENGINE/Containerfile`
**Dependencies**: `bubblewrap`, `Xvfb`, `x11vnc` (new OS packages in the runtime container); `@novnc/novnc` (new npm dep). No new Go module deps — `gorilla/mux`, `gorilla/websocket`, `google/uuid` already present (`go.mod`).

## Related Work

**Implements**: user request, this session (no tracker ticket)

**Back-references**:
- Prior session in this conversation: the KNIRVENGINE nav/tool-page hollowing-out that created `Sidebar.tsx`'s current 8-category nav and the `components/tools/**` tree this plan modifies.

**Forward-references**: (none yet — append future per-tool sandbox-integration plans here as they're created)

---

## CONTEXT REFERENCES

### Relevant Codebase Files — READ THESE BEFORE IMPLEMENTING

**Frontend:**
- `gui/src/App.tsx` (full file, esp. lines 246–527) — Why: the 9x-duplicated per-route layout block that must collapse into one `AppLayout`. Every `<Route path="/x" element={<ProtectedRoute><div className="min-h-screen...">...<Sidebar/>...<main>...</main></div></ProtectedRoute>}/>` is the pattern being replaced.
- `gui/src/components/Sidebar.tsx` (lines 1–41 for `ActiveView` type, 56–134 for `navigation` array, 226–274 for render/gating loop) — Why: nav reorder target; `canAccessPage`/`canAccessSubPage` gating call sites to extend.
- `gui/src/components/AuthContext.tsx` (lines 195–221 for `pageAccess`/`toolPages`, 217–221 for `subPageAccess`) — Why: `sandbox` must move earlier in `toolPages` array (order is cosmetic there, but add it explicitly); no structural change needed to the access-control shape itself, just data.
- `gui/src/components/Dashboard.tsx` (lines 80–97 `loadTarget`, 98–111 `chooseTarget`, 156 `clearTarget`) — Why: exact points to call `SandboxContext`'s `setTargetLabel`/session-label sync when a project is loaded/cleared.
- `gui/src/components/tools/Sandbox.tsx` (full file) — Why: current landing page pattern (`isSubRoute` + nested `<Routes>`) — same pattern used by all 7 other tool-category pages (`Instrumentation.tsx`, `Reversing.tsx`, etc.) that `RequireSandbox` must wrap without breaking.
- `gui/src/components/tools/sandbox/Bubblewrap.tsx` (full file) — Why: this becomes the real launch form. Its existing `binds`/`unshareAll`/`shareNet`/`dieWithParent`/`display`/`target` state and generated-`command` preview are good UI to keep; only the `launch()` function's body changes from local mock to a real `SandboxClient.launch()` call.
- `gui/src/components/tools/sandbox/NoVnc.tsx` (full file) — Why: this becomes the real RFB client. Its connection-panel UI (host/port/path/quality/view-only) stays, but the black canvas placeholder (lines 84–106) is replaced with an `@novnc/novnc` `RFB` instance.
- `gui/src/components/onboarding/OnboardingProvider.jsx` (full file, 43 lines) — Why: **the** context-provider convention to mirror exactly for `SandboxProvider` — `createContext()` + `useX()` hook that throws if used outside provider + plain `useState` internals + a `value` object passed to `.Provider`.
- `gui/src/services/terminalService.js` (full file) — Why: the frontend REST+WS client shape to mirror for a new `sandboxService.js` (`create`/`get`/`list`/`close` + `getXWebSocketUrl`/`createXWebSocket`). Nearly copy-paste-able.
- `gui/src/utils/websocket.ts` (lines 1–40 for message-type interface conventions: `WebSocketMessage<T>`, `AgentStatusUpdate`, etc.) — Why: naming convention for new `SandboxStatusUpdate`/`SandboxLogLine` message types.
- `gui/vite.config.ts` (lines 40–49) — Why: confirms `/api` proxies to `http://localhost:8081` in dev — no new proxy config needed, new `/api/v1/sandboxes*` routes ride the same proxy.
- `gui/package.json` — Why: add `@novnc/novnc` here.

**Backend (Go, module `KNIRVENGINE/desktop-client`):**
- `api/terminal_manager.go` (full file, 644 lines) — Why: **the** structural template for the new `SandboxManager`. `TerminalManager`/`TerminalSession` → `SandboxManager`/`SandboxSession`. Mirror: the `sync.RWMutex`-guarded `map[string]*Session`, `context.WithCancel` for cancelable lifetime, `exec.CommandContext`, stdout/stderr pipe→goroutine→`broadcastToClients` fan-out, `gorilla/websocket` upgrade in `HandleXWebSocket`, and the exact `RegisterHandlers(router *mux.Router)` REST shape (lines 532–539: list/create/get/delete/history/ws).
- `api/target_system_service.go` (lines 1–150, esp. 16–27 `TargetSystemType` consts and 62–92 `TargetSystemConnection` interface + `TargetSystemService`) — Why: `TargetTypeApplication` already exists as a target type. The plan reuses `TargetSystemService` for target **identity** (register the sandboxed target as a `TargetSystem` record) while `SandboxManager` owns the **process/session** — don't duplicate the target model, extend it.
- `api/application_connection.go` (full file, 168 lines) — Why: the existing (unsandboxed) `TargetSystemConnection` implementation for `TargetTypeApplication`. `launchApp` (lines 113–129) is the precedent for "shell out to launch a target" — the new sandboxed path wraps this exact idea in `bwrap`, it doesn't replace the interface.
- `api/process_utils.go` + `api/process_utils_unix.go` (both full files, ~35 lines total) — Why: `terminateProcess(pid, force)` / `isProcessAlive(pid)` — reuse these verbatim for killing the `bwrap` process tree; don't write new SIGTERM/SIGKILL logic.
- `api/standard_response.go` (lines 1–60) — Why: `RespondWithSuccess`/`RespondWithSuccessAndStatus`/`RespondWithList` — use these for every new handler's response, not raw `json.NewEncoder`.
- `api/auth_service.go` (lines 543–585 `AuthMiddleware`, 587–599+ `RequirePermission`) — Why: pattern for gating new sandbox routes if per-route auth is added (note: `terminal_manager.go`'s own handlers rely on the router-global `securityMiddleware` from `simple_server.go:458` rather than per-route middleware — match that same reliance for consistency, don't add redundant per-route auth unless a specific sandbox permission is introduced).
- `simple_server.go` (lines 420–465, esp. 455 `api := s.router.PathPrefix("/api/v1").Subrouter()` and 458–465 global `Use()` middleware chain) — Why: confirms all `/api/v1/*` routes already get security+monitoring+CORS middleware for free; `SandboxManager.RegisterHandlers` needs no extra wiring for that.
- `main.go` (lines 466–469 service construction, 519–528 `RegisterHandlers` wiring) — Why: exact insertion point. `terminalManager := api.NewTerminalManager()` / `targetSystemService := api.NewTargetSystemService()` then later `terminalManager.RegisterHandlers(router)` / `targetSystemService.RegisterHandlers(router)` — add `sandboxManager := api.NewSandboxManager()` beside the others and `sandboxManager.RegisterHandlers(router)` in the same block.
- `Containerfile` (lines 1–40ish, the `microdnf install` list in the production stage) — Why: `bubblewrap`, `Xvfb` (package `xorg-x11-server-Xvfb`), and `x11vnc` are **not currently installed** in the UBI9-minimal runtime image — must be added or the whole feature no-ops in the shipped container even though it may work on a dev machine that happens to have them installed.
- `go.mod` (module line + `github.com/gorilla/mux v1.8.1`, `github.com/gorilla/websocket v1.5.3`, `github.com/google/uuid v1.6.0`, `github.com/golang-jwt/jwt/v5 v5.2.2`) — Why: confirms zero new Go dependencies are required.

### New Files to Create

**Backend:**
- `api/sandbox_manager.go` — `SandboxManager`/`SandboxSession`, REST handlers, WebSocket status/log stream, VNC WS↔TCP bridge handler.
- `api/sandbox_manager_test.go` — unit tests mirroring `target_system_test.go`'s style.

**Frontend:**
- `gui/src/components/SandboxContext.tsx` — `SandboxProvider` + `useSandbox()` hook (mirrors `OnboardingProvider.jsx`).
- `gui/src/services/sandboxService.ts` — REST+WS client (mirrors `terminalService.js`, but TypeScript to match `AuthContext.tsx`/`walletService.ts` convention for newer files).
- `gui/src/components/layout/AppLayout.tsx` — the consolidated Sidebar+content+dock shell that replaces the 9 duplicated route wrappers.
- `gui/src/components/layout/SandboxDock.tsx` — the persistent bottom-docked panel hosting the real noVNC `RFB` client.
- `gui/src/components/RequireSandbox.tsx` — the route-gate wrapper (locked in-place screen).
- `gui/src/hooks/useSandboxSession.ts` — thin re-export/selector hook other tool pages import (the documented downstream-tool contract point).

### Patterns to Follow

**Context provider shape** (mirror `gui/src/components/onboarding/OnboardingProvider.jsx:1-13` exactly):
```jsx
const SandboxContext = createContext();
export const useSandbox = () => {
  const context = useContext(SandboxContext);
  if (!context) throw new Error('useSandbox must be used within a SandboxProvider');
  return context;
};
export const SandboxProvider = ({ children }) => { /* useState + effects + value object */ };
```

**Go session-manager shape** (mirror `api/terminal_manager.go:52-120` exactly, renamed):
```go
type SandboxManager struct {
    sessions map[string]*SandboxSession
    mutex    sync.RWMutex
}
func NewSandboxManager() *SandboxManager { /* ... */ }
func (m *SandboxManager) CreateSession(userID int64, targetLabel, targetCommand string) (*SandboxSession, error) { /* ... */ }
```

**Go REST registration shape** (mirror `api/terminal_manager.go:532-539`):
```go
func (m *SandboxManager) RegisterHandlers(router *mux.Router) {
    router.HandleFunc("/api/v1/sandboxes", m.handleList).Methods("GET")
    router.HandleFunc("/api/v1/sandboxes", m.handleCreate).Methods("POST")
    router.HandleFunc("/api/v1/sandboxes/{id}", m.handleGet).Methods("GET")
    router.HandleFunc("/api/v1/sandboxes/{id}", m.handleStop).Methods("DELETE")
    router.HandleFunc("/api/v1/sandboxes/{id}/ws", m.handleStatusWebSocket)
    router.HandleFunc("/api/v1/sandboxes/{id}/vnc", m.handleVNCWebSocket)
}
```

**Response envelope** (mirror `api/standard_response.go:27-38`): every new handler calls `api.RespondWithSuccess(w, data, "message")` or `RespondWithSuccessAndStatus`, never raw `json.NewEncoder(w).Encode(...)`.

**Naming conventions**: Go — PascalCase exported types/methods, camelCase unexported (matches `terminal_manager.go` throughout). TS — `PascalCase` components/types, `camelCase` functions/hooks, tool-category files exactly match the existing `tools/<Category>.tsx` + `tools/<category>/<Tool>.tsx` two-level pattern already established by all 8 categories.

**Error handling**: Go handlers return `http.Error(w, fmt.Sprintf("...: %v", err), http.StatusX)` for the simple services (`terminal_manager.go:569,579`) — match this rather than the fancier `respondWithError` used in `auth_service.go`, since `SandboxManager` sits structurally next to `TerminalManager`, not next to `AuthService`.

---

## ARCHITECTURE

### The `SandboxSession` data model (the enforced contract)

This is the shape every future tool integration reads. Frontend (`gui/src/components/SandboxContext.tsx`) and backend (`api/sandbox_manager.go`) both model it; the frontend type is the wire contract.

```ts
type SandboxSessionStatus = 'idle' | 'provisioning' | 'running' | 'stopping' | 'stopped' | 'error';

interface SandboxSession {
  id: string;              // uuid, backend-assigned
  targetLabel: string;     // human name — from Dashboard's loaded project, or typed manually
  targetCommand: string;   // host path/command bwrap execs — always explicit, see Gotcha
  status: SandboxSessionStatus;
  pid?: number;             // bwrap's own pid, once running
  display: string;          // e.g. ":99"
  netnsId: string;          // opaque handle other tools reference to say "operate on THIS sandbox's network"
  vncWsPath?: string;       // e.g. "/api/v1/sandboxes/{id}/vnc" — set once status is 'running'
  statusWsPath: string;     // "/api/v1/sandboxes/{id}/ws" — available once status is 'provisioning'
  createdAt: string;
  error?: string;
}
```

`netnsId` is the seam: it's an opaque string today (the session `id` itself is sufficient since there's one session), but it's the field a future Proxy implementation reads to know "redirect traffic from netns X," a future Frida implementation reads to know "attach inside netns X's PID namespace," and a future Packet Capture implementation reads to know "capture on netns X's veth." Naming it distinctly from `id` now (even though it equals `id` in this single-session phase) avoids a rename when multi-session support eventually decouples them.

### Frontend layout consolidation (prerequisite for docking)

`App.tsx` today repeats this block 9 times (one per `<Route>`):
```tsx
<ProtectedRoute><div className="min-h-screen ..."><div className="flex">
  <Sidebar .../>
  <main className="flex-1 lg:ml-64">...<ThePage/></main>
</div></div></ProtectedRoute>
```
Collapse to one layout route wrapping an `<Outlet/>`:
```tsx
<Route element={<ProtectedRoute><AppLayout/></ProtectedRoute>}>
  <Route path="/dashboard" element={<Dashboard/>} />
  <Route path="/proxy" element={<RequireSandbox><Proxy/></RequireSandbox>} />
  {/* ...6 more gated tool routes... */}
  <Route path="/sandbox/*" element={<Sandbox/>} />
  <Route path="/settings" element={<Settings/>} />
</Route>
```
`AppLayout` renders `Sidebar` once, a `<main>` region for `<Outlet/>`, and `SandboxDock` **below** `<main>` (not inside it) so it survives every child-route swap — this is the only way "dock persists across tool navigation" is achievable; bolting a dock onto each individual tool page cannot satisfy that requirement.

### Enforcement UX

`RequireSandbox` reads `useSandbox().session?.status`. If not `'running'`, it renders the wrapped tool's own header chrome (so the page still visually identifies itself) plus a centered lock card: icon, "Sandbox required," one line naming the current status (`idle` → "No sandbox running", `provisioning` → "Sandbox starting…" with a spinner, `error` → the session's `error` string), and a button to `navigate('/sandbox/bubblewrap')`. Sidebar and dock remain fully visible/interactive underneath — this is a content-area lock, not a page redirect, per the confirmed decision.

### VNC bridge: Go-native, not `websockify`

The Containerfile's runtime stage is UBI9-minimal with **no Python**. Rather than adding a Python runtime just to run `websockify`, `SandboxManager.handleVNCWebSocket` implements the WS↔TCP bridge directly in Go using `gorilla/websocket` (already a dependency): upgrade the incoming connection, `net.Dial("tcp", "127.0.0.1:<vnc-port>")` to the session's `x11vnc` instance, then two goroutines pumping bytes each direction (`io.Copy`-style, framed as binary WS messages). This is ~40 lines, no new dependency, and matches the codebase's existing "everything in Go" shape better than introducing Python. The GUI's `@novnc/novnc` `RFB` client connects directly to this Go-served WebSocket URL — no separate noVNC HTML app needed, since noVNC ships as an importable JS module, not just a static webapp.

---

## IMPLEMENTATION PLAN

### Phase 1: Frontend Foundation (layout + context scaffold, no backend dependency)

**Independent of:** Phase 2 (Go work can start in parallel — different repos of code, only the wire contract in ARCHITECTURE above needs agreement).

**Tasks:**
- Consolidate `App.tsx` into `AppLayout` + `<Outlet/>`.
- Scaffold `SandboxContext`/`SandboxProvider` with **client-only mock state** initially (status toggles via local `setState`, same shape `Bubblewrap.tsx` already fakes) so Phase 1 is independently testable before Phase 2 lands.
- Reorder `Sidebar.tsx`'s `navigation` array: `dashboard`, `sandbox`, then the 7 tools, then `settings`.
- Build `RequireSandbox` and wire it around the 7 non-sandbox tool routes.
- Build `SandboxDock` shell (collapsed/expanded, resizable height, mounted in `AppLayout`) with placeholder content — real RFB wiring comes in Phase 4.

### Phase 2: Backend Foundation (Go service, structurally independent of Phase 1)

**Independent of:** Phase 1.

**Tasks:**
- `api/sandbox_manager.go`: `SandboxManager`/`SandboxSession` structs, `CreateSession`/`GetSession`/`ListSessions`/`CloseSession`, `RegisterHandlers`, REST handlers — mirroring `terminal_manager.go` 1:1 in shape. `Start()` at this stage can launch a **placeholder** command (e.g. `sleep` or an echo loop) to prove the session lifecycle end-to-end before real `bwrap`/`Xvfb` sequencing is added in Phase 3.
- Wire into `main.go` beside `terminalManager`/`targetSystemService`.
- `gui/src/services/sandboxService.ts` real REST+WS client (mirrors `terminalService.js`).

### Phase 3: Real Process Orchestration

**Depends on:** Phase 2 (needs the session skeleton to attach real subprocess logic to).

**Tasks:**
- Replace the Phase 2 placeholder `Start()` body with the real sequence: launch `Xvfb :99` → wait for socket readiness → launch `bwrap [binds...] --unshare-all --share-net --setenv DISPLAY :99 -- <targetCommand>` as a child of the session context → launch `x11vnc -display :99 -rfbport <port> -forever` as a sibling.
- Track all three PIDs on `SandboxSession`; `Close()`/`CloseSession` must tear down in reverse order (x11vnc → bwrap → Xvfb) using `terminateProcess` from `process_utils.go`.
- Stream each subprocess's stderr into the session's status WebSocket as log lines (mirror `terminal_manager.go`'s `handleOutput`/`broadcastToClients`).
- `Containerfile`: add `bubblewrap`, `xorg-x11-server-Xvfb`, `x11vnc` to the `microdnf install` list in the production stage.

### Phase 4: Real Dock Rendering

**Depends on:** Phase 3 (needs a real VNC port to connect to) and Phase 1 (needs `SandboxDock` to exist).

**Tasks:**
- `npm install @novnc/novnc` in `gui/`.
- `api/sandbox_manager.go`: add `handleVNCWebSocket` (the Go-native bridge from ARCHITECTURE).
- Rewrite `gui/src/components/tools/sandbox/NoVnc.tsx`'s canvas region (and `SandboxDock.tsx`, which renders the same client) to instantiate `new RFB(canvasEl, session.vncWsPath, {...})` instead of the decorative placeholder.
- Swap `SandboxContext` from Phase-1 mock state to real `sandboxService.ts` calls.

### Phase 5: Target Bridging + End-to-End Enforcement

**Depends on:** Phases 1–4 all landed.

**Tasks:**
- `Dashboard.tsx`: on `loadTarget`/`chooseTarget` success, call `useSandbox().setTargetLabel(name)`; on `clearTarget`, clear it.
- `Bubblewrap.tsx`: replace local mock `launch()` with `useSandbox().launch({ targetLabel, targetCommand, binds, unshareAll, shareNet, dieWithParent, display })`; its existing generated-command preview stays as a client-side echo of what the backend will run.
- `RequireSandbox` switches from Phase-1 mock status to the real `session.status` from `SandboxContext`.

### Phase 6: Downstream Tool Contract

**Depends on:** Phase 5.

**Tasks:**
- `gui/src/hooks/useSandboxSession.ts`: thin hook re-exporting the relevant slice (`{ session, isReady }`) — this is the one import every future tool-integration PR should use rather than reaching into `SandboxContext` directly, keeping the seam narrow and documented.
- Touch each of the 7 tool pages' existing "status chip" (e.g. `Proxy.tsx`'s flow-count badge, `Wireshark.tsx`'s "capturing on eth0" badge) to read `session.targetLabel`/`session.netnsId` for display instead of hardcoded text — proves the contract is consumable without doing full per-tool backend work.
- Add a short `## Sandbox Integration Contract` doc comment block atop `useSandboxSession.ts` itself (not a separate doc file) describing the fields future tool PRs may rely on, so the seam is discoverable in-editor.

---

## STEP-BY-STEP TASKS

### CREATE gui/src/components/layout/AppLayout.tsx
- **IMPLEMENT**: Renders `Sidebar` (lifted `activeView`/`sidebarOpen` state from `App.tsx`), an `<Outlet/>` inside the existing `<main className="flex-1 lg:ml-64">` region, and `<SandboxDock/>` as a sibling below `<main>`, all inside the existing gradient wrapper divs currently duplicated per-route.
- **PATTERN**: `App.tsx:265-296` (the `/` route's block) — use this exact div structure once instead of 9 times.
- **IMPORTS**: `Outlet` from `react-router-dom`; `Sidebar`; `SandboxDock`.
- **GOTCHA**: The mobile-menu-toggle button (`App.tsx:276-286`) and `RealTimeNotifications` (`App.tsx:288-291`) currently only appear on the `/` and `/dashboard` routes, not the others — decide whether they become universal (recommended, since they're layout chrome) or stay Dashboard-only; recommend universal for consistency now that there's one layout.
- **VALIDATE**: `cd gui && npx tsc --noEmit`

### UPDATE gui/src/App.tsx
- **IMPLEMENT**: Replace all 10 `<Route>` blocks with the nested-route form from ARCHITECTURE's "Frontend layout consolidation" snippet. Wrap the 7 tool routes' elements in `<RequireSandbox>`.
- **PATTERN**: React Router v6 nested routes with a layout `element` and child `<Route>`s rendering into its `<Outlet/>` (standard v6 idiom; this repo's `react-router-dom` version — check `package.json` — already supports it since `Routes`/`Route`/`Navigate` are in use).
- **IMPORTS**: Remove per-page `Sidebar` prop drilling boilerplate; add `AppLayout`, `RequireSandbox`.
- **GOTCHA**: Keep the `*` catch-all `<Navigate to="/dashboard" replace/>` (`App.tsx:520`) as a child of the same layout route, not a sibling, or unauthenticated users hitting a bad path skip `ProtectedRoute`.
- **SATISFIES**: prerequisite for dock persistence (item 3 of the original request)
- **VALIDATE**: `cd gui && npm run build`

### CREATE gui/src/components/SandboxContext.tsx
- **IMPLEMENT**: `SandboxProvider`/`useSandbox()` holding `session: SandboxSession | null`, `launch(config)`, `stop()`, `setTargetLabel(label)`, derived `isReady = session?.status === 'running'`.
- **PATTERN**: `gui/src/components/onboarding/OnboardingProvider.jsx:1-13` for the context/hook shape exactly.
- **IMPORTS**: `sandboxService` (Phase 2+) or local mock (Phase 1 only, temporary).
- **GOTCHA**: Don't persist `session` to `localStorage` the way `AuthContext` persists `user`/`token` — a sandbox session is tied to a live backend process; surviving a page reload with stale session state would show "running" for a session that's actually dead. On mount, if a `sessionId` is remembered, re-fetch it via `sandboxService.get(id)` and trust the server's status, not a cached value.
- **VALIDATE**: `cd gui && npx tsc --noEmit`

### CREATE gui/src/components/RequireSandbox.tsx
- **IMPLEMENT**: `{ children }` wrapper; if `!isReady`, render a lock screen (icon, status-aware message, CTA button `navigate('/sandbox/bubblewrap')`); else render `children`.
- **PATTERN**: Visual style should match the existing tool-page header pattern (e.g. `gui/src/components/tools/Reversing.tsx`'s header block) so the lock screen doesn't look like a different app.
- **IMPORTS**: `useSandbox` from `SandboxContext`; `useNavigate` from `react-router-dom`.
- **VALIDATE**: manual — visit `/proxy` with no session, confirm lock screen; launch sandbox, confirm `/proxy` becomes usable without a page reload (context update alone should flip it).

### UPDATE gui/src/components/Sidebar.tsx
- **IMPLEMENT**: Move the `sandbox` entry (currently `Sidebar.tsx:123-132`) to immediately after the `dashboard` entry (currently `Sidebar.tsx:57`).
- **PATTERN**: N/A — pure array reorder, no structural change.
- **VALIDATE**: visual check — Sandbox is the second sidebar item.

### UPDATE gui/src/components/AuthContext.tsx
- **IMPLEMENT**: No structural change needed — `toolPages` (`AuthContext.tsx:196-199`) is an unordered access list, not display order. Confirm `'sandbox'` is present (it already is). No task here beyond verification.
- **VALIDATE**: `grep sandbox gui/src/components/AuthContext.tsx` shows it in `toolPages`.

### CREATE gui/src/components/layout/SandboxDock.tsx
- **IMPLEMENT**: Collapsible panel fixed to the bottom of the content area (not full app width — stays right of the sidebar, matching the user's explicit "bottom of the content area" wording). Header bar shows target label + status chip + collapse/expand toggle + resize handle (simple drag-to-resize on the top edge is sufficient, no need for a full resize library). Body hosts the `RFB` canvas once Phase 4 lands; Phase 1 body is a placeholder matching `NoVnc.tsx`'s existing "not connected" state.
- **PATTERN**: Visual language from `gui/src/components/tools/sandbox/NoVnc.tsx:84-106` (the canvas region) — reuse this exact card styling so the dock and the standalone noVNC tool page look like the same feature, not two.
- **GOTCHA**: Must render unconditionally in `AppLayout` (always mounted) and internally decide collapsed-vs-expanded — do NOT conditionally mount/unmount based on `session` presence, or the `RFB` connection tears down and reconnects on every session status flicker.
- **VALIDATE**: navigate between two tool pages with dock expanded, confirm no remount (check RFB connection doesn't drop — visible as a reconnect flash).

### CREATE api/sandbox_manager.go
- **IMPLEMENT**: `SandboxSession` struct (id, targetLabel, targetCommand, status, pid, display, netnsId, timestamps, private `cmd *exec.Cmd`/`ctx`/`cancelFunc`/`mutex`/`clients map[*websocket.Conn]bool`), `SandboxManager` struct (sessions map + mutex), `NewSandboxManager`, `CreateSession`, `GetSession`, `ListSessions`, `CloseSession`, `Start()` (Phase 2: placeholder subprocess; Phase 3: real bwrap/Xvfb/x11vnc sequence), `RegisterHandlers`.
- **PATTERN**: `api/terminal_manager.go` in its entirety — same file shape, same method signatures adapted to the new struct names. Copy the `handleOutput`/`broadcastToClients` goroutine pattern (`terminal_manager.go:369-416`) verbatim for status/log streaming.
- **IMPORTS**: `context`, `encoding/json`, `fmt`, `log`, `net/http`, `os/exec`, `sync`, `time`, `github.com/google/uuid`, `github.com/gorilla/mux`, `github.com/gorilla/websocket`.
- **GOTCHA**: Unlike a terminal session (one process), a sandbox session is 2-3 processes (Xvfb, bwrap, x11vnc). `Close()` must terminate them in reverse dependency order and tolerate any of them already being dead (wrap each `terminateProcess` call, log-and-continue rather than early-return on error).
- **SATISFIES**: item 2 of the original request ("must be launched inside the sandbox... before any other tool is usable" — this is the backing service that makes the launch real)
- **VALIDATE**: `cd .. && go build ./... && go test ./api/... -run TestSandbox -v` (write `sandbox_manager_test.go` mirroring `api/target_system_test.go`'s structure first)

### UPDATE main.go
- **IMPLEMENT**: Add `sandboxManager := api.NewSandboxManager()` beside `main.go:466-469`'s other service construction; add `sandboxManager.RegisterHandlers(router)` beside `main.go:523-526`.
- **PATTERN**: Exact copy of how `terminalManager`/`targetSystemService` are constructed and registered.
- **VALIDATE**: `go build ./... && ./knirv-engine -h` (confirm it still starts)

### UPDATE Containerfile
- **IMPLEMENT**: Add `bubblewrap xorg-x11-server-Xvfb x11vnc` to the `microdnf install -y` list in the production stage (alongside the existing `webkit2gtk4.0 gtk3 curl ca-certificates`).
- **GOTCHA**: Confirm these package names resolve on UBI9-minimal (some may require enabling the CodeReady Builder / EPEL repo first — if `microdnf install` fails to resolve `x11vnc` specifically, it's commonly only in EPEL, not the base UBI repos; add the EPEL `microdnf install epel-release` step ahead of it if needed).
- **VALIDATE**: `podman build -f Containerfile . ` (or `docker build`) completes without unresolved-package errors.

### CREATE gui/src/services/sandboxService.ts
- **IMPLEMENT**: `createSandboxSession`, `getSandboxSession`, `listSandboxSessions`, `closeSandboxSession`, `getSandboxStatusWebSocketUrl`, `getSandboxVncWebSocketUrl`, `createSandboxStatusWebSocket`.
- **PATTERN**: `gui/src/services/terminalService.js` — same function shapes, same `fetch`+error-handling convention, ported to `.ts` with explicit types matching the `SandboxSession` interface from ARCHITECTURE.
- **IMPORTS**: none beyond global `fetch`/`WebSocket`.
- **VALIDATE**: `cd gui && npx tsc --noEmit`

### UPDATE gui/src/components/Dashboard.tsx
- **IMPLEMENT**: In `loadTarget` (`Dashboard.tsx:94-97`), call `sandbox.setTargetLabel(name)` (via a `useSandbox()` hook call at the top of the component). In `clearTarget` (`Dashboard.tsx:156`), call `sandbox.setTargetLabel('')`.
- **GOTCHA**: Dashboard's `name` here is a directory name from either `showDirectoryPicker()` or the `webkitdirectory` file-input fallback (`Dashboard.tsx:112-123`) — it is NOT a filesystem path the Go backend can exec. Do not attempt to derive `targetCommand` from it. The Bubblewrap launch form's `target` field (`Bubblewrap.tsx:22`) remains a required, separately-typed host command/path. Document this limitation inline as a comment where `setTargetLabel` is called, since it's the kind of thing a future contributor will otherwise "fix" incorrectly by trying to wire the path through.
- **VALIDATE**: manual — load a project on Dashboard, navigate to Sandbox → Bubblewrap, confirm the launch form shows the project name as a label/pre-filled hint even though the command field stays empty/manual.

### UPDATE gui/src/components/tools/sandbox/Bubblewrap.tsx
- **IMPLEMENT**: Replace the local `launch()` body (`Bubblewrap.tsx:39-48`) with `await sandbox.launch({ targetLabel: sandbox.targetLabel, targetCommand: target, binds, unshareAll, shareNet, dieWithParent, display })`; replace the local `log` state's mock lines with the real stream from `SandboxContext` (subscribe to the status WebSocket via a `useEffect`, append incoming log lines).
- **PATTERN**: Existing component structure stays — only the data source for `log` and the body of `launch` change from "fabricate strings" to "read real state."
- **VALIDATE**: manual — launch, confirm log lines come from the real backend (e.g. include the actual generated `pid` returned by the API, not a hardcoded `84213`).

### UPDATE gui/src/components/tools/sandbox/NoVnc.tsx and gui/src/components/layout/SandboxDock.tsx
- **IMPLEMENT**: Both host the same real `RFB` client, parameterized by `session.vncWsPath`. Extract a shared `SandboxVncCanvas` component (new file, `gui/src/components/tools/sandbox/SandboxVncCanvas.tsx`) so the standalone tool page and the dock don't duplicate `RFB` setup/teardown logic.
- **PATTERN**: `@novnc/novnc`'s documented usage: `import RFB from '@novnc/novnc/lib/rfb'; const rfb = new RFB(containerEl, wsUrl, { credentials: {} }); rfb.addEventListener('connect', ...); return () => rfb.disconnect();` inside a `useEffect` keyed on `wsUrl`.
- **GOTCHA**: Two `RFB` instances (dock + standalone page, if both mounted) would double-connect to the same VNC session — `x11vnc` by default only accepts one viewer unless `-shared` is passed. Add `-shared` to the `x11vnc` launch args in `sandbox_manager.go` (Phase 3 task) so the standalone page and the dock can both be open without kicking each other.
- **VALIDATE**: manual — open the dock AND navigate to `/sandbox/novnc` simultaneously, confirm both show the live stream without disconnecting each other.

### CREATE gui/src/hooks/useSandboxSession.ts
- **IMPLEMENT**: `export const useSandboxSession = () => { const { session, isReady } = useSandbox(); return { session, isReady }; }` plus the doc-comment block described in Phase 6.
- **VALIDATE**: `cd gui && npx tsc --noEmit`

### UPDATE gui/src/components/tools/Proxy.tsx, PacketCapture.tsx, and the other 5 tool-category pages
- **IMPLEMENT**: Each page's existing hardcoded status badge (e.g. `Proxy.tsx`'s "7 flows" / listening-address text) reads `session?.targetLabel` where it currently hardcodes a target hostname/binary name, via `useSandboxSession()`.
- **PATTERN**: Minimal, cosmetic — do not add real backend calls to these 7 files in this plan; only swap a literal string for a context read.
- **VALIDATE**: manual — launch a sandbox with target label "my-app", confirm each tool page's header reflects "my-app" instead of the old hardcoded mock name.

---

## TESTING STRATEGY

### Unit Tests
- Go: `api/sandbox_manager_test.go` mirroring `api/target_system_test.go`'s style — test `CreateSession`/`GetSession`/`ListSessions`/`CloseSession` state transitions and the max-sessions-ish guard (mirror `terminal_manager.go:90-92`'s pattern, even if the sandbox limit is 1 for this phase). Mock/stub the actual `bwrap`/`Xvfb`/`x11vnc` exec calls behind an interface so tests don't require those binaries installed in CI (see Gotcha below).
- Frontend: `gui/src/tests/SandboxContext.test.tsx` (new) verifying `launch`/`stop`/`setTargetLabel` state transitions with `sandboxService` mocked (mirror how `AuthContext` is mocked in `gui/src/tests/App.test.jsx`).

### Integration Tests
- `integration-tests/` (repo root, per `CLAUDE.md`) is Go, real services, no mocks — a sandbox integration test there would actually invoke `bwrap`, which requires the test runner to have user-namespace permissions. Flag this as environment-dependent; gate it behind a build tag or `t.Skip()` if `bwrap` isn't on `$PATH`, matching how the repo already treats root-key-gated oracle tests as conditionally skipped per `CLAUDE.md`'s Oracle section.

### Edge Cases
- Launching a second session while one is already `running` (single-session scope: return a clear 409-equivalent error, don't silently kill the first).
- `bwrap` binary missing on `$PATH` entirely (dev machine without it installed) — `Start()` must fail fast with a specific `error` on the session (`"bwrap not found: install bubblewrap"`), not a generic exec error, so `RequireSandbox`'s lock screen can show something actionable.
- Target command that exits immediately (bad path/typo) — session should transition to `'error'`, not hang in `'provisioning'` forever; reuse the `terminal_manager.go:240-263` "goroutine waits on `cmd.Wait()`, updates status" pattern for each of the three subprocesses.
- Browser navigates away from `/sandbox/novnc` while dock is also open — confirm the standalone page's `RFB` cleanly disconnects (via the `useEffect` cleanup) without tearing down the dock's separate connection.
- User closes the sandbox session while a locked tool page is currently showing the "unlocked" state — `RequireSandbox` must re-lock live (context-driven, not a one-time check on mount).

---

## VALIDATION COMMANDS

### Level 1: Syntax & Style
```bash
cd packages/KNIRVENGINE/gui && npx tsc --noEmit
cd packages/KNIRVENGINE/gui && npm run lint
cd packages/KNIRVENGINE && go vet ./...
cd packages/KNIRVENGINE && gofmt -l . | grep -v vendor  # should be empty
```

### Level 2: Unit Tests
```bash
cd packages/KNIRVENGINE && go test ./api/... -run TestSandbox -v
cd packages/KNIRVENGINE/gui && npx jest --testPathPattern="SandboxContext|sandbox"
```

### Level 3: Integration Tests
```bash
cd packages/KNIRVENGINE && go build -o knirv-engine ./main.go
which bwrap Xvfb x11vnc || echo "install bubblewrap Xvfb x11vnc locally to exercise Phase 3 end-to-end"
```

### Level 4: Manual Validation
1. `go build -o knirv-engine . && ./knirv-engine` (or existing dev flow) + `cd gui && npm run dev`
2. Load a project on Dashboard → confirm Sidebar shows Sandbox as the 2nd item.
3. Visit `/proxy` directly → confirm the locked in-place screen, not a redirect.
4. Go to Sandbox → Bubblewrap, launch with a real local binary path → confirm real PID appears in the log, not a hardcoded one.
5. Confirm the dock at the bottom of the content area shows the live target display once `x11vnc` is up.
6. Navigate to Reversing → Ghidra while the dock is open → confirm the dock stays mounted and the VNC stream doesn't reconnect.
7. Visit `/proxy` again → confirm it's now unlocked and its header shows the real target label.
8. Stop the sandbox from the dock or Bubblewrap page → confirm `/proxy` re-locks without a page reload.

### Level 5: Additional Validation
- `podman build -f packages/KNIRVENGINE/Containerfile packages/KNIRVENGINE` — confirms the new OS packages actually resolve in the shipped image, not just on the dev machine.

---

## ACCEPTANCE CRITERIA

- [ ] Sidebar order is Dashboard, Sandbox, then the 7 tools, then Settings.
- [ ] `AppLayout` exists; `App.tsx` no longer duplicates the Sidebar/wrapper JSX per route.
- [ ] Visiting any of the 7 gated tool routes with no running session shows a locked in-place screen (sidebar/dock still visible), not a redirect.
- [ ] `SandboxManager` in the Go backend really spawns `bwrap`, tracks its PID, and can report real status over `GET /api/v1/sandboxes/{id}`.
- [ ] The dock renders a real, live `noVNC` framebuffer stream (not the old placeholder), and survives navigating between at least 3 different tool pages without disconnecting.
- [ ] Stopping the sandbox live-updates every gated route back to locked, with no page reload.
- [ ] Dashboard's loaded project name appears as the pre-filled target label on the Sandbox launch form.
- [ ] `useSandboxSession()` exists and at least one tool page (Proxy) demonstrably reads live session data instead of a hardcoded string.
- [ ] `Containerfile` installs `bubblewrap`, `xorg-x11-server-Xvfb`, `x11vnc` and still builds.
- [ ] All Level 1–3 validation commands pass with zero errors.

---

## COMPLETION CHECKLIST

- [ ] All 6 phases completed in order (Phase 1/2 may run in parallel)
- [ ] Each task's validation command passed immediately after that task
- [ ] Full test suite passes (Go `go test ./...`, frontend `npx jest`)
- [ ] No `tsc`/`go vet`/lint errors
- [ ] Manual validation steps 1–8 all confirmed in a real running instance
- [ ] Container image builds with the new OS packages
- [ ] Acceptance criteria all met

---

## OPEN QUESTIONS / ASSUMPTIONS

- **Assumed** — `sandbox_implementation.md` belongs at `packages/KNIRVENGINE/gui/sandbox_implementation.md` (co-located with the GUI it primarily describes, even though it also specifies Go backend work). Confirm before execution if a different location (package root, repo root) is preferred.
- **Assumed** — single active `SandboxSession` app-wide; no session switcher UI. Confirmed with user directly.
- **Assumed** — "dock to the bottom of the content area" means to the right of the sidebar (content-area width), not full browser width. This matches the user's literal wording; flagged here only because it's a real layout decision, not because it's actually in doubt.
- **Open** — exact UBI9 package names for `x11vnc`/`Xvfb` may require enabling EPEL (see Containerfile task's Gotcha) — needs a real build attempt to confirm, can't be verified from reading files alone.
- **Open** — whether `RequirePermission`-style per-route auth should gate `/api/v1/sandboxes/*` beyond the existing global security middleware (e.g., should launching a sandbox require a specific role/permission beyond "authenticated"?). Plan currently assumes parity with `terminal_manager.go`'s handlers (global middleware only, no extra per-route check) since sandbox and terminal are structurally the same kind of "operator gets a shell/session" capability. Flag before execution if sandbox launch should be root/bootnode-only.
- **Open** — `x11vnc`'s default auth model is none/VNC-password; this plan doesn't add a VNC password (relies on the WS bridge being backend-authenticated via the existing global middleware instead, with `x11vnc` itself only reachable via `127.0.0.1` from the Go process, never exposed on a public port). Confirm this is an acceptable security posture before shipping past a dev environment.

## NOTES (open canvas)

**Why a new `SandboxManager` instead of extending `ApplicationConnection`?** `TargetSystemConnection`'s interface (`Connect`/`Disconnect`/`Execute`/`GetStatus`) is a good fit for simple connect/execute targets (browser, filesystem, database) but doesn't model "this target is actually 2-3 coordinated subprocesses with a live output stream and a WebSocket audience" — that's exactly what `TerminalManager` already models for shell sessions. A sandbox session is much closer to a terminal session (long-lived process, streamed output, explicit start/stop lifecycle) than to the lighter `TargetSystemConnection` targets. Reusing `TargetSystemService` only for the target's **identity** (optionally registering a `TargetSystem` row of type `TargetTypeApplication` alongside the `SandboxSession`, for cross-cutting visibility in whatever UI already lists targets) keeps both existing services doing what they're already good at, rather than stretching one interface to cover a shape it wasn't designed for.

**Why Go-native VNC bridge over `websockify`?** Already justified in ARCHITECTURE — no Python in the runtime image, `gorilla/websocket` already a dependency, keeps the whole backend single-language. If a bridge more full-featured than a byte-pump is ever needed (target-side reconnect logic, multiple simultaneous VNC ports), revisit — but that's speculative for a v1.

**Tradeoff considered and rejected**: building the sandbox as a fully separate Rust/Tokio daemon per the original `tools.md` reference architecture (gRPC over Unix socket, dedicated "Syndicate Daemon"). Rejected for this phase because it introduces a second language/toolchain/build pipeline into a package that is currently 100% Go, for a capability (subprocess orchestration + WebSocket streaming) the existing Go stack already handles fine via `TerminalManager`'s precedent. If the sandbox subsystem's scope grows significantly (e.g., genuinely needs gRPC bidirectional streaming to multiple tool backends simultaneously), extracting it into a standalone daemon later is a natural refactor — the `SandboxManager`'s REST+WS surface as designed here would translate cleanly to a proxied-through-Go-to-daemon architecture without changing the frontend contract.

**Data-flow sketch (single session, happy path):**
```
Dashboard (pick project)
   │  targetLabel
   ▼
SandboxContext.setTargetLabel
   │
   ▼
Sandbox → Bubblewrap (fill targetCommand, launch)
   │  POST /api/v1/sandboxes
   ▼
SandboxManager.CreateSession → Start()
   │  spawns: Xvfb :99 → bwrap(...) → x11vnc -shared
   │  status: idle → provisioning → running   (streamed over /ws)
   ▼
SandboxContext.session.status === 'running'
   │
   ├──▶ RequireSandbox unlocks all 7 tool routes
   └──▶ SandboxDock connects RFB to /vnc, renders live framebuffer
```

## AMENDMENTS

(none yet)
