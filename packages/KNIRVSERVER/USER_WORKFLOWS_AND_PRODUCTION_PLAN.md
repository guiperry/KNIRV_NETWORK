# KNIRVSERVER — End-User Workflows, Use Cases & Production Readiness Plan

_Generated: 2026-04-18_

---

## Executive Summary

KNIRVSERVER is the central node application of the KNIRV Decentralized Trusted Execution Network (D-TEN). It is simultaneously an **AI orchestration platform**, **decentralized compute marketplace**, **regulatory compliance engine**, and **knowledge mining network**. The platform serves three distinct user archetypes — **Node Operators** (infrastructure), **Organizational Administrators** (policy/ontology owners), and **End Clients** (AI consumers) — across a desktop app (Electron), progressive web app (PWA), and headless API.

The primary end-to-end workflow is:

```
Onboard Organization → Ingest Ontology → Provision DVE Nodes →
Delegate Credentials → Enforce Guardrails → Run Inference/Validation →
Resolve Errors → Mine Knowledge → Anchor Evidence → Distribute Rewards
```

---

## Part 1: End-User Workflows

### 1.1 Primary Workflow — Full Platform Lifecycle

#### Phase 1: Onboard & Configure

**Step 1 — Account Setup**
- User opens the desktop app or navigates to the PWA
- Login page presents: credential login (email/password), token login, or new registration
- Registration collects: username, email, password, name, company, phone, role (admin / validator / observer)
- Email verification link sent; user clicks link to activate account
- JWT token (12-hour expiry) issued on first login; stored in `localStorage`

**Step 2 — Organizational Onboarding**
- Admin selects "Onboarding" from the navigation
- The onboarding wizard (`onboarding-guide.tsx`) walks through six modal steps:
  1. **API Keys** — enter LLM provider keys (Anthropic, Gemini, DeepSeek, Cerebras)
  2. **MCP Servers** — register Model Context Protocol server endpoints (plugin system entry points)
  3. **Policy Certificates** — upload PQC-signed policy certificates for governance anchoring
  4. **Custom Rules** — define custom guardrail rules (allow/deny/warn/log per domain or action type)
  5. **Preferences** — timezone, language, notification preferences
  6. **Cloud Pricing** — configure compute pricing for DVE node rental
- User also submits `POST /api/v1/onboarding/organizations` with their full `ValueSystem` + `Ontology`:
  - **ValueSystem**: guidelines, customs, mission, values, goals, risk appetite, cultural context
  - **Ontology**: trade secrets, business logic, user data, regulations, procedures, policies, FAQs, customer service protocols, industry jargon, stakeholder hierarchy, decision rights
- Platform auto-generates `GuardrailRule` objects for each ontology category
- Onboarding data persisted to the user profile; referenced throughout their session

---

#### Phase 2: Provision Compute (DVE Nodes)

**Step 3 — Provision a DVE Node**
- From the DVE Nodes panel, admin clicks "Create DVE Node"
- Creation form (`dve-creation-form.tsx`) collects: node name, resource limits (CPU, memory), capabilities
- Backend: `DVECreationService` provisions a container (Docker → Podman → native fallback)
  - Assigns SSH endpoint (ports 22000–22999)
  - Assigns Validation endpoint (ports 23000–23999)
  - Assigns Error Resolution endpoint (ports 24000–24999)
  - Registers node on KNIRVCHAIN via `TransactionChainClient`
- Node appears in the DVE Nodes panel with `status: active`

**Step 4 — Discover Peer Nodes (P2P)**
- DVE Manager queries KNIRVCHAIN for registered nodes (chain-native discovery)
- P2P Manager (libp2p + DHT) advertises capabilities on `/knirv/dve/capabilities/1.0.0`
- Admin can toggle DHT from the admin panel
- Connections panel displays live peer list
- KNIRVGATEWAY (TURN/STUN) enables NAT traversal for nodes behind firewalls

---

#### Phase 3: Ingest Data & Build Knowledge

**Step 5 — Ingest Ontology into Knowledge Base**
- Admin navigates to the Knowledge Base / Fabric Management panel
- Uploads documents (PDFs, markdown, text) to a named knowledge base index
- `POST /api/v1/knowledge-base/objects/{id}/index` — backend processes documents through GraphRAG-rs (Rust FFI)
- GraphRAG builds entity-relation graph from documents
- `GET /api/v1/knowledge-base/objects/{id}/index-status` — polls indexing progress
- `POST /api/v1/knowledge-base/objects/{id}/deploy` — deploys index for live inference queries

**Step 6 — Register Plugins / MCP Servers**
- Admin navigates to Plugin Management (Fabric Management tab)
- Uploads plugin binaries or WASM modules via `POST /objects/upload`
- Plugin metadata registered via `POST /api/v1/plugin/objects`
- Runtime started: `POST /objects/runtime/start`
- Plugin appears active; metrics/logs stream in real time from dashboard

**Step 7 — Configure ICME Intent Objectives**
- Admin posts `POST /api/icme/objectives` with intent scope (DVE-level or global), objectives, alignment thresholds
- ICME `AlignmentLoop` begins evaluating agent outputs for drift
- NER + embedding pipeline routes incoming context signals into the temporal hypergraph
- `GET /api/icme/alignment/status` — shows current alignment score

---

#### Phase 4: Delegate Credentials & Run Inference

**Step 8 — Delegate LLM Credentials**
- Admin navigates to "Inference" controls
- `DelegatorService` in InferenceService receives API keys entered during onboarding
- Provider chain configured: primary (e.g., Anthropic) → fallback (Cerebras) → fallback (Gemini)
- Mixture of Agents (MOA) configured: planner / executor / finalizer / verifier roles assigned to providers
- For root nodes: `bin/root.key` decryption at startup extracts API keys automatically; no manual entry required

