# KNIRVSERVER Governance Implementation Plan

**Reference:** `governanceToolkitComparison.html` — AGT gap analysis  
**Scope:** Native Go implementations only. No Python FFI, no AGT library imports.  
**Baseline:** Phases 1–4 scaffolding is live (`identitybridge`, `policyadapter`, `reliability`, `mcphardening` + `governance_handlers.go` routes).

---

## Current State Summary

All four roadmap phases from the comparison document have been scaffolded as Go packages wired into the KNIRVSERVER HTTP layer:

| Phase | Package | Route prefix | Status |
|-------|---------|--------------|--------|
| 1 — Identity bridge | `services/identitybridge` | `/api/v1/governance/identity` | Scaffolded |
| 2 — Policy adapter | `services/policyadapter` | `/api/v1/governance/policy` | Scaffolded |
| 3 — Reliability controls | `services/reliability` | `/api/v1/governance/reliability` | Scaffolded |
| 4 — MCP hardening | `services/mcphardening` | `/api/v1/governance/mcp` | Scaffolded |

The scaffolding covers struct definitions, basic CRUD, and in-memory state. What is missing is the functional depth that makes these layers governance-grade: policy evaluation, DID resolution, zero-trust continuous verification, compliance evidence generation, framework interoperability, and a verification CLI.

---

## Gap Analysis by Layer

### Layer 1: Identity Bridge

**Implemented:** `TrustEnvelope` creation/expiry/revocation, trust score arithmetic, `TrustMapping` for external source aliasing, federation flag, HTTP handlers.

**Missing:**

| Gap | Why it matters |
|-----|---------------|
| DID document resolution | `IdentitySourceDID` is a string constant; no W3C DID resolution happens |
| Zero-trust middleware | Envelopes are only checked on explicit API call, not per-request |
| OIDC/X509 parsing | `IdentitySourceOIDC`/`IdentitySourceX509` are declared but inert |
| Hard revocation log | `RevokeEnvelope` soft-expires; nothing external can query a revocation list |
| Continuous trust decay | Trust scores are set once and drift upward via delta; no time-based decay |

### Layer 2: Policy Adapter

**Implemented:** `NormalizedPolicyInput` schema, `PolicyEvaluation`/`NormalizedEnforcement` structs, `PortabilityContract` descriptor, `PolicySourceOPA` constant.

**Missing:**

| Gap | Why it matters |
|-----|---------------|
| Embedded Rego evaluator | `PolicySourceOPA` is a constant with no evaluation path |
| Policy-as-code DSL | Currently stores normalized inputs; no rule evaluation occurs |
| Per-request enforcement middleware | `NormalizeInput` is called manually; it doesn't intercept real requests |
| Hot-reload without restart | No file-watch or gRPC trigger to push new rules at runtime |
| GuardrailEngine integration | `policyadapter` and `guardrails` service are completely separate; no bridge |

### Layer 3: Reliability Controls

**Implemented:** `CircuitBreaker` (closed/open/half-open), `ErrorBudget`, `KillSwitch` arm/trip, `BreachEvent` log, statistics.

**Missing:**

| Gap | Why it matters |
|-----|---------------|
| DVE metric binding | Breakers track synthetic test calls; they are not wired to real DVE CPU/memory/latency |
| Escalation notification | When a kill switch fires, no other service is notified |
| SLO/SLA definitions | There is no concept of target reliability level or window |
| GuardrailEngine bridge | `ValidateResourceUsage` in guardrails is disconnected from `ReliabilityController` |
| Breach persistence | `breachEvents` are in-memory; crash loses history |

### Layer 4: MCP Hardening

**Implemented:** `ToolDefinition` registry, `PoisoningSignature` pattern matching, `ToolCallRecord` audit log, `MCPEndpoint` config, gateway statistics.

**Missing:**

| Gap | Why it matters |
|-----|---------------|
| Argument schema validation | Poisoning detection checks tool name patterns, not call argument content |
| Prompt injection detection | No analysis of string arguments for injection payloads |
| Response-side audit | Only call records exist; tool responses are not captured |
| KNIRVGATEWAY integration | MCP gateway sits in KNIRVSERVER but doesn't intercept KNIRVGATEWAY-routed MCP calls |
| Tool capability binding | No link from `ToolDefinition` to KNIRVCHAIN `CapabilityNode` records |

### Layer 5: Compliance / Verification (entirely missing)

AGT ships an `agent-governance-toolkit-cli` with a `verify` subcommand that produces cryptographically-bound compliance reports. KNIRVSERVER has `fintech/evidence_pack.go` but no cross-service compliance layer.

**Missing:**

- Compliance framework mapping (GDPR Article 22, SOC 2 CC6, ISO 42001 controls) over real KNIRVSERVER events
- Cross-service evidence correlator that joins identity + policy + reliability + MCP audit records into a single report
- HMAC chain over the audit log so evidence cannot be retroactively altered
- `knirvctl governance verify` CLI subcommand

### Layer 6: Framework Interoperability (partially missing)

AGT differentiates on adapter coverage (LangChain, AutoGen, OpenAI Agents SDK, etc.). KNIRVSERVER exposes REST. What's missing is a governance middleware SDK that external Go-based runtimes can embed without making REST calls.

---

## Implementation Plan

### Phase 1-A: DID Resolution Engine

