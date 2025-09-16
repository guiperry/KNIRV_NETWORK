## KNIRVCONTROLLER — Gap Analysis

Date: 2025-09-16

Purpose
- Provide a focused inventory of functionality currently implemented in `KNIRVCONTROLLER`, identify incomplete or missing capabilities, and recommend concrete implementation steps, priorities, tests, and acceptance criteria to get the component operational in production.

Methodology
- Read key artifacts: `package.json`, `README.md`, `src/` entry points and services, and test scaffolding.
- Traced services and UI components referenced by the app and server.
- Marked live implementations vs. simulated/mock implementations.

Summary — what exists today (implemented or scaffolded)
- Frontend: Full React + TypeScript single-page app (`src/App.tsx`, many components). UI features implemented with slideouts, NRV visualization, agent manager, cognitive shell UI, QR scanner UI, and mock-driven flows.
- Backend server: `src/server/api-server.ts` — Express-based API server with many REST endpoints (health, api keys, agents, cognitive endpoints, terminal simulation, websockets). Endpoints are functional but rely on internal mock/simulated services for many behaviors.
- Local persistence: `src/services/RxDBService.ts` implements RxDB collections and a basic database schema and initialization.
- API key management: `src/services/ApiKeyService.ts` implements key creation, validation, rate-limit checks, and usage recording (stores data in RxDB settings collection). Reasonable logic but storage and security are simple.
- Personal graph: `src/services/PersonalKNIRVGRAPHService.ts` manages a per-user graph object, node/edge primitives, heuristics for related-skill detection, and persistence calls that currently store minimal settings metadata to RxDB (graph persistence is simulated).
- Knirvana bridge: `src/services/KnirvanaBridgeService.ts` converts personal graph nodes into a game-state model and simulates agent lifecycle and NRN accounting.
- Wallet & blockchain stubs: `AbstraxionWalletService`, `XionMetaAccount`, `WalletIntegrationService` present but mostly mocked (simulate connect, transactions, gasless ops).
- Cortex & training: `CortexTrainingService` exists and simulates training loops; TensorFlow and WASM are present in dependency manifest but are mocked in tests and some components.
- Tests: Jest setup with many mocks to allow unit tests to run without heavy dependencies. Test scaffolding and numerous unit tests exist.
- Build tooling: Vite front-end, AssemblyScript/WASM build scripts, WASM loader, npm scripts for building and previewing.

High-level gaps (architectural / functional)
1) External network & chain integration (Critical)
   - Missing: Real connectivity to KNIRVCHAIN, KNIRVGRAPH, KNIRV-ORACLE, KNIRV-ROUTER, and XION bridge.
   - Current: Simulated transactions and in-memory operations. No production-grade RPC/IBC integrations.
   - Risk: Cannot perform real NRN/tokenized flows, skill invocation tracking, or cross-chain verification.

2) Wallet / Meta-account production integration (High)
   - Missing: Real Abstraxion/XION MetaAccount integration, secure key storage, gasless execution on-chain.
   - Current: Mock SDK methods; RxDB stores mock wallet docs; no hardware-backed key store.

3) Agent runtime & skill execution (Critical)
   - Missing: Deterministic agent runtime to deploy/execute WASM skills, resource isolation, and real invocation on DVE.
   - Current: UI simulates deploy/execute flow; `api-server` and services return simulated outputs.
   - Risk: No ability to validate or bill NRN for skill executions.

4) WASM build & runtime integration (High)
   - Missing: Verified build pipeline for production WASM (AssemblyScript / Rust) and runtime binding in browser/Node (with real guardrails).
   - Current: build scripts exist but tests mock the WASM module and runtime.

5) Persistence & sync for Personal Graph (Medium)
   - Missing: A real persisted graph model (structured documents) and sync/mirroring to the collective registry.
   - Current: Graph saved as settings entries (metadata only); import/export functions exist but are not linked to a collective backend.

6) Secure authentication & authorization (High)
   - Missing: Production authentication (JWT / session / OAuth) for users and admin flows; API key storage security improvements (secret encryption at rest).
   - Current: ApiKeyService stores keys in plain RxDB settings; server relies on X-API-Key header only.

7) Rate limiting, metrics, monitoring (Medium)
   - Missing: Durable rate-limiter (Redis/leaky-bucket), Prometheus metrics exposure, and production logging/alerting integration.
   - Current: ApiKeyService implements naive rate checks by reading usage entries from RxDB (not performant/durable).

8) End-to-end test coverage / integration tests (Medium)
   - Missing: Real integration tests against local network (docker-compose or k8s stack). Many unit tests use heavy mocking.

9) Production deployment wiring (Medium)
   - Missing: Built server packaging for Node/Rust/WASM, docker images that wire the controller to other components, secure env handling.
   - Current: `deployment/` and `docker-compose` exist in repo root but controller-specific manifest and secrets management require validation.

