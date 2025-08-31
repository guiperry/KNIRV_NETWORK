# KNIRV-NEXUS Completion Plan

## Executive Summary

This plan translates the gaps identified in `Nexus_Gap_Analysis.md` into concrete, incremental workstreams with code-aligned tasks, deliverables, and acceptance criteria. It is tailored to the current KNIRV-NEXUS structure:
- Unified Go backend with modular services: DVE Manager, Validation Core, Agent Server, Data Engine, DNS, CDE
- Frontend: Next.js/TypeScript with Socket.io-based real-time hooks
- Backend includes JWT middleware, libp2p skeleton, SSE, and BuntDB data engine

Primary objectives:
- Close security, real-time, and validation gaps
- Align frontend/backend APIs and real-time transport
- Implement operational modes and production deployment parity (K8s)
- Establish comprehensive integration and E2E testing

Estimated duration: 6–8 weeks.

---

## Scope & Assumptions

- **Scope**
  - Backend Go services under `KNIRVNEXUS/backend/**`
  - Frontend Next.js under `KNIRVNEXUS/src/**`
  - K8s manifests under `KNIRVNEXUS/k8s/**`
  - Unified server in `KNIRVNEXUS/backend/main.go`
  - P2P under `KNIRVNEXUS/backend/pkg/p2p/**`
  - SSE under `KNIRVNEXUS/backend/pkg/sse/**`
  - JWT middleware under `KNIRVNEXUS/backend/internal/web/middleware/**`

- **Assumptions**
  - Netlify functions are generated at build; do not edit generated functions. Only edit `.ts` source and the migration script when needed.
  - JWT middleware exists (confirmed) but requires user management endpoints and role-based enforcement across routes.
  - P2P manager exists but is partial; will be expanded with DHT  GossipSub.
  - Frontend currently expects Socket.io; backend uses Go → decision: provide a small Socket.io bridge in Next.js runtime and/or switch frontend to standard WebSocket/SSE hooks with minimal change.

---

## Current State Summary (Observed)

- **Auth**: JWT middleware present at `backend/internal/web/middleware/auth.go`. `main.go` wires it via `middleware.NewAuthMiddleware(...)`. Missing: login/refresh/revoke endpoints and user store.
- **P2P**: `backend/pkg/p2p/dve_p2p_manager.go` exists; needs DHT, GossipSub, and message routing for validation tasks/results.
- **Real-time**: Frontend `src/lib/socket.ts` uses Socket.io. Backend has `pkg/sse/` and Data Engine WS/SSE. A bridge or standardization is required.
- **Validation**: Structure present at `backend/internal/services/validation/**`. Core execution and proof generation are partial.
- **Operational Modes**: Not wired. Needs config/flag-based GUI vs. headless.
- **K8s**: Manifests exist but assume microservices. Need unified deployment alignment.

---

## Workstreams

### 1) Security & Authentication

**Objectives**
- Finalize JWT auth: login, refresh, revoke; role-based guards; config-backed secret management.
- Persist users/roles in BuntDB or a simple local user store initially.

**Deliverables**
- Auth API: `/api/auth/login`, `/api/auth/refresh`, `/api/auth/revoke`, `/api/auth/me`
- Role-based middleware enforcement across DVE/Validation routes
- Config-driven `Security.JWTSecret` and rotation plan

**Key changes**
- Add an `auth` service or handlers under `backend/internal/web` or a small `backend/internal/services/auth`
- Wire routes in `main.go` before other protected routes