DID resolution belongs entirely in **KNIRVORACLE** (`packages/KNIRVORACLE/`, module `github.com/knirvcorp/knirvoracle`), which is the authoritative trust and identity service in the KNIRV network. KNIRVSERVER's `identitybridge` does not implement its own resolver — it calls KNIRVORACLE's HTTP API (`:1317`) when `IdentitySourceDID` is specified.

#### 1-A-i: Resolver implementation in KNIRVORACLE

**New files:**

- `packages/KNIRVORACLE/internal/oracle/did/resolver.go` — core resolver logic
- `packages/KNIRVORACLE/internal/oracle/did/document.go` — DID document types
- `packages/KNIRVORACLE/internal/oracle/routes/did.go` — HTTP handler wired into the existing routes in `routes/routes.go`

The resolver uses KNIRVORACLE's existing `internal/oracle/crypto/` package (already has `ecdsa.go`, `address.go`, `keccak.go`) and adds Ed25519 support alongside it:

```go
// packages/KNIRVORACLE/internal/oracle/did/document.go
package did

type DIDDocument struct {
    Context            []string             `json:"@context"`
    ID                 string               `json:"id"`
    VerificationMethod []VerificationMethod `json:"verificationMethod"`
    Authentication     []string             `json:"authentication"`
    AssertionMethod    []string             `json:"assertionMethod"`
    Created            time.Time            `json:"created"`
    Updated            time.Time            `json:"updated"`
}

type VerificationMethod struct {
    ID                 string `json:"id"`
    Type               string `json:"type"`               // Ed25519VerificationKey2020
    Controller         string `json:"controller"`
    PublicKeyMultibase string `json:"publicKeyMultibase"` // multibase-encoded Ed25519 pubkey
}
```

```go
// packages/KNIRVORACLE/internal/oracle/did/resolver.go
package did

type Method string

const (
    MethodKNIRV Method = "knirv" // did:knirv:<nodeID> — stored in oracle ledger
    MethodKey   Method = "key"   // did:key:<multibase-pubkey> — resolved locally
)

type Resolver struct {
    store DIDStore // interface over oracle's internal ledger
}

type DIDStore interface {
    Get(did string) (*DIDDocument, error)
    Put(doc *DIDDocument) error
    Deactivate(did string) error
}

func NewResolver(store DIDStore) *Resolver
func (r *Resolver) Resolve(did string) (*DIDDocument, error)    // dispatches by method
func (r *Resolver) Register(doc *DIDDocument) error
func (r *Resolver) Deactivate(did string) error
func (r *Resolver) resolveKNIRV(id string) (*DIDDocument, error) // ledger lookup
func (r *Resolver) resolveKey(encoded string) (*DIDDocument, error) // local Ed25519 decode
```

`did:key:` resolution decodes the multibase-encoded Ed25519 public key inline using `golang.org/x/crypto/ed25519` — no network call. `did:knirv:` documents are stored in KNIRVORACLE's own ledger (not KNIRVCHAIN), keyed by node ID, and written when a node first registers with the oracle.

**New HTTP routes added to `packages/KNIRVORACLE/internal/oracle/routes/did.go`** and registered in `routes/routes.go`:

```
POST /did/register     — register a new did:knirv: document
GET  /did/{did}        — resolve any supported DID (did:knirv: or did:key:)
POST /did/{did}/deactivate — deactivate a did:knirv: document
```

These sit at the same `:1317` port as all other KNIRVORACLE routes.

#### 1-A-ii: KNIRVSERVER identity bridge integration

`identitybridge/service.go` does not get a `did.go` file. Instead, when `IdentitySourceDID` is specified in `CreateEnvelope`, the bridge makes a single HTTP GET to KNIRVORACLE:

```
GET http://localhost:1317/did/{did}
```

It parses the returned `DIDDocument`, extracts the `Ed25519VerificationKey2020` verification method, verifies the envelope request's signature against that key, and if valid produces the `TrustEnvelope`. The oracle URL is read from config (`governance.did.oracle_url`, defaulting to `http://localhost:1317`).

This keeps KNIRVSERVER free of any DID ledger responsibility and respects the existing boundary: KNIRVORACLE is the root of trust, KNIRVSERVER enforces downstream.

---

### Phase 1-B: Zero-Trust Per-Request Middleware

**File:** `packages/KNIRVSERVER/backend/internal/web/middleware/governance.go`

This lives in the existing `middleware` package alongside `auth.go`, `middleware.go`, and `pwa.go`. It follows the same struct-with-constructor pattern as `AuthMiddleware` and returns `func(http.Handler) http.Handler` (the Gorilla Mux HTTP handler form — matching `CORSMiddlewareHTTP` in the same package).

**Integration with `AuthMiddleware`:** The governance middleware runs *after* JWT auth in the middleware chain. It reads the `AuthContext` already stored at `AuthContextKey` by `AuthMiddleware`, so it does not re-validate the JWT. Instead it reads the `X-KNIRV-Identity` header (a `TrustEnvelope.IdentityID`) and performs the governance trust check using the identity already authenticated:

```
CORSMiddlewareHTTP() → AuthMiddleware.Require() → GovernanceMiddleware.Enforce() → handler
```

The middleware:

