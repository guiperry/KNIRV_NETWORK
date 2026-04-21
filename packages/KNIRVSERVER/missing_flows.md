# Missing Flows — Codebase vs Specification Gap Analysis

_Generated: 2026-04-20_
_Document: Cross-reference of `USER_WORKFLOWS_AND_PRODUCTION_PLAN.md` vs current codebase_

---

## Executive Summary

This document identifies all UI components and backend wiring that are specified in `USER_WORKFLOWS_AND_PRODUCTION_PLAN.md` but are either missing entirely or incomplete in the current codebase. The gaps are organized by severity and workflow phase.

---

## 1. Critical Blockers (Must-Fix Before Production)

### 1.1 Security — Testnet Hardcoded Tokens

**Files containing hardcoded test tokens:**
- `frontend/src/app/login/page.tsx` (lines 25-27)
- `frontend/src/lib/auth-context.tsx` (lines 100-116, 240-256)
- `desktop/renderer.js` (lines 385-387)
- `frontend/renderer.js` (lines 378-380)
- `backend/internal/web/middleware/auth.go` (lines 355-367)

**Missing:**
- Production build flag to gate these tokens
- Environment-based token configuration
- Removal from production builds

### 1.2 Security — AuthRequired Enforcement

**Specified:** `config/production.yaml` must set `security.auth_required: true`

**Missing:**
- Verify `config.Security.AuthRequired` cannot be overridden at runtime without admin auth
- Confirm all HTTP routes check this flag before allowing unauthenticated access

### 1.3 Authentication — Email Verification

**Specified:** `POST /api/auth/verify-email` flow

**Status:** Route exists but verify:
- SMTP credentials configured in production config
- Frontend verification UX wired end-to-end
- Email sender properly configured

### 1.4 Authentication — Password Reset Flow

**Specified:** `POST /api/auth/forgot-password` / `POST /api/auth/reset-password`

**Missing:** Verify these routes exist or implement them

### 1.5 Authentication — Token Revocation on Logout

**Specified:** `POST /api/auth/revoke`

**Missing:**
- Verify frontend calls this on logout
- Confirm in-memory revocation list is populated

### 1.6 Stripe/PayPal Webhook Verification

**Specified:**
- Verify `Stripe-Signature` header on Stripe callbacks
- Verify PayPal IPN/webhook callbacks

**Missing:** Confirm webhook signature verification is implemented

### 1.7 NRN Transaction Idempotency

**Specified:** Duplicate NRN transfer submissions must be detected

**Missing:** Verify idempotency logic exists in `NRNPaymentHandlers`

---

## 2. High-Priority Items (Pre-Launch Polish)

### 2.1 Frontend — Demo Mode Gating

**Specified:** `demo-mode-toggle.tsx` disables real API calls

**Missing:**
- Confirm toggle is not accessible to non-admin roles in production
- Verify all demo mode code checks role before allowing access

### 2.2 Frontend — Error Boundary Coverage

**Specified:** All major panels have React error boundaries

**Missing:**
- Verify `error.tsx` exists at app level
- Confirm each panel has error boundary to prevent full-page crashes

### 2.3 Frontend — WebSocket Reconnect Logic

**Specified:** `use-knirv-socket.ts` implements exponential backoff reconnect

**Missing:** Verify exponential backoff reconnect logic is implemented

### 2.4 Frontend — Loading and Empty States

**Specified:** All data panels have loading spinners and empty-state messaging

**Missing:** Audit all panels for proper loading/empty states

### 2.5 Role-Based Access Enforcement

**Specified:** Backend routes enforce role checks (not just RequireAuth)

**Missing:**
- Verify backend inspects role from JWT claims for admin-only operations
- Confirm `role-guard.tsx` is properly wired with backend enforcement

### 2.6 Operations — Structured Logging

**Specified:** All services use structured (JSON) logging

**Missing:** Verify consistent logging fields (service name, correlation ID, severity)

### 2.7 Operations — Health Check Completeness

**Specified:** `GET /health` and `GET /api/health` return structured JSON

**Missing:** Confirm `SystemHealthService` covers all critical services

### 2.8 Operations — Metrics Endpoint

**Specified:** Prometheus `/metrics` endpoint

**Missing:** Verify or add Prometheus endpoint for network monitor

---