**Go Snippet: Login Handler and Route Registration**
```go
// backend/internal/web/auth_handlers.go
package web

import (
  "encoding/json"
  "net/http"
  "time"

  "nexus-backend/internal/web/middleware"
  "nexus-backend/internal/database"
)

type AuthHandlers struct {
  db   *database.BuntDBManager
  auth *middleware.AuthMiddleware
}

func NewAuthHandlers(db *database.BuntDBManager, auth *middleware.AuthMiddleware) *AuthHandlers {
  return &AuthHandlers{db: db, auth: auth}
}

type LoginRequest struct {
  Username string `json:"username"`
  Password string `json:"password"`
}
type LoginResponse struct {
  Token string `json:"token"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
  var req LoginRequest
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, "invalid request", http.StatusBadRequest)
    return
  }
  // TODO: Replace with proper user store  password hash check
  if req.Username == "" || req.Password == "" {
    http.Error(w, "invalid credentials", http.StatusUnauthorized)
    return
  }
  role := "user"
  if req.Username == "admin" {
    role = "admin"
  }
  token, err := h.auth.GenerateToken("u:"+req.Username, req.Username, role, 12*time.Hour)
  if err != nil {
    http.Error(w, "failed to issue token", http.StatusInternalServerError)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
```

```go
// backend/main.go (inside setupRoutes)
authMiddleware, err := middleware.NewAuthMiddleware(s.db, s.config.Security.JWTSecret)
if err != nil {
  log.Printf("Warning: Failed to create auth middleware: %v", err)
  authMiddleware = nil
}

// Auth routes
if authMiddleware != nil {
  authHandlers := web.NewAuthHandlers(s.db, authMiddleware)
  s.router.HandleFunc("/api/auth/login", authHandlers.Login).Methods("POST")
  // TODO: add refresh, revoke, me endpoints
}
```

**Acceptance Criteria**
- Protected endpoints reject requests without valid Bearer token
- Login issues JWT; revoke invalidates token; refresh returns new token
- Admin role can access privileged operations (e.g., node management)

---

### 2) Real-time Communication Alignment (Hybrid WS + SSE)

**Objectives**
- Standardize backend on native WebSocket for the unified Go binary (GUI mode).
- Use Server-Sent Events (SSE) for serverless deployments (Netlify) for robustness and cost-effectiveness.
- Provide a reusable frontend abstraction that selects WS or SSE based on environment.

**Recommendation**
- Backend: expose a WebSocket endpoint (e.g., GET `/ws`) that publishes system, DVE, and validation events.
- Netlify: expose an SSE function (e.g., `/.netlify/functions/nexus-sse`) that streams the same event types.
- Frontend: add a `realtime-service.ts` that auto-selects WS or SSE using env flags and re-emits typed events to the UI.

**Deliverables**
- Go: WS endpoint (or reuse Data Engine WS if already broadcasting desired topics).
- Netlify: SSE function that forwards events from backend/event bus to the client.
- Frontend: `src/lib/realtime-service.ts` with environment-driven transport selection and a small event API.

**TypeScript Snippet: Reusable Realtime Service (WS/SSE switch)**
```ts
// src/lib/realtime-service.ts
import { EventEmitter } from 'events'

type Transport = 'ws' | 'sse' | 'none'
type EventName =
  | 'connected' | 'disconnected' | 'error' | 'message'
  | 'dve-node-updated' | 'validation-task-updated'
  | 'system-notification'

export class RealtimeService extends EventEmitter {
  private ws: WebSocket | null = null
  private sse: EventSource | null = null
  private mode: Transport = 'none'
  private reconnect: number | null = null

  constructor() {
    super()
    if (typeof window !== 'undefined') this.connect()
  }

  get transport(): Transport { return this.mode }

  disconnect() {
    if (this.reconnect) window.clearTimeout(this.reconnect)
    this.ws?.close(); this.ws = null
    this.sse?.close(); this.sse = null
    if (this.mode !== 'none') { this.mode = 'none'; this.emit('disconnected') }
  }

  private connect() {
    const forced = (process.env.NEXT_PUBLIC_REALTIME_TRANSPORT || '').toLowerCase()
    const netlify = (process.env.NEXT_PUBLIC_NETLIFY_ENV || '').toLowerCase() === 'true'
    const hostIsNetlify = /\.netlify\.app$/.test(window.location.hostname)
    const choice: Transport =
      forced === 'ws' || forced === 'sse' ? (forced as Transport)
      : (netlify || hostIsNetlify) ? 'sse' : 'ws'
    choice === 'ws' ? this.connectWS() : this.connectSSE()
  }

  private connectWS() {
    const explicit = process.env.NEXT_PUBLIC_WS_URL
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = explicit || `${proto}//${window.location.host}/ws`
    this.ws = new WebSocket(url); this.mode = 'ws'
    this.ws.onopen = () => this.emit('connected', { type: 'ws' })
    this.ws.onmessage = (e) => this.handle(e.data)
    this.ws.onerror = (e) => this.emit('error', e)
    this.ws.onclose = () => {
      this.emit('disconnected')
      this.reconnect = window.setTimeout(() => this.connectWS(), 5000)
    }
  }

  private connectSSE() {
    const url = process.env.NEXT_PUBLIC_SSE_URL || '/.netlify/functions/nexus-sse'
    this.sse = new EventSource(url); this.mode = 'sse'
    this.sse.onopen = () => this.emit('connected', { type: 'sse' })
    this.sse.onmessage = (e) => this.handle(e.data)
    ;['dve-node-updated','validation-task-updated','system-notification'].forEach((ev) => {
      this.sse!.addEventListener(ev, (e: MessageEvent) => {
        try { this.emit(ev as EventName, JSON.parse(e.data)) }
        catch { this.emit(ev as EventName, (e as any).data) }
      })
    })
    this.sse.onerror = (e) => this.emit('error', e)
  }

  private handle(raw: any) {
    try {
      const data = typeof raw === 'string' ? JSON.parse(raw) : raw
      const name = (data.event || data.type) as EventName
      const payload = data.payload ?? data.data ?? data
      if (typeof name === 'string') { this.emit(name, payload); return }
      this.emit('message', data)
    } catch { this.emit('message', raw) }
  }
}

// Singleton usage in app code:
// export const realtimeService = new RealtimeService()
```

Go WS endpoint (sketch)
```go
// backend/main.go (add near top of file)
// import "github.com/gorilla/websocket"
var wsUpgrader = websocket.Upgrader{
  // TODO: Restrict origins in production
  CheckOrigin: func(r *http.Request) bool { return true },
}

// backend/main.go (inside setupRoutes)
s.router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
  conn, err := wsUpgrader.Upgrade(w, r, nil)
  if err != nil {
    log.Printf("ws upgrade error: %v", err)
    return
  }
  defer conn.Close()

  // Example: welcome + echo; replace with event-bus forwarding
  _ = conn.WriteJSON(map[string]any{
    "event":   "system-notification",
    "payload": "Welcome to KNIRV-NEXUS Realtime",
  })

  for {
    _, msg, err := conn.ReadMessage()
    if err != nil { break }
    _ = conn.WriteMessage(websocket.TextMessage, msg)
  }
})
```

Netlify SSE function (outline)
```ts
// netlify/edge-functions/nexus-sse.ts (Edge Function recommended)
export default async function handler(req: Request) {
  const headers: Record<string, string> = {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
  }

  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      const enc = new TextEncoder()
      const send = (event: string, data: unknown) => {
        const payload = typeof data === 'string' ? data : JSON.stringify(data)
        controller.enqueue(enc.encode(`event: ${event}\n`))
        controller.enqueue(enc.encode(`data: ${payload}\n\n`))
      }

      // Heartbeat to keep connection alive
      const hb = setInterval(() => controller.enqueue(enc.encode(`: heartbeat\n\n`)), 15000)

      // TODO: subscribe to your backend/event bus and call send('dve-node-updated', payload)
      // Example (pseudo):
      // bus.on('dve-node-updated', (p) => send('dve-node-updated', p))
      // bus.on('validation-task-updated', (p) => send('validation-task-updated', p))

      // Cleanup on disconnect (if supported by runtime)
      ;(req as any).signal?.addEventListener?.('abort', () => clearInterval(hb))
    },
  })

  return new Response(stream, { headers })
}
```

Notes:
- **Env flags**: `NEXT_PUBLIC_REALTIME_TRANSPORT` ('ws'|'sse'), `NEXT_PUBLIC_NETLIFY_ENV` ('true'), `NEXT_PUBLIC_WS_URL`, `NEXT_PUBLIC_SSE_URL`.
- **UI hooks** should consume `realtimeService.on('dve-node-updated', ...)` instead of Socket.io.

**Acceptance Criteria**
- Unified binary: frontend connects via WebSocket to `/ws` and receives live DVE/validation updates.
- Netlify portal: frontend connects via SSE function and receives the same events.
- No Socket.io dependency remains; hooks operate through the transport-agnostic service.

---

### 3) Operational Modes (GUI vs Headless)

**Objectives**
- Add command-line flags and/or config to toggle:
  - GUI mode: serve static exported frontend from Go server (or start Next.js separately in dev)
  - Headless mode: run APIs only

**Deliverables**
- `--mode=gui|headless` flag; `Config.Operational.Mode` override
- In GUI mode, serve `out/` via Go file server or embed assets

**Go Snippet: Flags and Mode Handling**
```go
// backend/main.go (main())
var configFile = flag.String("config", "", "Path to configuration file")
mode := flag.String("mode", "", "Operational mode: gui|headless (overrides config)")
flag.Parse()
// ...
server, err := NewServer(config)
if err != nil { log.Fatalf("Failed to create server: %v", err) }
if *mode != "" {
  config.Operational.Mode = *mode
}
if err := server.Start(); err != nil {
  log.Fatalf("Failed to start server: %v", err)
}
```

```go
// backend/main.go (inside setupRoutes)
if s.config.Operational.Mode == "gui" {
  // Serve exported Next.js site from KNIRVNEXUS/out
  // Adjust pathing if backend runs from backend/ working dir
  staticDir := filepath.Join("..", "out")
  if _, err := os.Stat(staticDir); err == nil {
    fs := http.FileServer(http.Dir(staticDir))
    s.router.PathPrefix("/").Handler(fs)
  } else {
    log.Printf("GUI mode enabled but staticDir not found: %s", staticDir)
  }
}
```

**Acceptance Criteria**
- `--mode=headless` serves only APIs
- `--mode=gui` serves static frontend and APIs together

---

### 4) API Endpoint Completion

**Objectives**
- Replace placeholders with full CRUD and query endpoints for:
  - DVE nodes (register/update/list/metrics)
  - Validation tasks (create/assign/progress/results)
  - Health/system metrics

**Deliverables**
- Completed handlers in `backend/internal/services/dvemanager` and `validation`
- Auth guards (`RequireAuth`, `RequireRole`) applied

**Go Snippet: Example Protected Route**
```go
// backend/internal/services/validation/routes.go (excerpt)
func (vc *ValidationCore) RegisterRoutes(r *mux.Router, auth *middleware.AuthMiddleware) {
  s := r.PathPrefix("/api/validation").Subrouter()
  if auth != nil {
    s.Use(auth.RequireAuth)
  }
  s.HandleFunc("/tasks", vc.CreateTaskHandler).Methods("POST")            // admin/user
  s.HandleFunc("/tasks", vc.ListTasksHandler).Methods("GET")              // user
  s.HandleFunc("/tasks/{id}", vc.GetTaskHandler).Methods("GET")
  s.HandleFunc("/tasks/{id}/assign", vc.AssignTaskHandler).Methods("POST") // admin
  s.HandleFunc("/tasks/{id}/results", vc.SubmitResultHandler).Methods("POST")
}
```

**Acceptance Criteria**
- CRUD works with persistence in BuntDB via Data Engine abstractions
- Role checks enforced; validation returns meaningful errors

---

### 5) Validation Engine Completion

**Objectives**
- Implement execution flow: queue → assign → execute → score → result → persist
- Add deterministic scoring and result schema alignment with `models.ValidationResult`

**Deliverables**
- Completed `validation_executor.go` with pluggable execution strategies
- Result proof placeholder (see Workstream 8)

**Go Snippet: Executor Skeleton**
```go
// backend/internal/services/validation/validation_executor.go (excerpt)
package validation