1. Reads `AuthContextKey` from the request context (set by `AuthMiddleware`) — returns 401 if absent (auth must precede governance).
2. Reads `X-KNIRV-Identity` header for the `TrustEnvelope.IdentityID`.
3. Looks up the envelope from `IdentityBridge` and checks `RevocationList.IsRevoked`.
4. Checks expiry and minimum trust level for the route's tier.
5. Applies a time-based trust decay (`TrustDecayRate` config — e.g. 0.001 per hour since `IssuedAt`).
6. If trust falls below threshold, returns HTTP 401 with `WWW-Authenticate: KNIRV-Identity`.
7. Stores the resolved `TrustEnvelope` in the request context under its own `contextKey` for downstream handlers.

```go
package middleware

// GovernanceTrustKey is the context key for the resolved TrustEnvelope.
const GovernanceTrustKey contextKey = "governance_trust"

type GovernanceMiddleware struct {
    bridge    *identitybridge.IdentityBridge
    revoked   *identitybridge.RevocationList
    tiered    map[string]identitybridge.TrustLevel // route prefix → minimum level
    decayRate float64
}

func NewGovernanceMiddleware(
    bridge *identitybridge.IdentityBridge,
    revoked *identitybridge.RevocationList,
    tiers map[string]identitybridge.TrustLevel,
    decayRate float64,
) *GovernanceMiddleware

func (gm *GovernanceMiddleware) Enforce(next http.Handler) http.Handler
func GovernanceTrustFromContext(ctx context.Context) *identitybridge.TrustEnvelope
```

`GovernanceTrustFromContext` is the public accessor downstream handlers use, mirroring `GetRequestID` in `middleware.go`.

Route tier table (configured in `config/governance.yaml`):

```yaml
governance:
  zero_trust:
    tiers:
      "/api/v1/governance/identity": high
      "/api/v1/governance/policy":   medium
      "/api/v1/governance/reliability": high
      "/api/v1/governance/mcp":      high
      "/oracle/":                    maximum
    decay_rate_per_hour: 0.001
```

---

### Phase 1-C: Hard Revocation List

**File:** `packages/KNIRVSERVER/backend/internal/services/identitybridge/revocation.go`

A concurrent-safe revocation list with a tamper-evident append-only log:

```go
type RevocationEntry struct {
    IdentityID string    `json:"identity_id"`
    NodeID     string    `json:"node_id"`
    Reason     string    `json:"reason"`
    RevokedAt  time.Time `json:"revoked_at"`
    RevokedBy  string    `json:"revoked_by"`
    ChainHash  string    `json:"chain_hash"` // SHA-256 of previous entry + this entry
}

type RevocationList struct {
    mu      sync.RWMutex
    entries []*RevocationEntry
    index   map[string]int // identityID → slice position
    tip     string         // hash of last entry
}

func (rl *RevocationList) Revoke(identityID, nodeID, reason, revokedBy string) *RevocationEntry
func (rl *RevocationList) IsRevoked(identityID string) bool
func (rl *RevocationList) VerifyChain() error  // recomputes all hashes
func (rl *RevocationList) Export() []*RevocationEntry
```

`IdentityBridge.GetEnvelope` checks `RevocationList.IsRevoked` before returning a valid envelope.

**New HTTP routes:**
- `GET /api/v1/governance/identity/revocation` — export full list
- `GET /api/v1/governance/identity/revocation/verify` — verify chain integrity

---

### Phase 2-A: Embedded Rego Evaluator

**File:** `packages/KNIRVSERVER/backend/internal/services/policyadapter/rego.go`

Embed a Rego policy evaluator using `github.com/open-policy-agent/opa/rego` (pure Go, no cgo). This keeps enforcement inside KNIRVSERVER without shelling out or calling an external OPA server.

```go
type RegoEvaluator struct {
    mu       sync.RWMutex
    modules  map[string]string // policyID → Rego source
    prepared map[string]*rego.PreparedEvalQuery
}

func NewRegoEvaluator() *RegoEvaluator
func (re *RegoEvaluator) LoadPolicy(policyID, regoSource string) error
func (re *RegoEvaluator) UnloadPolicy(policyID string) error
func (re *RegoEvaluator) Evaluate(policyID string, input map[string]interface{}) (*PolicyEvaluation, error)
func (re *RegoEvaluator) HotReload(policyID, newSource string) error  // atomic swap
```

`PolicyAdapter` gains a `RegoEvaluator` field. When `NormalizeInput` is called and a Rego policy is registered for the `action_type`, the evaluator runs and the result is fed directly into `RecordEnforcement`. No manual invocation required.

**New HTTP routes:**
- `POST /api/v1/governance/policy/rego` — load a Rego module
- `DELETE /api/v1/governance/policy/rego/{id}` — unload
- `POST /api/v1/governance/policy/rego/{id}/reload` — hot-reload source

**Go module addition** (`backend/go.mod`):
```
github.com/open-policy-agent/opa v0.67.1
```

---

### Phase 2-B: Per-Request Policy Enforcement Middleware

**File:** `packages/KNIRVSERVER/backend/internal/web/middleware/governance.go` (same file as Phase 1-B)

Extend `GovernanceMiddleware` to accept a `*policyadapter.PolicyAdapter` and `*policyadapter.RegoEvaluator` at construction. After the identity check passes, `Enforce` calls `PolicyAdapter.NormalizeInput` using the request's action type (derived from HTTP method + route prefix) and then `RegoEvaluator.Evaluate`. The `TrustEnvelope` resolved in the identity step is injected as a context field in the policy input, so Rego rules can condition on trust level directly.