## 3. Workflow Gaps (By Phase)

### Phase 1: Onboard & Configure

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Account Setup (registration) | Full flow | ⚠️ Partial | Email verification UX wired? Password reset flow? |
| Organizational Onboarding | Six modal steps | ✅ Done | Backend wiring for ValueSystem + Ontology persistence |
| API Keys Modal | Step 1 | ⚠️ Partial | Backend save to BuntDB? |
| MCP Servers Modal | Step 2 | ⚠️ Partial | Endpoint registration backend? |
| Policy Certs Modal | Step 3 | ⚠️ Partial | PQC-signed cert backend? |
| Custom Rules Modal | Step 4 | ❌ Missing | GuardrailRule generation from ontology |
| Preferences Modal | Step 5 | ⚠️ Partial | Backend persistence? |
| Cloud Pricing Modal | Step 6 | ⚠️ Partial | Backend pricing config? |

**Backend Wiring Missing:**
- `POST /api/v1/onboarding/organizations` handler to save ValueSystem + Ontology
- GuardrailRule auto-generation from ontology categories
- ValueSystem persistence to user profile

### Phase 2: Provision Compute (DVE Nodes)

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| DVE Creation Form | `dve-creation-form.tsx` | ✅ Done | — |
| DVECreationService | Backend provision | ✅ Done | — |
| P2P Discovery | Chain + DHT | ✅ Done | DHT toggle wiring |
| SSH Access Modal | `ssh-access-modal.tsx` | ✅ Done | Port 22000-22999 backend |
| Validation Access | Port 23000-23999 | ✅ Done | — |
| Error Resolution | Port 24000-24999 | ✅ Done | — |

**Backend Wiring Missing:**
- KNIRVGATEWAY (TURN/STUN) NAT traversal integration

### Phase 3: Ingest Data & Build Knowledge

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Knowledge Base Indexing | `POST /api/v1/knowledge-base/objects/{id}/index` | ✅ Done | GraphRAG FFI wired? |
| Index Status | `GET /api/v1/knowledge-base/objects/{id}/index-status` | ⚠️ Check | Polling endpoint? |
| Deploy Index | `POST /api/v1/knowledge-base/objects/{id}/deploy` | ⚠️ Check | Live inference deployment? |
| Plugin Upload | `POST /objects/upload` | ✅ Done | — |
| Plugin Runtime | `POST /objects/runtime/start` | ⚠️ Check | WASM runtime integration |
| ICME Objectives | `POST /api/icme/objectives` | ✅ Done | Alignment loop wired? |

### Phase 4: Delegate Credentials & Run Inference

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| DelegatorService | Provider chain config | ✅ Done | MOA ensemble config UI? |
| Inference Endpoints | `POST /api/inference/*` | ✅ Done | — |
| ContextStrategist | Document chunking | ✅ Done | — |
| ConversationMemory | Session context | ✅ Done | — |

### Phase 5: Enforce Guardrails & Validate

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Policy Editor | `policy-editor.tsx` | ✅ Done | — |
| Policy Commit | `POST /api/guardrail/policies/{id}/commit` | ⚠️ Check | Blockchain anchoring? |
| GuardrailEngine | Real-time enforcement | ✅ Done | KNIRVHASHER gRPC? |
| Violations Panel | `guardrail-violations-panel.tsx` | ✅ Done | Real-time updates? |
| Validation Service | `POST /api/validation/*` | ✅ Done | — |
| ProofGenerator | Cryptographic proof | ⚠️ Check | Implementation? |

### Phase 6: Resolve Errors & Mine Knowledge

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Error Node Creation | `POST /api/knirvgraph/error-node` | ✅ Done | DHT propagation? |
| Error Queue | `GET /api/knirvgraph/error-queue` | ⚠️ Check | UI endpoint? |
| Error Resolution Modal | WebSocket session | ⚠️ Partial | xterm.js wired? |
| Solution Node | Vault encrypted | ⚠️ Check | Encryption backend? |
| Anchoring Service | `POST /api/anchoring/` | ⚠️ Check | PQC evidence pack? |