import (
  "context"
  "time"
  "nexus-backend/internal/models"
)

type Executor interface {
  Execute(ctx context.Context, task *models.ValidationTask) (*models.ValidationResult, error)
}

type DefaultExecutor struct{}

func (e *DefaultExecutor) Execute(ctx context.Context, task *models.ValidationTask) (*models.ValidationResult, error) {
  start := time.Now()
  // TODO: route by task.Type, run testcases, compute score
  // Placeholder: success with fixed score
  res := &models.ValidationResult{
    ID:              "res-" & task.ID,
    TaskID:          task.ID,
    ValidatorNodeID: task.AssignedNodeID,
    Status:          "success",
    Score:           1.0,
    Results:         map[string]any{"summary": "ok"},
    TestResults:     []models.TestResult{},
    Proof:           "", // set by proof system
    TEEAttestation:  "",
    ExecutionTime:   time.Since(start),
    CreatedAt:       time.Now(),
    Signature:       "",
  }
  return res, nil
}
```

**Acceptance Criteria**
- Tasks transition through lifecycle and persist results
- Basic scoring and result models match frontend expectations

---

### 6) TEE & Attestation (Software Simulation First)

**Objectives**
- Implement TEE attestation structures and a software-verification pathway
- Store and validate attestation on result submission

**Deliverables**
- `TEEVerifier` interface; `SoftwareTEEVerifier` implementation
- Integrate into validation result path

**Go Snippet: TEE Verification Stub**
```go
// backend/internal/services/validation/tee_verifier.go
package validation