If the policy decision is `deny`, the middleware returns HTTP 403 with the matched rules serialized in the body. No further handler invocation occurs.

The full enforcement chain through the existing middleware stack becomes:

```
CORSMiddlewareHTTP()
  → RequestIDMiddleware()     [middleware.go — adds X-Request-ID]
  → AuthMiddleware.Require()  [auth.go — validates JWT, stores AuthContext]
  → GovernanceMiddleware.Enforce()  [governance.go — trust check → policy eval]
      → handler
```

On denial at any governance step:
```
identity expired/revoked  → 401  WWW-Authenticate: KNIRV-Identity
trust below tier minimum  → 401  WWW-Authenticate: KNIRV-Identity
policy decision = deny    → 403  {matched_rules: [...]}
```

---

### Phase 2-C: GuardrailEngine Bridge

**File:** `packages/KNIRVSERVER/backend/internal/services/guardrails/governance_bridge.go`

Wire the existing `GuardrailEngine` to `PolicyAdapter` so that every guardrail decision is also recorded as a `NormalizedEnforcement`:

```go
type GovernanceBridge struct {
    guardrails *GuardrailEngine
    policy     *policyadapter.PolicyAdapter
    identity   *identitybridge.IdentityBridge
}

func (gb *GovernanceBridge) ValidateWithGovernance(nodeID, action string, metrics map[string]float64, ctx map[string]interface{}) (*GuardrailResult, *policyadapter.NormalizedEnforcement)
```

`ValidateWithGovernance` calls `GuardrailEngine.ValidateResourceUsage`, maps the result into `NormalizeInput`, then calls `RegoEvaluator.Evaluate` if a policy is registered for the resource domain. The `NormalizedEnforcement` is persisted so compliance reports can reference guardrail decisions.

---

### Phase 3-A: DVE Metric Binding

**File:** `packages/KNIRVSERVER/backend/internal/services/reliability/dve_binding.go`

Connect `ReliabilityController` to the DVE manager so circuit breakers track real node metrics instead of synthetic calls:

```go
type DVEMetricBinding struct {
    ctrl       *ReliabilityController
    dveManager DVEMetricProvider  // interface over dvemanager.DVEManager
    ticker     *time.Ticker
    stopCh     chan struct{}
}

type DVEMetricProvider interface {
    GetNodeMetrics(nodeID string) (cpu, memoryMB, latencyMs float64, err error)
    ListActiveNodes() ([]string, error)
}

func (b *DVEMetricBinding) Start(interval time.Duration)
func (b *DVEMetricBinding) Stop()
func (b *DVEMetricBinding) syncOnce()  // called by ticker
```

`syncOnce` iterates all active DVE nodes, reads CPU/memory/latency via `DVEMetricProvider`, and calls the relevant `CircuitBreaker.RecordSuccess` or `RecordFailure` based on threshold config. When a breaker transitions to open, it also calls `ReliabilityController.ArmKillSwitch` for the node.

Thresholds configured in `config/governance.yaml`:

```yaml
governance:
  reliability:
    dve_binding:
      sync_interval: 10s
      cpu_threshold: 0.85
      memory_threshold_mb: 3072
      latency_threshold_ms: 2000
```

---

### Phase 3-B: Escalation Notification Pipeline

**File:** `packages/KNIRVSERVER/backend/internal/services/reliability/escalation.go`

When a kill switch fires, broadcast the event to registered handlers synchronously before returning:

```go
type EscalationEvent struct {
    AgentID   string    `json:"agent_id"`
    NodeID    string    `json:"node_id"`
    Reason    string    `json:"reason"`
    KillAt    time.Time `json:"kill_at"`
    Severity  string    `json:"severity"` // warn | critical | fatal
}

type EscalationHandler interface {
    OnEscalation(event EscalationEvent)
}

type EscalationPipeline struct {
    mu       sync.RWMutex
    handlers []EscalationHandler
}

func (ep *EscalationPipeline) Register(h EscalationHandler)
func (ep *EscalationPipeline) Fire(event EscalationEvent)
```

Built-in handlers registered at startup:
- `WebSocketEscalationHandler` — broadcasts over the existing WebSocket hub (`policy:kill_switch` event type)
- `GuardrailEscalationHandler` — calls `GuardrailEngine.BlockNode(nodeID)` immediately
- `PolicyEscalationHandler` — records a `NormalizedEnforcement` with decision `block`

`ReliabilityController.TripKillSwitch` gains a `pipeline *EscalationPipeline` field and calls `pipeline.Fire` after updating state.

---

### Phase 3-C: SLO Definitions

**File:** `packages/KNIRVSERVER/backend/internal/services/reliability/slo.go`

```go
type SLODefinition struct {
    ID              string        `json:"id"`
    Name            string        `json:"name"`
    NodeID          string        `json:"node_id"`
    MetricName      string        `json:"metric_name"`  // e.g. "latency_p99"
    Target          float64       `json:"target"`       // e.g. 0.99 for 99% uptime
    Window          time.Duration `json:"window"`       // e.g. 30 * 24h
    ErrorBudgetID   string        `json:"error_budget_id"`
}

type SLOTracker struct {
    mu    sync.RWMutex
    slos  map[string]*SLODefinition
    burns map[string][]burnEvent  // sloID → burn events in window
}

func (st *SLOTracker) RegisterSLO(def SLODefinition) error
func (st *SLOTracker) RecordBurn(sloID string, magnitude float64)
func (st *SLOTracker) CurrentBurn(sloID string) float64
func (st *SLOTracker) BudgetRemaining(sloID string) float64  // returns 0..1
```