**Step 9 — Run Inference**
- Client submits task via KNIRVSHELL console or API: `POST /api/inference/*`
- Request routed through `InferenceService`:
  - `ContextStrategist` chunks large documents
  - `ConversationMemory` maintains session context
  - MOA ensemble dispatches sub-tasks to configured providers
  - `TaskOrchestratorInterface` handles multi-step reasoning
- Result returned with confidence score; CognitiveEngine ingests result for learning

---

#### Phase 5: Enforce Guardrails & Validate

**Step 10 — Configure Guardrail Policies**
- Admin opens Policy Editor (`policy-editor.tsx`)
- Creates/edits `Policy` objects: name, rules (resource thresholds, ontology access, action controls), priority, enabled state
- `POST /api/guardrail/policies` — policy stored and registered with ICME intent registry
- `POST /api/guardrail/policies/{id}/commit` — policy hash anchored to ValidationChain, `tx_hash` returned
- WebSocket broadcasts `policy:update` event to all connected dashboards

**Step 11 — Real-Time Guardrail Enforcement**
- Every DVE node submission checked by `GuardrailEngine`:
  - `ValidateResourceUsage(nodeID, metrics)` — checks CPU/memory/network vs. thresholds
  - `ValidateOntologyConstraint(nodeID, domain, concept)` — domain access control
  - KNIRVHASHER gRPC called for cryptographic hash-based security validation
- Violation created on breach: severity, description, affected node recorded
- Guardrail Violations Panel (`guardrail-violations-panel.tsx`) shows live violation list
- Admin reviews and resolves violations: `POST /api/guardrail/violations/{id}/resolve`

**Step 12 — Submit Validation Tasks**
- Admin or automated service submits validation tasks: `POST /api/validation/*`
- `ValidationCore` queues tasks with priority; `ValidationExecutor` processes concurrently
- `LLMValidator` or `ModelValidator` evaluates task
- `ProofGenerator` creates cryptographic proof of result
- Result stored; confidence score + proof returned
- CognitiveEngine `TaskWorkerPool` ingests result for pattern analysis

---

#### Phase 6: Resolve Errors & Mine Knowledge

**Step 13 — Create Error Nodes**
- Failed task or detected anomaly triggers error node creation
- `POST /api/knirvgraph/error-node` — error registered in KNIRVGRAPH (tells other KNIRVGRAPHS via internal DHT)
- KNIRVCHAIN mines `ErrorNode → SkillNode` transition
- `skill.md` file written to KNIRVCHAIN distributed storage
- Error queue visible via `GET /api/knirvgraph/error-queue`

**Step 14 — Error Resolution Session**
- Admin opens Error Resolution Dashboard
- Selects an active error node; opens Error Resolution Modal
- WebSocket-based interactive session connects to DVE node's error resolution endpoint (port 24000–24999)
- HERO Model reads accumulated `skill.md` files and attempts automated resolution
- Human architect can intervene via the web terminal (`xterm.js`)
- Successful resolution creates a Solution Node in the Vault (encrypted `.md`)

**Step 15 — Solution Validation & Anchoring**
- `SolutionNodeValidator` verifies PQC signature on solution node
- `ComplianceEngine` checks solution against regulatory scenarios (`RegulatoryScenario`)
- `AnchoringService` creates PQC-signed evidence pack
- `POST /api/anchoring/` — evidence anchored on-chain; `tx_hash` returned
- Solution visible in Solution Vault panel (`plugin-vault-panel.tsx`)

---

#### Phase 7: Analytics, Telemetry, & Scaling

**Step 16 — Monitor System Health**
- System Health panel polls `GET /api/system-health/*` — aggregated health across all services
- Telemetry card shows eBPF kernel-level metrics (syscall counts, LSM policy events)
- Predictive Analytics panel (`predictive-analytics-panel.tsx`) shows CPU/memory forecasts
- `ProactiveDetector` surfaces anomalies before thresholds are breached
- Module Log Viewer streams real-time logs via SSE from `GET /api/cognitive/telemetry`

**Step 17 — Manage Rollups (Root Nodes)**
- RollupService polls TransactionChain every 30 seconds
- Builds batches of pending transactions: `BuildNextBatch()`
- Submits via Oracle: `SubmitBatch()` → `oracle.SubmitRollup(record)`
- Reconciles finalized/disputed batches: `ReconcileWithOracle()`
- Rollup status visible at `GET /api/rollups/*`

**Step 18 — Scale DVE Network**
- CognitiveEngine `DistributedScaler` evaluates load patterns
- `KubernetesScaler` or native container orchestrator provisions additional nodes
- `DynamicScheduler` assigns tasks based on node reputation and resource availability
- `LoadBalancer` in DVEManager distributes tasks (round-robin / reputation-based / resource-based)

---

### 1.2 Secondary Workflows

#### Workflow A — Financial Compliance (Fintech Plugin)

1. Admin enables Fintech Plugin from Plugin Management
2. Fintech ontologies loaded: KYC, AML, Basel, SEC
3. `FintechValidator` runs regulatory compliance validation on agent outputs
4. `FidelityScorer` scores data quality for compliance reporting
5. `ReplayEngine` replays transactions for audit trails
6. `EvidenceBuilder` assembles compliance evidence packs
7. Financial Compliance Dashboard (`financial-compliance-dashboard.tsx`) displays compliance status
8. NRV token tracing via `NRVTracer` for regulatory tracking