import "nexus-backend/internal/models"

type TEEVerifier interface {
  Verify(att *models.TEEAttestation) (bool, string)
}

type SoftwareTEEVerifier struct{}

func (v *SoftwareTEEVerifier) Verify(att *models.TEEAttestation) (bool, string) {
  if att.TEEType == "software" && att.Status == "valid" {
    return true, ""
  }
  return false, "unsupported or invalid attestation"
}
```

```go
// backend/internal/services/validation/handlers.go (on SubmitResult)
// After parsing result:
if result.TEEAttestation != "" {
  // TODO: parse JSON into models.TEEAttestation
  ok, reason := vc.teeVerifier.Verify(&att)
  if !ok {
    http.Error(w, "attestation invalid: "reason, http.StatusBadRequest)
    return
  }
}
```

**Acceptance Criteria**
- Validation results with software TEE attestation pass verification
- Invalid/mismatched attestations are rejected

---

### 7) P2P Networking Completion (libp2p DHT & GossipSub)

**Objectives**
- Enable peer discovery (Kademlia DHT), pub/sub for:
  - `validation/request`
  - `validation/result`
  - `node/announcement`

**Deliverables**
- Expand `backend/pkg/p2p/dve_p2p_manager.go` with libp2p host, DHT, GossipSub
- Define message schemas per `models.P2PMessage`

**Go Snippet: PubSub Setup (Simplified)**
```go
// backend/pkg/p2p/dve_p2p_manager.go (excerpt)
package p2p