When `BudgetRemaining` drops below 0.1 (10%), the tracker automatically fires `EscalationPipeline` with severity `warn`. At 0.0, severity is `critical` and the kill switch is armed.

**New HTTP routes:**
- `POST /api/v1/governance/reliability/slos`
- `GET /api/v1/governance/reliability/slos`
- `GET /api/v1/governance/reliability/slos/{id}/burn`

---

### Phase 4-A: Argument Schema Validation

**File:** `packages/KNIRVSERVER/backend/internal/services/mcphardening/schema.go`

Extend `ToolDefinition` with JSON Schema validation for call arguments:

```go
type ToolDefinition struct {
    Name           string            `json:"name"`
    Description    string            `json:"description"`
    Parameters     []string          `json:"parameters"`
    Required       bool              `json:"required"`
    AllowedIPs     []string          `json:"allowed_ips,omitempty"`
    AllowedDomains []string          `json:"allowed_domains,omitempty"`
    MaxArgs        int               `json:"max_args,omitempty"`
    Timeout        time.Duration     `json:"timeout,omitempty"`
    // NEW:
    ArgSchema      map[string]ArgDef `json:"arg_schema,omitempty"`
    MaxArgLength   int               `json:"max_arg_length,omitempty"`
}

type ArgDef struct {
    Type     string `json:"type"`    // string | number | boolean | object
    Required bool   `json:"required"`
    MaxLen   int    `json:"max_len,omitempty"`
    Pattern  string `json:"pattern,omitempty"` // regex
}

func ValidateArgs(tool *ToolDefinition, args map[string]interface{}) ([]string, error)
```

`MCPGateway.ProcessToolCall` calls `ValidateArgs` before the poisoning detector. Validation failures set `ToolCallStatusDenied` without hitting the more expensive pattern scan.

---

### Phase 4-B: Prompt Injection Detection

**File:** `packages/KNIRVSERVER/backend/internal/services/mcphardening/injection.go`

A pure-Go string-argument scanner for prompt injection patterns, run after schema validation:

```go
type InjectionSignature struct {
    Pattern     string  // compiled regexp
    Weight      float64 // 0..1
    Description string
}

type InjectionDetector struct {
    mu         sync.RWMutex
    signatures []*compiledInjectionSig  // pre-compiled *regexp.Regexp
    threshold  float64
}

type InjectionResult struct {
    Score      float64
    Triggered  []string  // matched signature descriptions
    Blocked    bool
}

func NewInjectionDetector(threshold float64) *InjectionDetector
func (id *InjectionDetector) RegisterSignature(sig InjectionSignature) error
func (id *InjectionDetector) Scan(args map[string]interface{}) InjectionResult
```

Default signatures loaded at startup cover common injection patterns:
- `ignore previous instructions`
- `system prompt override`
- `<|im_start|>system` / `<|im_end|>` token boundaries
- Unicode lookalike character sequences in argument keys
- Recursive self-reference chains (`tool calls tool calls tool`)

`InjectionDetector` is added to `MCPGateway`. `ProcessToolCall` runs `InjectionDetector.Scan` over all string-typed arguments after schema validation.

---

### Phase 4-C: Response-Side Audit

**File:** `packages/KNIRVSERVER/backend/internal/services/mcphardening/response.go`

Extend `ToolCallRecord` with a response field and add a `RecordResponse` method:

```go
type ToolCallResponse struct {
    CallID       string                 `json:"call_id"`
    Output       map[string]interface{} `json:"output"`
    OutputHash   string                 `json:"output_hash"`  // SHA-256 of serialized output
    Duration     time.Duration          `json:"duration"`
    RecordedAt   time.Time              `json:"recorded_at"`
    Anomaly      bool                   `json:"anomaly,omitempty"`
    AnomalyReason string               `json:"anomaly_reason,omitempty"`
}

func (gw *MCPGateway) RecordResponse(callID string, output map[string]interface{}, duration time.Duration) (*ToolCallResponse, error)
```

`RecordResponse` also runs `InjectionDetector.Scan` over the response body — a poisoned tool can inject directives through its output, not just through arguments.

**New HTTP routes:**
- `POST /api/v1/governance/mcp/calls/{id}/response`
- `GET /api/v1/governance/mcp/calls/{id}/response`

---

### Phase 5: Compliance Evidence Layer (new package)

**Package:** `packages/KNIRVSERVER/backend/internal/services/compliance`

This is the missing layer that AGT's `agent-governance-toolkit-sre` and verification CLI provide. It correlates events from all four governance services into framework-mapped compliance reports.

#### 5-A: Event Correlator

**File:** `services/compliance/correlator.go`