### Phase 7: Analytics & Scaling

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| System Health Panel | Real-time polling | ✅ Done | — |
| Predictive Analytics | `predictive-analytics-panel.tsx` | ✅ Done | CPU/memory forecasts? |
| ProactiveDetector | Anomaly surfacing | ⚠️ Check | Pre-threshold alerting? |
| Module Log Viewer | SSE streaming | ⚠️ Check | Real-time SSE? |
| Rollup Service | Batch submission | ⚠️ Check | Oracle integration? |
| DistributedScaler | Auto-scaling | ⚠️ Check | K8s/native orchestration? |

---

## 4. Secondary Workflows (Incomplete)

### Workflow A — Financial Compliance

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Fintech Plugin | Enable/disable | ✅ Done | — |
| Fintech Validator | Regulatory checks | ✅ Done | KYC/AML/Basel/SEC ontologies? |
| FidelityScorer | Quality scoring | ⚠️ Check | Implementation? |
| ReplayEngine | Audit trails | ⚠️ Check | Implementation? |
| EvidenceBuilder | Compliance packs | ⚠️ Check | Implementation? |
| NRVTracer | Token tracing | ⚠️ Check | Implementation? |

### Workflow B — SSH Access

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| SSH Modal | Port 22000-22999 | ✅ Done | — |
| Web Terminal | xterm.js | ⚠️ Partial | Backend SSH proxy? |

### Workflow C — Mobile Controller Pairing

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| QR Code Display | `qr-code-display.tsx` | ✅ Done | — |
| WebSocket Pairing | Real-time events | ⚠️ Check | Backend pairing service? |

### Workflow D — DNS Management

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| DNS Management Panel | `dns-management.tsx` | ✅ Done | CloudFlare API integration? |
| Auto-update A Record | 5-minute intervals | ⚠️ Check | Cron/scheduler wiring? |

### Workflow E — Badge Creation Lab & DVE Badge Attachment

| Feature | Specified | Status | Missing |
|---------|-----------|--------|---------|
| Badge Lab Panel | `badge-lab-panel.tsx` | ✅ Done | — |
| Badge purpose input | Text field | ✅ Done | — |
| Values selection | 7 value tags | ✅ Done | — |
| Ontology element selection | 9 ontology tags | ✅ Done | — |
| **Auth credential scoping** | API keys, JWT bindings, provider creds in badge | ❌ Missing | `BadgeCreateRequest.Metadata` does not capture auth credentials; UI has no credential fields |
| Badge generation (AI/SVG) | AI-driven design encoding selected signals | ⚠️ Stub | Client-side static SVG — no actual AI generation or signal-to-visual encoding |
| Mint to Chain | `POST /api/knirvshell/chain/badge/create` | ✅ Done | — |
| `use-badge-lab.ts` hook | createBadge / mintBadge / getBadge | ✅ Done | — |
| Backend badge routes | `/chain/badge/create`, `/mint`, `/{id}` | ✅ Done | — |
| **DVE badge attachment** | `POST /api/knirvshell/chain/badge/mint` with DVE node ID | ⚠️ Partial | Route exists; no enforcement pipeline wired to GuardrailEngine |
| **GuardrailEngine badge injection** | Badge ontology tags → guardrail rules for DVE | ❌ Missing | GuardrailEngine does not read badge metadata to generate scoped rules |
| **ICME value alignment from badge** | Badge value signals → AlignmentLoop scoring | ❌ Missing | AlignmentLoop does not receive badge value signals |
| **Auth credential scope enforcement** | DVE agents restricted to badge-scoped credentials | ❌ Missing | DelegatorService does not check badge-scoped credential boundaries |
| **badge_id stamping on tasks** | Every DVE task/validation result/error node records active badge_id | ❌ Missing | Task/validation/error node schemas have no badge_id field |
| **Badge stacking (AND-evaluation)** | Multiple badges per DVE with merged rules | ❌ Missing | No merge logic defined or implemented |
| **Badge status on DVE card** | Attached badges displayed on DVE node card | ❌ Missing | `dve-nodes-panel.tsx` has no badge attachment display |
| Badge retrieval | `GET /api/knirvshell/chain/badge/{id}` | ✅ Done | — |

**Backend Wiring Missing:**
- `GuardrailEngine` must accept badge metadata at DVE attachment time and inject ontology-scoped rules
- `ICME AlignmentLoop` must receive badge value signals when a badge is attached to a DVE
- `DelegatorService` / `InferenceService` must restrict credential usage to the badge-scoped set for badge-attached DVEs
- `DVEManager` must store the badge-to-DVE binding and surface it on the DVE node status
- Task, validation result, and error node creation paths must stamp the active `badge_id`