import (
  "context"
  "log"

  "github.com/libp2p/go-libp2p"
  dht "github.com/libp2p/go-libp2p-kad-dht"
  pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type DVEP2PManager struct {
  host  host.Host
  ps    *pubsub.PubSub
  subs  map[string]*pubsub.Subscription
  ctx   context.Context
  cancel context.CancelFunc
}

func NewDVEP2PManager(chainID, role string, db *buntdb.DB) (*DVEP2PManager, error) {
  ctx, cancel := context.WithCancel(context.Background())
  h, err := libp2p.New()
  if err != nil { cancel(); return nil, err }
  kdht, err := dht.New(ctx, h)
  if err != nil { cancel(); return nil, err }
  if err := kdht.Bootstrap(ctx); err != nil { cancel(); return nil, err }
  ps, err := pubsub.NewGossipSub(ctx, h)
  if err != nil { cancel(); return nil, err }
  return &DVEP2PManager{host: h, ps: ps, subs: map[string]*pubsub.Subscription{}, ctx: ctx, cancel: cancel}, nil
}

func (m *DVEP2PManager) Start() {
  topics := []string{"validation/request", "validation/result", "node/announcement"}
  for _, t := range topics {
    sub, err := m.ps.Subscribe(t)
    if err != nil { log.Printf("p2p subscribe error: %v", err); continue }
    m.subs[t] = sub
    go m.consume(t, sub)
  }
}

func (m *DVEP2PManager) Publish(topic string, data []byte) error {
  topicObj, err := m.ps.Join(topic)
  if err != nil { return err }
  return topicObj.Publish(m.ctx, data)
}
```

**Acceptance Criteria**
- Nodes discover peers and exchange validation requests/results via pubsub
- Basic topology/peer health endpoints return meaningful data

---

### 8) Cryptographic Proofs (Phase 1: Hash/Signature)

**Objectives**
- Provide minimal cryptographic guarantees for validation results:
  - TaskResult hash, signed by validator key

**Deliverables**
- `ProofBuilder` that creates proof string and signature
- Signature verification endpoint or internal check

**Go Snippet: Simple Proof Builder**
```go
// backend/internal/services/validation/proof.go
package validation

import (
  "crypto/ed25519"
  "crypto/sha256"
  "encoding/hex"
  "encoding/json"
)

type ProofBuilder struct {
  priv ed25519.PrivateKey
  pub  ed25519.PublicKey
}

func (p *ProofBuilder) Build(payload any) (proof, sigHex string, err error) {
  data, err := json.Marshal(payload)
  if err != nil { return "", "", err }
  sum := sha256.Sum256(data)
  sig := ed25519.Sign(p.priv, sum[:])
  return hex.EncodeToString(sum[:]), hex.EncodeToString(sig), nil
}
```

**Acceptance Criteria**
- Results include reproducible `Proof` and `Signature`
- Verification confirms signature over result hash

---

### 9) Kubernetes Manifests (Unified Binary)

**Objectives**
- Update K8s to deploy the unified backend (and optionally serve static GUI)
- Config via ConfigMap/Secret; health/readiness probes

**Deliverables**
- Replace per-service Deployments with one Deployment for the unified server
- Service exposing API port and (optionally) static GUI

**K8s Snippet: Deployment (Excerpt)**
```yaml
# KNIRVNEXUS/k8s/validation-core-deployment.yaml (repurpose to unified)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: knirvnexus
spec:
  replicas: 1
  selector:
    matchLabels: { app: knirvnexus }
  template:
    metadata:
      labels: { app: knirvnexus }
    spec:
      containers:
        - name: knirvnexus
          image: your-repo/knirvnexus:latest
          args: ["--mode=$(OPERATIONAL_MODE)"]
          env:
            - name: OPERATIONAL_MODE
              valueFrom:
                configMapKeyRef:
                  name: knirvnexus-config
                  key: OPERATIONAL_MODE
          ports:
            - containerPort: 7080 # API
          readinessProbe:
            httpGet: { path: /api/health, port: 7080 }
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet: { path: /health, port: 7080 }
            initialDelaySeconds: 10
            periodSeconds: 20
```

**Acceptance Criteria**
- Pod becomes ready; `/api/health` green
- Configurable mode and secrets mounted; graceful rolling updates

---

### 10) Monitoring & Observability

**Objectives**
- Add Prometheus metrics and structured logging

**Deliverables**
- `/metrics` endpoint via `promhttp`
- Basic counters/histograms for requests and validation lifecycle

**Go Snippet: Prometheus**
```go
// backend/main.go (setupRoutes)
import "github.com/prometheus/client_golang/prometheus/promhttp"
// ...
s.router.Handle("/metrics", promhttp.Handler())
```

**Acceptance Criteria**
- Metrics endpoint exposes standard and custom metrics
- Logs include request IDs and error context

---

### 11) Frontend Alignment

**Objectives**
- Align endpoints and payloads used by the dashboard to those actually provided in the backend (e.g., `/api/dve-nodes`, `/api/validation-tasks`)
- Implement auth context usage with real JWT

**Deliverables**
- Update `src/lib/auth-context.tsx` to store and attach JWT to fetches
- Hooks consuming real-time events from the chosen bridge

**TypeScript Snippet: Attaching JWT**
```ts
// src/lib/db.ts (example fetch wrapper)
export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = (typeof window !== 'undefined') ? localStorage.getItem('jwt') : null
  const headers = new Headers(options.headers || {})
  if (token) headers.set('Authorization', `Bearer ${token}`)
  headers.set('Content-Type', 'application/json')
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_BASE}${path}`, { ...options, headers })
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return res.json()
}
```