```go
type ComplianceEvent struct {
    ID        string            `json:"id"`
    Timestamp time.Time         `json:"timestamp"`
    Source    string            `json:"source"`  // identity | policy | reliability | mcp
    EventType string            `json:"event_type"`
    EntityID  string            `json:"entity_id"`
    NodeID    string            `json:"node_id"`
    AgentID   string            `json:"agent_id,omitempty"`
    Outcome   string            `json:"outcome"` // allow | deny | flag | breach | kill
    Detail    map[string]interface{} `json:"detail"`
    ChainHash string            `json:"chain_hash"`
}

type EventCorrelator struct {
    mu     sync.RWMutex
    events []*ComplianceEvent
    tip    string  // running HMAC-SHA256 chain tip
    key    []byte  // HMAC key loaded from root.key
}

func (ec *EventCorrelator) Record(source, eventType, entityID, nodeID, agentID, outcome string, detail map[string]interface{})
func (ec *EventCorrelator) VerifyChain() error
func (ec *EventCorrelator) Query(filter EventFilter) []*ComplianceEvent
func (ec *EventCorrelator) Export(format string) ([]byte, error)  // json | csv | ndjson
```

Each governance service receives an `*EventCorrelator` at construction and calls `correlator.Record` on every meaningful state change. The HMAC chain makes each event cryptographically dependent on the previous one — retroactive alteration requires the HMAC key.

The HMAC key is derived from KNIRVSERVER's existing `root.key` secret store.

#### 5-B: Compliance Framework Mapper

**File:** `services/compliance/frameworks.go`

Maps KNIRV governance events to specific control requirements for three frameworks:

```go
type ControlMapping struct {
    Framework   string   `json:"framework"`   // GDPR | SOC2 | ISO42001
    ControlID   string   `json:"control_id"`  // e.g. "CC6.1"
    ControlName string   `json:"control_name"`
    EventTypes  []string `json:"event_types"` // which ComplianceEvent.EventType values satisfy this control
    Satisfied   bool     `json:"satisfied"`
    EvidenceIDs []string `json:"evidence_ids"`
}

type ComplianceReport struct {
    GeneratedAt time.Time         `json:"generated_at"`
    WindowStart time.Time         `json:"window_start"`
    WindowEnd   time.Time         `json:"window_end"`
    Framework   string            `json:"framework"`
    Controls    []ControlMapping  `json:"controls"`
    ChainValid  bool              `json:"chain_valid"`
    ReportHash  string            `json:"report_hash"`
}

func GenerateReport(framework string, events []*ComplianceEvent, window time.Duration) (*ComplianceReport, error)
```

**Framework control coverage:**

| Framework | Controls covered |
|-----------|-----------------|
| GDPR | Art. 22 (automated decision-making), Art. 25 (data minimization), Art. 32 (security of processing) |
| SOC 2 | CC6.1 (logical access), CC6.6 (external communication), CC7.2 (monitoring), CC9.2 (vendor risk) |
| ISO 42001 | 6.1 (risk assessment), 8.1 (operational planning), 9.1 (monitoring), 10.1 (continual improvement) |

#### 5-C: Report HTTP Routes

**File:** `packages/KNIRVSERVER/backend/internal/web/compliance_handlers.go`

```
GET  /api/v1/governance/compliance/events           — query event log
GET  /api/v1/governance/compliance/events/verify    — verify HMAC chain
POST /api/v1/governance/compliance/report           — generate report (body: {framework, window_days})
GET  /api/v1/governance/compliance/report/{id}      — retrieve a generated report
```

Reports are stored in memory with a configurable max-count (`governance.compliance.max_reports: 100`).

---

### Phase 6: Governance Verification CLI

**Location:** `packages/KNIRVSHELL/cmd/`

The project CLI is `knirv` — the Cobra binary in `packages/KNIRVSHELL/cmd/root.go`. There is no separate `knirvctl`. The governance commands are added as a new top-level subcommand following the same pattern as `mcp.go` (which pulls subcommand implementations from `packages/KNIRVSHELL/cmd/mcp/`).

**Files:**

| File | Role |
|------|------|
| `packages/KNIRVSHELL/cmd/governance.go` | Declares `governanceCmd`, registers it on `rootCmd` in `init()`, adds subcommand group |
| `packages/KNIRVSHELL/cmd/governance/verify.go` | `VerifyCmd` — verify HMAC chain |
| `packages/KNIRVSHELL/cmd/governance/report.go` | `ReportCmd` — generate compliance report |
| `packages/KNIRVSHELL/cmd/governance/identity.go` | `IdentityCmd` + `IdentityListCmd` — envelope management |
| `packages/KNIRVSHELL/cmd/governance/policy.go` | `PolicyCmd` + `PolicyEvalCmd` — Rego policy evaluation |
| `packages/KNIRVSHELL/cmd/governance/mcp.go` | `MCPCmd` + `MCPAuditCmd` — MCP call audit dump |

**`cmd/governance.go` structure:**

```go
package cmd

import (
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/cmd/governance"
    "github.com/spf13/cobra"
)

var governanceCmd = &cobra.Command{
    Use:   "governance",
    Short: "Inspect and verify KNIRVSERVER governance state",
    Long:  `Query trust envelopes, policy decisions, compliance evidence, and MCP audit records from a running KNIRVSERVER instance.`,
    Run:   func(cmd *cobra.Command, args []string) { cmd.Help() },
}

func init() {
    rootCmd.AddCommand(governanceCmd)
    governanceCmd.AddCommand(governance.VerifyCmd)
    governanceCmd.AddCommand(governance.ReportCmd)
    governanceCmd.AddCommand(governance.IdentityCmd)
    governanceCmd.AddCommand(governance.PolicyCmd)
    governanceCmd.AddCommand(governance.MCPCmd)
}
```

**Conventions (match existing KNIRVSHELL cmd style):**