Concrete remediation plan (tasks, acceptance criteria, tests, priority)
Priority key: P0 = Critical (blocker for production), P1 = High, P2 = Medium

P0 — Agent runtime + chain & token integration
- Tasks:
  1. Implement an agent execution runtime for WASM skills (server-side orchestrator + lightweight sandbox). Files: create `src/runtime/agent-runtime.ts`, wire to `/api/agents/*` endpoints.
  2. Integrate with KNIRVCHAIN RPC / smart contract endpoints (or a gateway) to record skill invocations and burn NRN. Use @cosmjs for CosmWasm interactions. Add configuration in `src/config/xion-config.ts`.
  3. Create end-to-end billing test: deploy a sample WASM skill, execute, assert NRN balance decreased and invocation logged.
- Acceptance:
  - Skills execute in sandboxed WASM with CPU/memory limits.
  - Invocation events recorded on-chain (or simulated local test ledger) and reimbursed accordingly.
  - Tests: unit tests for runtime, an integration test that runs a WASM skill and asserts token burn.

P1 — Wallet & MetaAccount production integration
- Tasks:
  1. Replace mocks in `AbstraxionWalletService` / `XionMetaAccount` with real SDK usage. Implement secure key storage using OS keystore or HSM where available, or encrypt keys stored in RxDB.
  2. Implement real transaction signing and monitoring, and surface status via APIs and UI.
  3. Add smoke tests that simulate sending USDC -> conversion -> NRN and assertion on balances.
- Acceptance:
  - Users can connect a real MetaAccount and create signed transactions.
  - UI reflects live balance and transaction status.

P1 — WASM build & runtime polish
- Tasks:
  1. Harden build scripts for AssemblyScript and Rust wasm, produce deterministic artifacts and sourcemaps. Add CI job that builds wasm on PRs.
  2. Integrate `@napi-rs/wasm-runtime` or appropriate host for server-side invocation; ensure browser loader is used for client features only.
  3. Add unit tests that load the built wasm and run a key exported function to validate shape.

P1 — Auth, API key security & rate limiting
- Tasks:
  1. Migrate ApiKey storage to encrypted RxDB documents and/or move to server-side secure storage (vault). Introduce HMAC verification and rotate keys.
  2. Add JWT-based auth for user sessions and admin role checks in `api-server.ts`. Keep ApiKeys for machine-to-machine but make creation controlled.
  3. Replace naive rate limiting with Redis-based leaky bucket or token bucket and move checks to middleware.

P2 — Graph persistence and collective sync
- Tasks:
  1. Convert `PersonalKNIRVGRAPHService` to persist full graph documents in `db.graphs` collection; add import/export endpoints.
  2. Implement a sync/merge protocol to push new skill nodes to KNIRVANA/collective network (KnirvanaBridgeService → real API).

P2 — Monitoring, metrics, CI/CD
- Tasks:
  1. Expose Prometheus metrics (request counters, latencies, wasm execution counts) and wire to `deployment/monitoring` stack.
  2. Add GitHub Actions or CI job that runs: lint, unit tests, wasm build, and a smoke integration that starts the API server and runs a small E2E.

Quick wins (low-effort, high-value)
- Wire simple persistence: persist graphs as JSON in RxDB `graphs` collection instead of `settings` entries.
- Add a WebSocket broadcast for cognitive state changes in `api-server.ts` to enable real-time UI updates beyond echo.
- Add CI job to run `npm run build:wasm` and `npm run build` on PRs to prevent regressions.

User capture flow (Errors, Context, Ideas) — additions
 - Purpose: `KNIRVCONTROLLER` is primarily used to capture training data for SLM WASM Agents from mobile devices. The controller captures three user-submitted node types:
    - Errors -> become Skill nodes (used to train SLM agents)
    - Context (MCP Servers) -> become Capability nodes
    - Ideas -> become Property nodes (ownable)

 - UI requirement (center button): The app's center/fab button should expand into three sub-buttons when pressed, letting the user choose to submit an Error, a Context server, or a new Idea to their private graph. The center button behaviour is currently mocked in the UI and needs production wiring.

 - Slicing requirements for new nodes:
    - Factuality Slicing for Skills (Errors): Each new Skill node created from an Error must go through a factuality-slicing pipeline. This pipeline should:
       1. Extract candidate factual statements from the Error description and associated context (logs, screenshots, transcripts).
       2. Score each statement for verifiability using heuristics: corroborating logs, stack traces, timestamps, network context, and similarity to known error patterns.
       3. Produce a factuality slice: a ranked set of factual assertions with provenance metadata (source, confidence, evidence links).
       4. Attach the factuality slice to the Skill node and persist it.

    - Feasibility Slicing for Ideas (Properties): Instead of factuality slicing, Ideas should be accompanied by a novel feasibility slice which includes an "is this already in existence" report. The feasibility slice must:
       1. Run similarity checks against existing public and private property nodes (local graph and optionally a collective index).
       2. Query lightweight market/knowledge signals (cached search, local heuristics, or a remote search index when available) to report whether similar projects exist.
       3. Produce a feasibility score and a short report listing similar items, approximate market size indicators (if available), and a feasibility confidence.
       4. Attach the feasibility slice to the Property node and persist it.