---

### Workflow F — Workflow Execution

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Workflow Definition | Step ordering | ⚠️ Check | UI for step definition? |
| Execute API | `POST /api/workflow/execute` | ✅ Done | Dependency graph resolver? |
| Execution Events | WebSocket broadcast | ⚠️ Check | SSE events? |
| Status Endpoint | `GET /api/workflow/executions/{id}` | ⚠️ Check | Execution state tracking? |

### Workflow G — NRN Token Payments

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Transfer API | `POST /api/nrn/transfer` | ✅ Done | On-chain submission? |
| Oracle Balance | oracleBalanceAdapter | ⚠️ Check | Root node integration? |
| Stripe/PayPal | `/api/v1/payments/` | ⚠️ Check | Fiat checkout? |

### Workflow H — KNIRVSHELL Terminal

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Console Panel | `console-panel.tsx` | ✅ Done | — |
| Execute API | `POST /api/v1/shell/execute` | ✅ Done | — |
| Sub-commands | wallet/validation/tee/p2p/chain | ⚠️ Check | Full implementation? |
| Session Management | create/list/stop/input | ⚠️ Check | Full state machine? |

### Workflow I — Active Memory & Reasoning

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Active Memory Panel | `active-memory-panel.tsx` | ✅ Done | — |
| Reasoning Explorer | Graph traces | ⚠️ Check | Graph UI implementation? |
| Solution Vault | `plugin-vault-panel.tsx` | ✅ Done | PQC decryption? |

---

## 5. Backend API Routes (Missing / Unverified)

### Missing Endpoints

| Endpoint | Method | Specified In | Status |
|----------|--------|-------------|--------|
| `/api/guardrail/violations/{id}/resolve` | POST | Step 11 | ❌ Missing |
| `/api/system-health/*` | GET | Step 16 | ⚠️ Partial |
| `/api/cognitive/telemetry` | GET | Step 16 | ⚠️ Missing SSE |
| `/api/rollups/*` | GET | Step 17 | ⚠️ Partial |
| `/api/nrn/balance` | GET | UC-2 | ⚠️ Check |
| `/api/v1/knowledge-base/objects/{id}/query` | POST | UC-5 | ⚠️ Check |

### Unverified Implementation

| Endpoint | Method | Specified In | Notes |
|-----------|--------|-------------|-------|
| `/api/icme/alignment/status` | GET | Step 9 | Return alignment score? |
| `/api/active-memory/*` | PUT | Step 15 | Write reasoning traces? |

---

## 6. Frontend Components (Missing)

| Component | Specified In | Status |
|-----------|--------------|--------|
| `dve-creation-form.tsx` | Step 3 | ✅ Done |
| `policy-editor.tsx` | Step 10 | ✅ Done |
| `guardrail-violations-panel.tsx` | Step 11 | ✅ Done |
| `ssh-access-modal.tsx` | Workflow B | ✅ Done |
| `error-resolution-dashboard.tsx` | Step 14 | ⚠️ Partial |
| `error-resolution-modal.tsx` | Step 14 | ✅ Done |
| `predictive-analytics-panel.tsx` | Step 16 | ✅ Done |
| `dns-management.tsx` | Workflow D | ✅ Done |
| `qr-code-display.tsx` | Workflow C | ✅ Done |
| `financial-compliance-dashboard.tsx` | Workflow A | ✅ Done |
| `plugin-vault-panel.tsx` | Step 15 | ✅ Done |
| `knirvshell-console.tsx` | Workflow G | ⚠️ Partial (uses console-panel.tsx) |

---

## 7. Services (Missing / Incomplete)