- Use `github.com/spf13/cobra` for all commands (not `flag`).
- Use the package-level `log` (`logrus.New()` from `root.go`) for output — do not create a second logger.
- Use `viper.GetString("node_url")` to read `--node-url` from the inherited persistent flag rather than defining a new flag.
- HTTP calls go to `viper.GetString("node_url")` (default `http://localhost:8084`). Auth token is read from `~/.knirv.yaml` via viper key `governance.token`.
- Use `github.com/spf13/viper` for any command-local config (e.g. `governance.default_framework`).

**Example command surface:**

```
knirv governance verify                   — verify HMAC chain integrity and print summary
knirv governance report --framework gdpr  — generate compliance report to stdout
knirv governance identity list            — list active trust envelopes
knirv governance policy eval --file input.json --policy mypolicy
knirv governance mcp audit --limit 50
```

---

### Phase 7: Framework Interoperability SDK

The SDK is split across two packages matching the existing project split between KNIRVBASE (importable library) and KNIRVSERVER (the running process):

#### 7-A: Client — `packages/KNIRVSDK/go/governance/`

**Module:** `github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go` (declared in `packages/KNIRVSDK/go/go.mod`)

**New package path within module:** `github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/governance`

This sits alongside the existing SDK subdirectories — `gateway/`, `transaction/`, `transmission/`, `oracled/` — following the same flat layout. External agent frameworks import this package to embed governance checks without making HTTP calls. It communicates with the `GovernanceAgent` server (Phase 7-B) over a Unix domain socket, keeping the enforcement path sub-millisecond.

No new entries in `packages/KNIRVSDK/go/go.mod` are required — the socket client only needs `net`, `encoding/json`, and `sync` from the standard library.

```go
// packages/KNIRVSDK/go/governance/client.go
package governance

// Types are declared locally so KNIRVSDK has no dependency on KNIRVSERVER internals.
type TrustEnvelope struct {
    IdentityID string     `json:"identity_id"`
    NodeID     string     `json:"node_id"`
    AgentID    string     `json:"agent_id,omitempty"`
    TrustLevel string     `json:"trust_level"`
    TrustScore float64    `json:"trust_score"`
    ExpiresAt  time.Time  `json:"expires_at"`
}

type PolicyInput struct {
    NodeID     string                 `json:"node_id"`
    AgentID    string                 `json:"agent_id,omitempty"`
    Action     string                 `json:"action"`
    ActionType string                 `json:"action_type"`
    Metrics    map[string]float64     `json:"metrics,omitempty"`
    Context    map[string]interface{} `json:"context,omitempty"`
}

type PolicyDecision struct {
    Decision     string   `json:"decision"`      // allow | deny
    Reason       string   `json:"reason,omitempty"`
    MatchedRules []string `json:"matched_rules,omitempty"`
}

type ToolCallStatus string

const (
    ToolCallAllowed  ToolCallStatus = "allowed"
    ToolCallDenied   ToolCallStatus = "denied"
    ToolCallFlagged  ToolCallStatus = "flagged"
    ToolCallBlocked  ToolCallStatus = "blocked"
)

// DefaultSocketPath is exported so the KNIRVSERVER GovernanceAgent and this
// client agree on the socket location without requiring config on both sides.
const DefaultSocketPath = "/var/run/knirvserver/governance.sock"

type Client struct {
    socketPath string
    conn       net.Conn
    mu         sync.Mutex
}

func NewClient(socketPath string) (*Client, error)
func (c *Client) CheckIdentity(nodeID, agentID string) (*TrustEnvelope, error)
func (c *Client) EvaluatePolicy(input PolicyInput) (*PolicyDecision, error)
func (c *Client) RecordToolCall(agentID, nodeID, toolName string, args map[string]interface{}) (ToolCallStatus, error)
func (c *Client) Close() error
```

The socket protocol — length-prefixed JSON (4-byte big-endian length + JSON body) — is the full wire contract. Any language can implement a compatible client without a generated stub.

#### 7-B: Server — `packages/KNIRVSERVER/backend/internal/services/governance_sdk/`

Runs as a goroutine inside the main KNIRVSERVER process, started during server initialization alongside the other governance services. Uses the same socket path constant (`governance.DefaultSocketPath` or the override from `governance.sdk.socket_path` in `config/governance.yaml`).

```go
// packages/KNIRVSERVER/backend/internal/services/governance_sdk/server.go
package governance_sdk

type GovernanceAgent struct {
    socketPath string
    identity   *identitybridge.IdentityBridge
    policy     *policyadapter.PolicyAdapter
    mcp        *mcphardening.MCPGateway
    listener   net.Listener
}

func NewGovernanceAgent(
    socketPath string,
    ib  *identitybridge.IdentityBridge,
    pa  *policyadapter.PolicyAdapter,
    mcp *mcphardening.MCPGateway,
) *GovernanceAgent

func (ga *GovernanceAgent) Start() error  // spawns accept loop goroutine
func (ga *GovernanceAgent) Stop()
```

`GovernanceAgent` is initialized in `main.go` immediately after the four governance services are constructed (`s.governanceIdentity`, `s.governancePolicy`, `s.governanceMCP`), and `Start()` is called before `ListenAndServe`.

---

## Sequencing and Dependencies