#### Workflow B — SSH Access to DVE Containers

1. Admin selects DVE node from DVE Nodes panel
2. Clicks "SSH Access" — SSH Access Modal opens
3. Connects to DVE container SSH endpoint (port 22000–22999)
4. Web terminal (`xterm.js`) provides in-browser SSH session
5. Alternative: copy SSH credentials and connect via local terminal
6. Admin performs container-level debugging or configuration

#### Workflow C — Mobile Controller Pairing

1. Admin navigates to Controller Integration panel
2. QR code displayed (`qr-code-display.tsx`) with session pairing token
3. User scans QR code with mobile app
4. WebSocket connection established for real-time controller events
5. Mobile device sends control commands forwarded to relevant services

#### Workflow D — DNS Management

1. Admin navigates to DNS Management panel (`dns-management.tsx`)
2. Configures CloudFlare API credentials in settings
3. DNS Service auto-updates A record for `server.knirv.com` every 5 minutes
4. Admin can manually create/update/delete DNS records via UI
5. Dynamic IP changes propagated automatically

#### Workflow E — Badge Creation Lab & DVE Badge Attachment

1. Admin navigates to the Badge Creation Lab panel (`badge-lab-panel.tsx`)
2. Enters a badge purpose/description in the Badge Purpose field
3. Selects **Values** to embed in the badge (Guidelines, Customs, Etiquette, Mission Statement, Stated Values, Goals & Objectives, Insights) — these become the governance signals the badge enforces
4. Selects **Ontology Elements** to embed (Trade Secrets, Business Logic, User Data, Rules, Regulations, Procedures, Policies, FAQs, Customer Service Bullets) — these define which knowledge domains the badge grants and governs access to
5. Configures **Auth Credentials** that will be scoped to the badge: API keys, JWT claims, role bindings, and provider credentials that badge-attached DVE nodes are authorized to use
6. Clicks "Generate Badge" — the badge design is rendered as an SVG with the selected signals visually incorporated (400×400, NFT-ready)
7. Admin reviews the badge design and clicks "Mint to Chain" — `POST /api/knirvshell/chain/badge/create` submits the badge with all metadata (name, badge_type, description, image_data, metadata containing values + ontology + auth credential refs) to KNIRVCHAIN
8. KNIRVCHAIN records the badge as a chain-native capability token; a `badge_id` is returned and displayed
9. Admin navigates to the DVE Nodes panel, selects a DVE node, and opens the DVE management view
10. Admin attaches the minted badge to the DVE: `POST /api/knirvshell/chain/badge/mint` with `badge_id` + `agent_id` (DVE node ID)
11. Once attached, all actors connecting to that DVE (agents, validators, observers, external API clients) inherit the badge's ontology access rules, value alignment constraints, and auth credentials automatically
12. Badge enforcement is handled by `GuardrailEngine` — every request to the DVE is checked against the badge's embedded ontology tags and value signals before execution proceeds
13. Badge status visible on the DVE card; multiple badges can be stacked on a single DVE node (each badge's rules are AND-evaluated)

**Badge Enforcement Contract (once attached to a DVE):**
- **Ontology access**: only knowledge domains listed in the badge's ontology elements are accessible to the DVE's agents
- **Value alignment**: ICME `AlignmentLoop` scores outputs against the badge's value signals; misaligned outputs trigger guardrail violations
- **Auth credentials**: provider API keys and JWT role bindings scoped within the badge are the only credentials available to DVE-resident agents — no direct credential access outside badge scope
- **Auditability**: badge `badge_id` is recorded on every DVE task, validation result, and error node created during the badge's active period; evidence packs reference the badge for compliance anchoring

---

#### Workflow F — Workflow Execution (Multi-Step Automation)

1. Admin defines a workflow: ordered steps with dependency relationships and retry policies
2. Submits: `POST /api/workflow/execute` with step definitions
3. `WorkflowService` resolves dependency graph
4. Steps dispatched to DVE nodes via `DVETaskExecutor`
5. Execution events broadcast over WebSocket
6. Status tracked: `GET /api/workflow/executions/{id}`
7. Final result aggregated across steps

#### Workflow G — NRN Token Payments

1. User initiates NRN transfer: `POST /api/nrn/transfer`
2. `TransactionChainClient` submits on-chain transfer
3. For root nodes: Oracle's `oracleBalanceAdapter` provides live balance before transfer
4. Non-root nodes: balance read from TransactionChain directly
5. Payment confirmation returned with `tx_hash`
6. Fiat alternatives: Stripe checkout session or PayPal order via `/api/v1/payments/`

#### Workflow H — KNIRVSHELL Terminal Commands

1. Admin opens Console Panel from dashboard
2. Types commands into KNIRVSHELL interface
3. `POST /api/v1/shell/execute` — command dispatched to KNIRVSHELL backend
4. Sub-commands available: `wallet`, `validation`, `tee`, `p2p`, `chain`
5. Session management: `create`, `list`, `stop`, `input`
6. Output streamed back to console in real time

#### Workflow I — Active Memory & Reasoning Explorer

1. Developer navigates to Active Memory panel
2. Browses encrypted markdown documents (reasoning traces, solution nodes)
3. `GET /api/active-memory/*` — decrypted documents served from PQC markdown storage
4. `PUT /api/active-memory/*` — write new reasoning traces
5. Arrow Flight stream from Nexus Memory Server provides high-throughput batch access
6. Reasoning Explorer tab shows graph traces (`ReasoningEngine` graph)
7. Solution Vault tab shows validated solution nodes with compliance status

---

## Part 2: Use Cases

### UC-1: Enterprise AI Governance Deployment

**Actor**: Enterprise IT Administrator  
**Goal**: Deploy KNIRVSERVER as an internal AI governance layer that enforces corporate policies on all LLM usage

**Scenario**:
- IT Admin onboards the organization with corporate `ValueSystem` (risk appetite: low, cultural context: regulated industry) and `Ontology` (legal procedures, compliance policies, customer data rules)
- Platform auto-generates 50+ guardrail rules from the ontology
- All internal LLM requests route through KNIRVSERVER's `InferenceService`
- `GuardrailEngine` checks every prompt and response against corporate policies
- Violations logged, reviewed, and escalated via WebSocket alerts
- Policy hashes anchored to ValidationChain for immutable audit trail
- Monthly compliance reports generated from Vault solution nodes and evidence packs

**Value**: Zero-trust AI governance with cryptographic accountability

---

### UC-2: Decentralized Error Mining Network

**Actor**: Node Operator / Network Participant  
**Goal**: Operate a KNIRVGRAPH node that mines AI error knowledge and earns NRN rewards

**Scenario**:
- Operator installs KNIRVSERVER, provisions DVE nodes
- Connects to KNIRVGRAPH via P2P DHT discovery
- Client errors reported to the node via `POST /api/knirvgraph/error-node`
- KNIRVCHAIN mines `ErrorNode → SkillNode` transitions
- HERO Model reads accumulated `skill.md` files; operating agent earns mining rewards proportional to contribution
- Operator monitors earnings via NRN balance at `GET /api/nrn/balance`
- Rollup batches submitted (if root node) for network-wide finalization

**Value**: Transforms AI failure data into monetizable collective knowledge

---

### UC-3: Regulated AI Financial Services

**Actor**: Fintech Platform Developer  
**Goal**: Build an AI-powered financial analysis service with full regulatory compliance

**Scenario**:
- Developer enables Fintech Plugin; loads KYC, AML, Basel, SEC ontologies
- Customer data processed through `FintechValidator` before any LLM analysis
- `SemanticDistance` engine checks regulatory concept proximity
- `ReplayEngine` produces transaction audit trails for examiners
- `EvidenceBuilder` assembles SEC-compliant evidence packs
- `NRVTracer` traces token flows for AML compliance
- All outputs anchored to ValidationChain with PQC signatures
- Financial Compliance Dashboard provides regulator-ready views

**Value**: Production-grade AI for regulated financial environments with audit-ready evidence

---

### UC-4: Multi-Provider AI Orchestration

**Actor**: AI Application Developer  
**Goal**: Build a resilient AI application with automatic provider failover and ensemble reasoning

**Scenario**:
- Developer configures InferenceService with provider chain (Anthropic → Cerebras → Gemini)
- MOA (Mixture of Agents) configured: Anthropic as planner, Cerebras as executor, Gemini as verifier
- Application submits complex multi-step tasks via `TaskOrchestratorInterface`
- `ContextStrategist` automatically chunks large documents for token limits
- `ConversationMemory` maintains context across session turns
- If Anthropic is unavailable, automatic fallback to Cerebras
- CognitiveEngine learns which provider performs best for which task types
- `AdaptationEngine` automatically adjusts provider routing based on learned patterns

**Value**: Resilient, intelligent multi-provider AI orchestration without vendor lock-in

---

### UC-5: Sovereign Knowledge Management

**Actor**: Research Organization  
**Goal**: Build a private semantic knowledge base with graph-augmented retrieval

**Scenario**:
- Researcher uploads proprietary documents to Knowledge Base
- GraphRAG-rs (Rust) builds entity-relation graph from document corpus
- Researcher queries: `POST /api/v1/knowledge-base/objects/{id}/query` with semantic query
- FAISS + BM25 hybrid search returns ranked results with graph context
- ICME hypergraph tracks query patterns over time
- Knowledge base deployed: `POST /api/v1/knowledge-base/objects/{id}/deploy`
- All data encrypted at rest via PQC markdown storage
- Ontology guardrails prevent unauthorized concept access

**Value**: Private, sovereign knowledge management with state-of-the-art retrieval

---

## Part 3: User Stories

### Organizational Administrator

- **US-01**: As an administrator, I want to define my organization's ontology so the platform enforces our business rules on all AI operations.
- **US-02**: As an administrator, I want to see all guardrail violations in real time so I can respond to policy breaches immediately.
- **US-03**: As an administrator, I want to commit policies to the blockchain so I have an immutable audit trail of all governance decisions.
- **US-04**: As an administrator, I want to configure LLM provider failover chains so AI services remain available if a provider goes down.
- **US-05**: As an administrator, I want to configure per-DVE resource thresholds so no single workload can consume excessive compute.
- **US-06**: As an administrator, I want to onboard my organization's value system so guardrail rules are automatically generated from our culture and risk appetite.

### Node Operator

- **US-07**: As a node operator, I want to provision DVE containers with one click so I can quickly expand compute capacity.
- **US-08**: As a node operator, I want to SSH directly into DVE containers from the browser so I can debug without leaving the dashboard.
- **US-09**: As a node operator, I want to monitor eBPF kernel-level telemetry so I can detect security anomalies at the syscall level.
- **US-10**: As a node operator, I want predictive analytics warnings before my nodes breach resource thresholds so I can scale proactively.
- **US-11**: As a node operator, I want automatic DNS updates so my node's IP changes are propagated without manual intervention.
- **US-12**: As a root node operator, I want the oracle to load my credentials automatically from an encrypted key file so secrets are never stored in plaintext.

### Developer / Integrator

- **US-13**: As a developer, I want to upload WASM plugins so I can extend the platform without modifying core code.
- **US-14**: As a developer, I want to define multi-step workflows with dependency resolution so I can automate complex AI pipelines.
- **US-15**: As a developer, I want to build GraphRAG knowledge bases so users can query documents with semantic graph retrieval.
- **US-16**: As a developer, I want to access reasoning traces in the Active Memory layer so I can debug AI decision chains.
- **US-17**: As a developer, I want to use KNIRVSHELL's programmatic API so I can automate platform operations from CI/CD pipelines.
- **US-18**: As a developer, I want webhook / WebSocket events for all platform state changes so I can build reactive integrations.

### End User (AI Consumer)

- **US-19**: As an end user, I want to submit inference tasks that automatically use my organization's knowledge base so answers are grounded in our proprietary data.
- **US-20**: As an end user, I want to see confidence scores on AI responses so I can calibrate trust in the output.
- **US-21**: As an end user, I want to submit error reports that feed into the collective KNIRV knowledge network so my failure data improves future AI performance.
- **US-22**: As an end user, I want to make NRN token payments for compute so I can pay per use without a monthly subscription.

### Badge Lab

- **US-26**: As an administrator, I want to design a badge that encodes my organization's ontology tags and value signals so that any DVE attached to that badge automatically enforces our governance rules without per-request configuration.
- **US-27**: As an administrator, I want to scope auth credentials (API keys, JWT role bindings) inside a badge so that DVE-resident agents can only use credentials I have explicitly authorized within the badge.
- **US-28**: As an administrator, I want to mint a badge to KNIRVCHAIN so the badge's governance contract is immutable and auditable.
- **US-29**: As an administrator, I want to attach a minted badge to one or more DVE nodes so the badge's ontology access rules, value alignment thresholds, and auth credentials are automatically inherited by all actors connecting to those DVEs.
- **US-30**: As a compliance officer, I want every DVE task result to reference the badge ID active at execution time so compliance evidence packs can prove which governance contract was in force.

### Compliance Officer

- **US-23**: As a compliance officer, I want all AI outputs validated against regulatory ontologies (KYC, AML, SEC) before they leave the platform.
- **US-24**: As a compliance officer, I want cryptographically-signed evidence packs anchored on-chain so I can prove compliance to regulators.
- **US-25**: As a compliance officer, I want transaction replay for audit purposes so I can reconstruct any historical AI decision.

---

## Part 4: Production Readiness — What Needs to Be Done

### 4.1 Critical Blockers (Must-Fix Before Production)

#### Security

- [ ] **Remove testnet hardcoded tokens** — `renderer.js` and `login/page.tsx` contain `testnet-admin-123`, `testnet-validator-456`, `testnet-observer-789`. These must be removed or gated behind a build flag that is disabled in production builds.
- [ ] **Enforce `AuthRequired = true` in production config** — `config/production.yaml` must set `security.auth_required: true`. Confirm `config.Security.AuthRequired` cannot be overridden at runtime without admin auth.
- [ ] **TLS everywhere** — Confirm all HTTP routes are served behind TLS in production. The root key embeds TLS cert/key; verify these are loaded and enforced on startup when the key is present.
- [ ] **Rate limiting on auth endpoints** — `POST /api/auth/login` and `POST /api/auth/register` must have rate limiting to prevent brute force. Confirm `client IP` audit trail is wired to an actual limiter, not just logged.
- [ ] **JWT secret rotation** — JWT secret loaded from `root.key` must support rotation without downtime. Define rotation procedure.
- [ ] **PQC key rotation procedure** — `PQCManager` supports key rotation; document and test the rotation flow including re-encryption of existing markdown storage.

#### Stability

- [ ] **DVE Manager demo-seed removal** — `ChainNativeNodeDiscovery` replaces demo seeding. Confirm demo nodes are fully removed from `dve_manager.go` initialization path in production config.
- [ ] **Oracle binary deployment** — `knirvoracle` is an external binary managed via Unix socket. Define packaging, versioning, and restart policy (systemd unit, health check probe).
- [ ] **KNIRVCHAIN / KNIRVGRAPH / KNIRVGATEWAY binary co-location** — These three external binaries are managed by KNIRVSERVER. Production packaging must bundle all binaries with correct versions; startup failure of any binary must be handled gracefully.
- [ ] **Container runtime selection in production** — `TEESecurityService` selects Docker → Podman → native. Confirm production environment has a supported runtime and the selection logic matches the deployment target.
- [ ] **Provisioning timeout tuning** — Container provisioning timeout is 5 minutes; cleanup is 10 minutes. Validate these are appropriate for production container images.

#### Data Integrity

- [ ] **BuntDB persistence and backup** — BuntDB is an embedded key-value store with no built-in replication. Define backup strategy (snapshot schedule, off-node copy) for production data.
- [ ] **Rollup persistence** — Rollup state is persisted to `<app_data>/rollups/transaction_rollups.json`. This must be on durable storage; confirm it is excluded from any ephemeral volume.
- [ ] **Markdown PQC storage durability** — PQC-encrypted `.md` files are the primary knowledge store. Confirm storage path is on durable, backed-up disk.

---

### 4.2 High-Priority Items (Pre-Launch Polish)

#### Authentication & Authorization

- [ ] **Email verification flow** — `POST /api/auth/verify-email` exists; confirm the email sender is configured in production (SMTP credentials in config) and the verification UX is tested end-to-end.
- [ ] **Password reset flow** — Verify a `POST /api/auth/forgot-password` / `POST /api/auth/reset-password` flow exists or add it; password reset is a basic auth requirement.
- [ ] **Role-based access enforcement** — `role-guard.tsx` exists on the frontend; confirm backend routes enforce role checks (not just `RequireAuth` but also role inspection from JWT claims) for admin-only operations.
- [ ] **Token revocation on logout** — `POST /api/auth/revoke` exists; confirm the frontend calls this on logout and the in-memory revocation list is populated.

#### Frontend

- [ ] **Demo mode gating** — `demo-mode-toggle.tsx` disables real API calls. Confirm this toggle is not accessible to non-admin roles in production.
- [ ] **Error boundary coverage** — `error.tsx` exists at the app level; ensure all major panels have React error boundaries to prevent full-page crashes from a single panel failure.
- [ ] **WebSocket reconnect logic** — `use-knirv-socket.ts` must implement exponential backoff reconnect for production reliability.
- [ ] **Loading and empty states** — All data panels must have proper loading spinners and empty-state messaging to prevent confusing blank UIs during slow backend responses.

#### Operations

- [ ] **Structured logging** — Confirm all services use structured (JSON) logging with consistent fields (service name, correlation ID, severity) for production log aggregation.
- [ ] **Health check completeness** — `GET /health` and `GET /api/health` should return structured JSON with per-service status. Confirm `SystemHealthService` covers all critical services.
- [ ] **Graceful shutdown** — Server must handle SIGTERM by draining in-flight requests, stopping external binaries cleanly, and flushing BuntDB before exit.
- [ ] **Metrics endpoint** — Add or verify a Prometheus `/metrics` endpoint for the network monitor (Grafana dashboards in `devtools/network-monitor/`).
- [ ] **Configurable log levels** — Production should default to `INFO`; `DEBUG` mode must be disabled.

#### Payments

- [ ] **Stripe webhook signature verification** — Stripe callbacks must validate `Stripe-Signature` header to prevent spoofed payment events.
- [ ] **PayPal webhook verification** — Same requirement for PayPal IPN/webhook callbacks.
- [ ] **NRN transaction idempotency** — Duplicate NRN transfer submissions must be detected and rejected to prevent double-spend.

---

### 4.3 Medium-Priority Items (First 30 Days Post-Launch)

#### Feature Completeness

- [ ] **Badge DVE Attachment** — `POST /api/knirvshell/chain/badge/mint` currently mints a badge to an agent ID but does not wire the badge's ontology tags, value signals, or auth credentials into `GuardrailEngine` enforcement for the target DVE. The full enforcement pipeline (badge metadata → GuardrailEngine rule injection → ICME alignment scoring → audit trail stamping with `badge_id`) must be completed.
- [ ] **Badge Auth Credential Scoping** — Badge creation UI collects values and ontology elements but does not yet capture auth credentials (API keys, JWT role bindings, provider credential refs). The `BadgeCreateRequest.Metadata` field must be extended and the backend must validate that DVE agents cannot access credentials outside their attached badge's scope.
- [ ] **Badge Stacking / AND-evaluation** — When multiple badges are attached to a DVE, their ontology and value constraints must be AND-evaluated. Define and implement the merge semantics in `GuardrailEngine`.
- [ ] **Badge Status on DVE Card** — The DVE Nodes panel (`dve-nodes-panel.tsx`) must display attached badges and their enforcement status on the DVE card.
- [ ] **Badge Generation (AI-driven)** — Current badge generation is a client-side SVG stub. Wire to an actual AI image generation call or structured SVG templating that visually encodes the selected values and ontology elements.

- [ ] **DVE Rental Service** — `dverental` is marked deprecated. Confirm whether rental-based compute allocation is a planned user workflow; if so, replace or resurrect the service.
- [ ] **KNIRVARENA integration** — The HERO Model and Dataset Forge (KNIRVARENA) are separate packages. Define the integration surface: how does KNIRVSERVER expose `skill.md` data to KNIRVARENA, and how do reward distributions flow back?
- [ ] **CDE (Containerized Development Environment)** — CDE service exists with a resource pool; confirm whether this is exposed to end users and document the UX for accessing CDEs.
- [ ] **Agent Service (oh-my-pi) production readiness** — Define what `oh-my-pi` agent runtime does in production, its resource limits, and security isolation within DVE containers.

#### Observability

- [ ] **Distributed tracing** — Add OpenTelemetry trace propagation (or use KNIRVCHAIN's `internal/tracing/`) across all service calls for production debugging.
- [ ] **Alertmanager integration** — Wire `devtools/network-monitor/config/alertmanager.yml` rules to PagerDuty or Slack for production on-call.
- [ ] **eBPF telemetry dashboards** — Create Grafana panels for eBPF syscall metrics, LSM policy events, and XDP packet filter stats.

#### Testing

- [ ] **E2E test suite for primary workflow** — Write an integration test covering the full primary workflow (onboard → provision DVE → configure guardrail → run inference → create error node → resolve → anchor). The `integration-tests/` directory covers service startup but not the full user workflow.
- [ ] **Load testing** — Benchmark `InferenceService` under concurrent load; measure DVE provisioning latency under burst; test WebSocket server with 100+ simultaneous clients.
- [ ] **Guardrail policy fuzz testing** — Fuzz `PolicyEngine.EvaluateAction()` with malformed inputs to ensure no bypass vectors.
- [ ] **Root key loading test** — Add a test that loads a real (test) encrypted root key and verifies all secrets are correctly extracted.

---

### 4.4 Deployment Checklist

```
Infrastructure
  [ ] Systemd unit file for knirv-server (docs/SYSTEMD_SERVICE.md — verify up to date)
  [ ] Systemd unit files for knirvoracle, knirvchain, knirvgraph, knirvgateway binaries
  [ ] AppArmor / SELinux profiles for all binaries
  [ ] Durable block storage mounted at <app_data>
  [ ] TLS certificates provisioned (Let's Encrypt or corporate CA)
  [ ] CloudFlare API credentials for DNS service
  [ ] Network firewall rules for ports: 8084, 8082, 7080, 22000-24999 (DVE)

Secrets
  [ ] root.key generated and encrypted with strong password
  [ ] ORACLE_KEY_PASSWORD set in environment (never in code or config file)
  [ ] Stripe/PayPal keys in root.key or environment variables
  [ ] SMTP credentials for email verification

Build
  [ ] Frontend built: cd frontend && npm run build && npm run export
  [ ] All embedded binaries compiled and placed in bin/
  [ ] GraphRAG-rs Rust library compiled and embedded via CGO
  [ ] Production config validated: config/production.yaml

Operations
  [ ] Log aggregation configured (e.g., Loki, CloudWatch, Datadog)
  [ ] Prometheus scrape config for /metrics
  [ ] Grafana dashboards imported from devtools/network-monitor/config/grafana/
  [ ] Backup job for BuntDB and PQC markdown storage (daily minimum)
  [ ] Alertmanager rules active with on-call contact
  [ ] Runbook written for: service restart, root key rotation, database restore, DVE provisioning failure
```

---

### 4.5 Priority Matrix Summary

| Area | Blocker | High | Medium |
|---|---|---|---|
| Security | Remove testnet tokens, enforce auth, TLS, rate limiting, JWT rotation | Role enforcement, token revocation, email verification | Fuzz testing, load testing |
| Data | BuntDB backup, rollup durability, PQC storage | — | Distributed tracing |
| Stability | External binary packaging, container runtime, graceful shutdown | Health check completeness, structured logging | eBPF dashboards |
| Features | Oracle binary deployment | Demo mode gating, WebSocket reconnect | DVE Rental, KNIRVARENA integration, CDE UX |
| Payments | — | Stripe/PayPal webhook verification, NRN idempotency | — |
| Testing | — | E2E primary workflow, load testing | Full fuzz suite |

---

### 4.5 Final Order TODOs

1. Ensure Network Website and Coporate website are updated to reflect the information as detailed in this document.
2.

---

## Part 5: DVE Installation & Browser Routing

This section documents the distributed DVE Installation workflow that distributes installer functions across KNIRVGATEWAY, KNIRVORACLE, and KNIRVSERVER, plus the DVE browser routing via `knirv://` URI scheme.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        INSTALLER FLOW                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. KNIRVGATEWAY                                                    │
│     ├─ Registry/STUN Discovery (tunnel/registry.go)                │
│     └─ Port Discovery (tunnel/stun.go)                             │
│            │                                                         │
│            ▼                                                         │
│  2. KNIRVORACLE (core)                                              │
│     ├─ Wallet Generation (crypto/ecdsa.go)                            │
│     └─ DVE URI Generation (adapt generateTransferID pattern)         │
│            │                                                         │
│            ▼                                                         │
│  3. KNIRVSERVER                                                    │
│     ├─ Service Setup (reuse knirvchain.Manager pattern)              │
│     ├─ InstallComplete Tracking (extend DVENode)                      │
│     └─ Validation Chain Submission (validationchain.Client)         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                       DVE ROUTING FLOW                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Browser: knirv://abc123/my-dve-id                                    │
│         │                                                             │
│         ▼                                                             │
│  KNIRVSERVER (Router)                                                │
│     ├─ Parse DVE ID from URI                                         │
│     ├─ Check BuntDB cache                                            │
│     │   └─ If miss → Query Validation Chain                             │
│     ├─ Validate (no auth for public pages)                             │
│     └─ Serve Go HTML Template (Public DVE Page)                       │
│                                                                      │
│  KNIRVARENA (interactive)                                            │
│     └─ WebSocket proxy for interactive DVE features                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

### DVE Installation Workflow

#### Phase 1: Registry & STUN Discovery → KNIRVGATEWAY

**Step 1 — Node Registration**
- Installer client calls `POST /api/installer/register-bootnode`
- KNIRVGATEWAY `RegistryManager` registers the bootnode in BuntDB
- Returns: nodeID, chainID, registration confirmation

**Step 2 — STUN/Port Discovery**
- STUN client queries KNIRVGATEWAY's STUN server
- STUN server returns: public IP, exposed ports (UDP/TCP)
- NAT traversal information gathered for future P2P connections

#### Phase 2: Chain URI Generation → KNIRVORACLE

**Step 3 — Wallet Generation**
- KNIRVGATEWAY proxy calls KNIRVORACLE (Unix socket)
- KNIRVORACLE uses existing `crypto.GenerateKeyPair()` (from `crypto/ecdsa.go`)
- Returns: wallet address (Ethereum-style), public key

**Step 4 — DVE URI Generation**
- KNIRVGATEWAY proxy calls KNIRVORACLE DVE URI endpoint
- KNIRVORACLE adapts `generateTransferID` pattern to generate DVE URI
- Format: `knirv://{uuid}/{dve_id}`
- Returns: complete DVE URI string

#### Phase 3: Service Setup → KNIRVSERVER

**Step 5 — System Service Setup**
- KNIRVSERVER `InstallerService` (new) provisions required services
- Reuses `knirvchain.Manager` pattern for lifecycle management
- Services started: Docker/Podman containers, validation endpoints

**Step 6 — InstallComplete Tracking**
- KNIRVSERVER updates `DVENode` in BuntDB with new fields:
  - `InstallComplete` (bool)
  - `InstallCompleteAt` (timestamp)
  - `InstallPhase` (string: "registry" → "stun" → "wallet" → "dve_uri" → "service" → "complete")
  - `DVEURI` (string)
  - `WalletAddress` (string)

**Step 7 — Validation Chain Submission**
- KNIRVSERVER submits DVE URI record to ValidationChain
- `validationchain.Client` extended with new methods:
  - `SubmitDVEURI(req DVEURIRecord) (txHash, error)`
  - `GetDVEURI(dveID) (*DVEURIRecord, error)`
  - `GetDVEURIByWallet(walletAddr) ([]*DVEURIRecord, error)`

#### Phase 4: DVE Browser Routing

**Step 8 — DVE URI Registry**
- New `DVEURIRegistry` service in KNIRVSERVER
- BuntDB-backed with ring cache for hot URIs
- Fallback to ValidationChain on cache miss

**Step 9 — DVE Proxy Handlers**
- Route pattern: `/dve/{dve_id}/...`
- Uses existing `ViewportProxyImpl` pattern
- Proxy validation requests to DVE containers
- Public pages require no authentication

**Step 10 — Public DVE HTML Templates**
- New Go template directory: `backend/templates/dve/`
- `public_page.gohtml` — main public validation page
- `validation_records.gohtml` — search and display validation records
- `metrics_panel.gohtml` — DVE metrics display
- `search_form.gohtml` — validation record search form

**Step 11 — Interactive DVE (WebSocket)**
- New `DVEClient.ts` in KNIRVARENA
- Extends existing `ArenaClient` pattern
- WebSocket connection for:
  - Real-time validation record updates
  - Interactive validation requests
  - Live metrics streaming

---

### Integration with Existing Code

| Component | Existing File | Integration |
|-----------|-------------|------------|
| Registry | `packages/KNIRVGATEWAY/internal/tunnel/registry.go` | Add `RegisterBootnode`, `GetBootnodes` methods |
| STUN | `packages/KNIRVGATEWAY/internal/tunnel/stun.go` | Reuse `STUNServer` for client creation |
| Wallet | `packages/KNIRVORACLE/internal/oracle/crypto/ecdsa.go` | Reuse `GenerateKeyPair()` |
| DVE URI | `packages/KNIRVORACLE/internal/oracle/crosschain/router.go` | Adapt `generateTransferID` pattern |
| DVE Object | `packages/KNIRVSERVER/backend/internal/objects/dve.go` | Add InstallComplete fields |
| Installer | `packages/KNIRVSERVER/backend/pkg/knirvchain/manager.go` | Reuse Manager pattern |
| Validation Chain | `packages/KNIRVSERVER/backend/internal/services/blockchain/validationchain/client.go` | Add DVE URI methods |
| Viewport Proxy | `packages/KNIRVSERVER/backend/internal/runtime/viewport_proxy.go` | Reuse ReverseProxy pattern |
| Arena Client | `packages/KNIRVARENA/packages/ts_client_2/src/networking/ArenaClient.ts` | Extend for DVE |

---

### Files to Modify/Create

**Existing Files to Modify:**
- `packages/KNIRVGATEWAY/internal/tunnel/registry.go` — Add installer methods
- `packages/KNIRVGATEWAY/internal/server/server.go` — Register routes
- `packages/KNIRVSERVER/backend/internal/objects/dve.go` — Add InstallComplete fields
- `packages/KNIRVSERVER/backend/internal/services/blockchain/validationchain/client.go` — Add DVE URI methods
- `packages/KNIRVSERVER/backend/internal/web/api_router.go` — Register DVE proxy routes
- `packages/KNIRVARENA/packages/ts_client_2/src/App.tsx` — Add DVE route

**New Files to Create:**
- `packages/KNIRVGATEWAY/internal/installer/stun_client.go` — STUN client
- `packages/KNIRVGATEWAY/internal/installer/dve_uri.go` — Proxy to KNIRVORACLE
- `packages/KNIRVORACLE/internal/oracle/routes/wallet.go` — Wallet generation route
- `packages/KNIRVORACLE/internal/oracle/routes/dve_uri.go` — DVE URI generation route
- `packages/KNIRVSERVER/backend/internal/services/installer/installer.go` — Installer service
- `packages/KNIRVSERVER/backend/internal/services/dve_uri_registry.go` — DVE URI registry
- `packages/KNIRVSERVER/backend/internal/web/dve_proxy_handlers.go` — DVE proxy handlers
- `packages/KNIRVSERVER/backend/templates/dve/*.gohtml` — Public DVE templates
- `packages/KNIRVARENA/packages/ts_client_2/src/networking/DVEClient.ts` — DVE WebSocket client

---

### Implementation Order

```
Phase 1: KNIRVGATEWAY
├── 1.1 extend registry.go (installer methods)
└── 1.2 add server.go routes + new installer files

Phase 2: KNIRVORACLE  
├── 2.1 new wallet.go (reuse crypto)
└── 2.2 new dve_uri.go (adapt pattern)

Phase 3: KNIRVSERVER
├── 3.1 extend objects/dve.go (InstallComplete)
├── 3.2 new installer/installer.go (reuse Manager pattern)
└── 3.3 extend validationchain/client.go

Phase 4: DVE Routing
├── 4.1 new dve_uri_registry.go
├── 4.2 new dve_proxy_handlers.go
├── 4.3 new templates/dve/
├── 4.4 new DVEClient.ts
└── 4.5 register routes
```

_This document reflects the codebase state as of 2026-04-18. Revisit after each significant milestone._