**Acceptance Criteria**
- Dashboard loads live data (not mocks)
- Authenticated requests succeed with stored JWT

---

### 12) Testing (Integration & E2E)

**Objectives**
- Expand integration tests and add E2E tests

**Deliverables**
- Go integration tests for auth, DVE nodes, validation lifecycle
- Playwright/Cypress E2E for dashboard flows (login, list nodes, task updates)

**Go Snippet: Integration Test Skeleton**
```go
// KNIRVNEXUS/integration-tests/validation_lifecycle_test.go
package integration_tests

import "testing"

func TestValidationLifecycle(t *testing.T) {
  // 1) Login → get JWT
  // 2) Create task
  // 3) Assign task
  // 4) Submit result
  // 5) Verify persisted state and score
}
```

**Acceptance Criteria**
- CI runs backend integration tests and frontend E2E on PR
- Test reports captured under `integration-tests/reports/`

---

### 13) Documentation Alignment

**Objectives**
- Update docs to reflect unified binary, real-time transport, and K8s changes

**Deliverables**
- Update existing `.md` under KNIRVNEXUS (no new READMEs unless needed) to match reality
- Add operational mode usage to `README-DEPLOYMENT.md`

**Acceptance Criteria**
- Docs instructions reproduce a working deployment locally and on K8s