Implementation tasks for the capture flow and slicing (priority mapping)
 - P0 (Critical) — Center button capture UI + API
    1. Implement FAB expansion in `src/components/KnirvShell` or main input component; add three actions: Submit Error, Submit Context, Submit Idea. Wire to UI modals for structured submission.
    2. Add REST endpoints to `src/server/api-server.ts` for `POST /api/graph/error`, `POST /api/graph/context`, `POST /api/graph/idea` that accept structured payloads and return created node IDs and slice status.
    3. Update `PersonalKNIRVGRAPHService` with methods that accept pre-computed slices and persist nodes fully (migrate to real `graphs` collection during P2 work).
    4. Unit tests: UI event -> API call -> graph service invoked -> persistence verified.

 - P1 (High) — Factuality slicing pipeline for Skill nodes
    1. Create `src/slices/factualitySlice.ts` implementing: statement extraction (simple sentence-level heuristics + regex for code/log patterns), evidence matching against context payload, scoring, and provenance metadata.
    2. Integrate a plug-in architecture so future ML models can augment the slice (e.g., use a small LLM or a retriever to expand evidence).
    3. Add unit tests with example error payloads verifying that slices include provenance and expected score ordering.

 - P1 (High) — Feasibility slice pipeline for Ideas
    1. Create `src/slices/feasibilitySlice.ts` implementing: similarity checks (text-similarity via vector search or TF-IDF on local graph), existence detection, and a simplified market/feasibility scoring.
    2. Provide a remote/optional adapter that can call a search index or knowledge API when configured.
    3. Add tests that mock existing graph entries and validate the 'already exists' report and feasibility score.

Acceptance criteria for slicing features
 - Factuality slice created for every new Skill node with at least one provable assertion when evidence exists.
 - Feasibility slice produced for Idea nodes with a deterministic similarity ranking and clear 'exists' boolean when similarity threshold exceeded.
 - API responses return node ID, slice summary (top 3 items), and storage status.

Integration and tests
 - Add end-to-end test: simulate mobile submission of Error (with log snippet + screenshot metadata), assert that `/api/graph/error` returns node ID, Skill node persisted with factuality slice attached, and UI shows created Skill.
 - Add mocks for the optional remote knowledge API so CI can run the feasibility slice tests without external network calls.

Notes and alignment with existing code
 - `App.tsx` and components already include UI flows for submit error/context/idea (see handlers `handleSubmitError`, `handleSubmitContext`, `handleSubmitIdea`) that call `PersonalKNIRVGRAPHService`. Those handlers should be extended to call the new API endpoints and trigger slicing server-side (or client-side if configured offline).
 - `PersonalKNIRVGRAPHService.addErrorNode`, `addContextNode`, and `addIdeaNode` already exist and should be extended to accept slices and store them in the graph model.

Requirements coverage update
 - The new capture flow and slicing requirements have been integrated into the Gap Analysis and remediation plan (this document). Status: Done.


Risks & mitigations
- Running untrusted WASM: require sandboxing, timeouts, and resource quotas. Consider Wasmtime or Node `vm`-based sandboxes plus wasm runtime.
- Financial operations: ensure conversions and burns are run only after atomic on-chain confirmations or use escrow patterns.
- Data leakage: encrypt secrets and API keys at rest; do not expose raw keys in API responses.

Next steps (recommended immediate sprint)
1. Implement graph persistence in RxDB and add a small API to export/import graphs (1–2 days).
2. Wire Prometheus metrics & small CI check for wasm build (1–2 days).
3. Design agent runtime contract and PoC sandbox for a simple WASM skill execution (3–5 days).
4. Plan blockchain integration spike: selective integration tests with a local testnet or simulator for XION/KNIRVCHAIN (3–7 days).

Appendix — files of interest (examples)
- `src/server/api-server.ts` — primary HTTP + WebSocket server
- `src/services/ApiKeyService.ts` — API key lifecycle and rate check
- `src/services/RxDBService.ts` — local DB and collections
- `src/services/PersonalKNIRVGRAPHService.ts` — personal graph manager
- `src/services/KnirvanaBridgeService.ts` — game/collective bridge
- `src/services/AbstraxionWalletService.ts`, `src/services/XionMetaAccount.ts` — wallet/blockchain stubs
- `package.json` — build/test scripts (wasm scripts included)

Requirements coverage
- Drafted a focused gap analysis and prioritized remediation list covering: agent runtime, blockchain integration, wallet integration, WASM pipeline, graph persistence, auth, monitoring, and CI — status: Done (this document).

If you want, I can now: (A) open a small PR that creates the graph persistence collection and a basic export/import API, or (B) scaffold the agent runtime PoC and tests. Tell me which to start and I'll proceed.