```
1-A DID resolver          → 1-B zero-trust middleware (needs envelope validation)
1-C revocation list       → 1-B (needs IsRevoked check)
2-A Rego evaluator        → 2-B per-request enforcement (needs evaluator)
2-B enforcement middleware → 2-C guardrail bridge (needs normalized input)
3-A DVE metric binding    → 3-C SLO definitions (SLO records need real metrics)
3-B escalation pipeline   → 3-C (SLO threshold fires escalation)
4-A argument schema       → 4-B injection detection (scan runs after schema pass)
4-B injection detection   → 4-C response audit (response scanner reuses detector)
5-A event correlator      ← all phases above emit events into it
5-B framework mapper      → 5-C report routes (mapper feeds report generator)
6   CLI                   ← all HTTP routes from 1–5
7   SDK                   ← identity + policy + MCP services (reliability via escalation callback)
```

**Recommended execution order** (each item is independently mergeable):

1. `1-A` + `1-C` — DID + revocation (no middleware yet, safe to merge standalone)
2. `2-A` — Rego evaluator (additive to `policyadapter`, no behavior change without loaded policies)
3. `4-A` + `4-B` + `4-C` — MCP hardening depth (existing `mcphardening` package, additive)
4. `3-A` + `3-B` + `3-C` — Reliability depth (requires DVEManager interface impl)
5. `1-B` + `2-B` — Middleware (touches request path; needs 1-A and 2-A complete first)
6. `2-C` — GuardrailEngine bridge (requires 2-B working)
7. `5-A` + `5-B` + `5-C` — Compliance layer (depends on all services emitting events)
8. `6` — CLI (depends on all HTTP routes)
9. `7` — SDK (Unix socket server, independent of HTTP routes)

---

## File Manifest

```
packages/KNIRVORACLE/
└── internal/oracle/
    ├── did/                                           [NEW package — Phase 1-A]
    │   ├── document.go
    │   └── resolver.go
    └── routes/
        ├── routes.go                                  [EXISTS — register new DID routes here]
        ├── did.go                                     [NEW — Phase 1-A HTTP handlers]
        └── ...                                        [existing route files unchanged]

packages/KNIRVSERVER/backend/
├── internal/
│   ├── services/
│   │   ├── identitybridge/
│   │   │   ├── service.go                             [EXISTS — add oracle HTTP call for IdentitySourceDID]
│   │   │   └── revocation.go                          [NEW — Phase 1-C]
│   │   ├── policyadapter/
│   │   │   ├── service.go                             [EXISTS]
│   │   │   └── rego.go                                [NEW — Phase 2-A]
│   │   ├── reliability/
│   │   │   ├── service.go                             [EXISTS]
│   │   │   ├── dve_binding.go                         [NEW — Phase 3-A]
│   │   │   ├── escalation.go                          [NEW — Phase 3-B]
│   │   │   └── slo.go                                 [NEW — Phase 3-C]
│   │   ├── mcphardening/
│   │   │   ├── service.go                             [EXISTS]
│   │   │   ├── schema.go                              [NEW — Phase 4-A]
│   │   │   ├── injection.go                           [NEW — Phase 4-B]
│   │   │   └── response.go                            [NEW — Phase 4-C]
│   │   ├── guardrails/
│   │   │   └── governance_bridge.go                   [NEW — Phase 2-C]
│   │   ├── compliance/                                [NEW package — Phase 5]
│   │   │   ├── correlator.go
│   │   │   └── frameworks.go
│   │   └── governance_sdk/                            [NEW package — Phase 7-B]
│   │       └── server.go
│   └── web/
│       ├── governance_handlers.go                     [EXISTS — extend routes]
│       ├── middleware/
│       │   ├── auth.go                                [EXISTS]
│       │   ├── middleware.go                          [EXISTS]
│       │   ├── pwa.go                                 [EXISTS]
│       │   └── governance.go                          [NEW — Phase 1-B + 2-B]
│       └── compliance_handlers.go                    [NEW — Phase 5-C]
└── config/
    └── governance.yaml                                [NEW — shared config]

packages/KNIRVSHELL/
└── cmd/
    ├── root.go                                        [EXISTS — add governanceCmd in init()]
    ├── governance.go                                  [NEW — Phase 6 top-level command]
    └── governance/
        ├── verify.go                                  [NEW — Phase 6]
        ├── report.go                                  [NEW — Phase 6]
        ├── identity.go                                [NEW — Phase 6]
        ├── policy.go                                  [NEW — Phase 6]
        └── mcp.go                                     [NEW — Phase 6]

packages/KNIRVSDK/go/
├── go.mod                                             [EXISTS — no new deps needed]
├── gateway/                                           [EXISTS]
├── transaction/                                       [EXISTS]
├── transmission/                                      [EXISTS]
├── oracled/                                           [EXISTS]
└── governance/                                        [NEW package — Phase 7-A]
    └── client.go
```

---

## Go Module Change

One new dependency is required for Phase 2-A:

```
github.com/open-policy-agent/opa v0.67.1
```

All other phases use only the Go standard library plus packages already in `backend/go.mod` (`github.com/gorilla/mux`, `github.com/google/uuid`, `crypto/sha256`, `crypto/hmac`, `encoding/json`, `regexp`, `net`, `sync`).

---

## What This Does Not Include

- Any import of `agent-governance-toolkit-*` Python or TypeScript packages
- Any subprocess or shell-out to an external policy evaluator
- Any mock or stub implementations — every interface defined above has a concrete Go implementation
- Replication of KNIRVSERVER's existing guardrail logic — all new code extends or bridges existing behavior rather than replacing it