---

## Milestones & Timeline

1. **Week 1**
   - Auth endpoints (login/me/revoke)
   - Bridge for real-time (Socket.io → SSE/WS)
   - API CRUD completion (DVE/Validation)
2. **Week 2**
   - Validation executor and lifecycle
   - TEE software verifier integration
   - Basic proof builder (hash & signature)
3. **Week 3**
   - P2P: DHT & GossipSub channels (request/result/announcement)
   - Operational modes and static GUI serving
4. **Week 4**
   - K8s unified manifests update
   - Prometheus metrics and logging improvements
5. **Weeks 5–6**
   - Frontend alignment (hooks, auth, endpoints)
   - Integration and E2E tests; stabilize CI
6. **Weeks 7–8**
   - Performance tuning, hardening, and doc alignment

---

## Risks & Mitigations

- **Socket.io vs Native WS/SSE mismatch**
  - Mitigation: temporary bridge; migrate to WS long-term
- **K8s manifests drift**
  - Mitigation: consolidate into single Deployment and standardize env/config
- **P2P complexity**
  - Mitigation: phase-in; keep clear interfaces between services and P2P manager

---

## Acceptance Criteria (Global)

- End-to-end flow: login → create validation task → assign → execute → result with proof → observe live updates on dashboard
- P2P messaging for validation request/result functional in local multi-node test
- K8s deployment runs the unified binary; health and metrics endpoints operational
- Tests: integration & E2E green; coverage improved for core paths

---

## Notes Tied to Current Codebase

- JWT middleware exists at `backend/internal/web/middleware/auth.go` and is already used in `main.go`. This plan adds user endpoints and ensures role enforcement across routes.
- P2P manager exists in `backend/pkg/p2p/dve_p2p_manager.go`. This plan adds DHT & GossipSub and topic routing aligned to `models.P2PMessage`.
- SSE manager exists at `backend/pkg/sse/sse_manager.go`. We unify real-time via a bridge (short-term), moving to native WS long-term.
- Respect Netlify directive: only edit `.ts` sources and migration script. Do not edit generated functions; check generated outputs after build.

---

## IMPLEMENTATION PROGRESS TRACKING

### Phase 1: Security & Authentication - [IN PROGRESS]

**Completed:**
- [x] JWT middleware exists in `backend/internal/web/middleware/auth.go`
- [x] Auth middleware is wired in `main.go`
- [x] User claims, token generation, and validation implemented
- [x] Role-based middleware (RequireAuth, RequireRole) implemented
- [x] Created auth handlers in `backend/internal/web/auth_handlers.go`
- [x] Added login, refresh, revoke, and me endpoints
- [x] Wired auth routes in `backend/cmd/nexus-server/main.go`
- [x] Added JWT secret configuration to development.yaml

**In Progress:**
- [ ] Test authentication endpoints and resolve port configuration
- [ ] Verify auth middleware integration
- [ ] Test role-based access control

**Next Steps:**
- Implement `/api/auth/login` endpoint
- Add user store integration with BuntDB
- Create frontend auth context

### Remaining Phases (Not Started):
- [ ] Phase 2: Real-time Communication Alignment
- [ ] Phase 3: Operational Modes
- [ ] Phase 4: API Endpoint Completion
- [ ] Phase 5: Validation Engine Completion
- [ ] Phase 6: TEE & Attestation
- [ ] Phase 7: P2P Networking Completion
- [ ] Phase 8: Cryptographic Proofs
- [ ] Phase 9: Kubernetes Manifests
- [ ] Phase 10: Monitoring & Observability
- [ ] Phase 11: Frontend Alignment
- [ ] Phase 12: Testing
- [ ] Phase 13: Documentation Alignment

**Current Focus:** Phase 1 - Security & Authentication (70% complete)