| Service | Specified In | Status |
|---------|--------------|--------|
| DVECreationService | Step 3 | ✅ Done |
| P2P Manager (libp2p + DHT) | Step 4 | ✅ Done |
| GraphRAG-rs | Step 5 | ⚠️ FFI wired |
| ICME AlignmentLoop | Step 9 | ⚠️ Partial |
| GuardrailEngine | Step 11 | ✅ Done |
| ValidationCore | Step 12 | ✅ Done |
| ProofGenerator | Step 12 | ⚠️ Check |
| ErrorNodeService | Step 13 | ⚠️ Partial |
| AnchoringService | Step 15 | ⚠️ Partial |
| RollupService | Step 17 | ⚠️ Partial |
| DistributedScaler | Step 18 | ⚠️ Partial |
| FintechValidator | Workflow A | ⚠️ Check |
| FidelityScorer | Workflow A | ❌ Missing |
| ReplayEngine | Workflow A | ❌ Missing |
| EvidenceBuilder | Workflow A | ❌ Missing |
| NRVTracer | Workflow A | ❌ Missing |

---

## 8. Testing Gaps

| Test | Specified In | Status |
|------|--------------|--------|
| E2E primary workflow | 4.3 | ❌ Missing |
| Load testing | 4.3 | ❌ Missing |
| Guardrail policy fuzz | 4.3 | ❌ Missing |
| Root key loading | 4.3 | ❌ Missing |

---

## 9. Integration Gaps

| Integration | Specified In | Status |
|-------------|--------------|--------|
| KNIRVARENA (HERO Model) | 4.3 | ❌ Missing |
| KNIRVARENA (Dataset Forge) | 4.3 | ❌ Missing |
| CDE Service UX | 4.3 | ❌ Missing |
| oh-my-pi agent runtime | 4.3 | ❌ Missing |

---

## 10. Observability Gaps

| Feature | Specified In | Status |
|---------|--------------|--------|
| OpenTelemetry tracing | 4.3 | ❌ Missing |
| Alertmanager → PagerDuty/Slack | 4.3 | ⚠️ Config exists |
| eBPF dashboards | 4.3 | ❌ Missing |

---

## Summary Checklist

### Critical (Must Fix)
- [ ] Remove testnet tokens from production code
- [ ] Enforce AuthRequired in production config
- [ ] Implement email verification flow
- [ ] Implement password reset flow
- [ ] Implement token revocation on logout
- [ ] Implement Stripe/PayPal webhook verification
- [ ] Implement NRN idempotency

### High Priority
- [ ] Gate demo mode behind admin role
- [ ] Add error boundaries to all panels
- [ ] Implement WebSocket reconnect with exponential backoff
- [ ] Add loading/empty states to all panels
- [ ] Enforce role-based access on backend routes
- [ ] Add structured logging to all services
- [ ] Complete health check coverage
- [ ] Add Prometheus metrics endpoint

### Badge Lab Wiring
- [ ] Add auth credential fields to badge creation UI and `BadgeCreateRequest.Metadata`
- [ ] Wire badge attachment to `GuardrailEngine` (ontology tags → scoped guardrail rules)
- [ ] Wire badge value signals to `ICME AlignmentLoop` scoring at DVE attachment time
- [ ] Restrict `DelegatorService` to badge-scoped credentials for badge-attached DVEs
- [ ] Stamp `badge_id` on all DVE task results, validation results, and error nodes
- [ ] Implement badge stacking AND-evaluation in `GuardrailEngine`
- [ ] Display attached badges and enforcement status on DVE node card
- [ ] Replace client-side SVG stub with real badge generation (AI or server-side SVG templating)
- [ ] Store badge-to-DVE bindings in `DVEManager` with status endpoint

### Workflow Wiring
- [ ] Wire ValueSystem + Ontology to backend persistence
- [ ] Wire GuardrailRule auto-generation from ontology
- [ ] Wire ICME alignment status to UI
- [ ] Wire pre-threshold alerting (ProactiveDetector)
- [ ] Wire SSE for module logs
- [ ] Wire RollupService to Oracle
- [ ] Wire DistributedScaler to container orchestrator

### Secondary Workflows
- [ ] Complete FidelityScorer implementation
- [ ] Complete ReplayEngine implementation
- [ ] Complete EvidenceBuilder implementation
- [ ] Complete NRVTracer implementation
- [ ] Wire xterm.js to SSH backend
- [ ] Wire mobile controller pairing service
- [ ] Wire DNS auto-update cron

### Testing
- [ ] Write E2E test for primary workflow
- [ ] Write load tests for InferenceService
- [ ] Write fuzz tests for GuardrailEngine
- [ ] Write root key loading test

---

_This document reflects gaps identified as of 2026-04-20. Update after each milestone._