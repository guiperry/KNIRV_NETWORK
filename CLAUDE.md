# CLAUDE.md

## Project Overview

KNIRV Network is a multi-package platform for guarded AI agent execution. Policies and guardrails wrap agent actions before they run, executions get recorded in audit trails, and failures get mined into reusable `skill.md` knowledge instead of disappearing into a log file nobody rereads. `KNIRVSERVER` is the single entry point: it embeds compiled binaries for every other backend package (chain, gateway, graph, oracle, hasher, agent) and extracts/launches each as a subprocess at runtime. No shared database — services coordinate over Unix sockets and a handful of TCP ports.

The repo's own `README.md` flags this explicitly, and it applies here too: **treat "sovereign layers," "IBC," "D-TEN," and similar long-range framing as roadmap language, not a description of the current codebase**, unless a specific file or endpoint backs it up.

Oracle service runs inside `packages/KNIRVSERVER` (root nodes only, via encrypted `packages/KNIRVSERVER/bin/root.key`). KNIRVGATEWAY does not launch or manage the oracle process — KNIRVSERVER does that — but on root nodes KNIRVGATEWAY does proxy `/oracle/*` traffic to it over `oracle.sock` (`packages/KNIRVGATEWAY/internal/server/server.go`); on non-root nodes it forwards oracle requests to a public upstream gateway instead (`defaultOracleGatewayURL`).

## Component Map

12 packages under `packages/`, each independent (own `go.mod`/`package.json`, no cross-package Go imports):

| Package | Tech | Module / entry |
|---------|------|-----------------|
| `packages/KNIRVSERVER` | Go | `go.mod` module `knirv-server`. Entry point — router, guardrails, DVE/agent execution, embedded frontend. Embeds every package below as a subprocess binary. |
| `packages/KNIRVCHAIN` | Go | `go.mod`. Node/agent registries, P2P discovery, mining, validation, wallet, data engine. |
| `packages/KNIRVGATEWAY` | Go | `go.mod`. Public portal, DHT/TURN, auth, payments, operator/URI routing. Embedded inside KNIRVSERVER at runtime. |
| `packages/KNIRVGRAPH` | Go + TS | Knowledge graph — NRV, ErrorNodes/SkillNodes, Proof-of-Solution, NRN economics, React graph explorer. |
| `packages/KNIRVORACLE` | Go | Root-node governance/checkpoints. Routes mount only when `bin/root.key` is present. |
| `packages/KNIRVHASHER` | Go | Repurposed ASIC mining hardware doing neural-network inference. Pipeline stages: `pipeline/0_DATA_CONNECTOR`, `pipeline/1_DATA_MAPPER`, `pipeline/2_DATA_ENCODER`, `pipeline/3_DATA_TRAINER` — each its own `go.mod` (module names `data-connector`, `data-mapper`, `data-encoder`, `data-trainer`), built/tested from inside its own subdirectory only. |
| `packages/KNIRVAGENT` | Go | Autonomous agent runtime, `module github.com/knirvcorp/knirvagent`. Its README is currently stale upstream boilerplate — trust the code, not that file. |
| `packages/KNIRVARENA` | TS/React/Three.js | 3D client where Human Architects submit training data against live error nodes. Flat layout — source is directly under `packages/KNIRVARENA/src/` (no nested `packages/ts_client_2/`). |
| `packages/KNIRVCONTROLLER` | React/TS + Vite | End-user app: vault, DVE identities, voice/text chat. Ships as PWA + native Android/iOS via Capacitor. |
| `packages/KNIRVBRIDGE` | TS | Browser wallet extension for NRN tokens / dApp interaction. |
| `packages/KNIRVBASE` | Go + Rust + TS | Shared SDK/library. **Three parallel implementations** (`go/`, `rust/`, `ts/`) with no top-level README reconciling them — per `packages/KNIRVSERVER/CALIBER_LEARNINGS.md`, `go/` is treated as source-of-truth; Rust/TS are expected to conform to it. TS dist consumed at `packages/KNIRVBASE/ts/dist/lib/index.js`. |
| `packages/KNIRVSDK` | Go / TS / Py | Developer SDKs, plus KNIRV-CLI (`@knirv/cli`) source. |

Other top-level dirs actually present: `integration-tests/` (Go, real services, no mocks), `modp/` (P-language formal verification), `shared-proto/`, `scripts/`, `websites/KNIRV.NETWORK/` (only site currently in `websites/`).

**Not real — do not go looking for these:** `devtools/` (no such directory anywhere in this repo), `packages/KNIRVHEART`, `websites/KNIRVHUB`, `websites/KNIRVRAMP`. If you find yourself about to reference any of these, stop and re-check against the actual directory tree — this file has drifted this way before.

**KNIRVSHELL is mid-migration, not a package yet.** `KNIRVSHELL`/`knirvshell` is referenced in live code (`packages/KNIRVGATEWAY/internal/server/server.go`'s `shellProxy`/`ShellSocketPath` at `/api/knirvshell/`; `packages/KNIRVSERVER/main.go` expects a `knirvshell` binary in its bin dir), but there is no `packages/KNIRVSHELL` and no `packages/KNIRVSERVER/pkg/knirvshell/` on disk yet. `packages/KNIRVSERVER/docs/CLI_Migration.md` describes the plan: move the CLI service out of `backend_server` and into a new embedded `pkg/knirvshell/` package following the `knirvoracle`/`knirvgateway` pattern. Until that lands, don't assume a `knirvshell` package exists — check `packages/KNIRVSERVER/pkg/` first.

## Architecture

**Entry points:**
- KNIRVSERVER: `packages/KNIRVSERVER/main.go` (launcher — spawns the embedded `backend_server` binary as a subprocess). Real backend source is in the **separate `KNIRV_CORP` repo**: `KNIRV_CORP/packages/server/backend_server/cmd/backend_server/main.go` (note: directory is `backend_server`, not `backend`). `packages/KNIRVSERVER/backend` does not exist in this repo — the backend ships here only as a vendored compiled binary via `//go:embed bin/backend_server`.
- KNIRVGATEWAY: `packages/KNIRVGATEWAY/cmd/gateway/main.go` → `packages/KNIRVGATEWAY/internal/server/`
- KNIRVCHAIN: `packages/KNIRVCHAIN/cmd/knirvchain/` → `packages/KNIRVCHAIN/internal/`

**KNIRVSERVER internals (in `KNIRV_CORP/packages/server/backend_server`, NOT in this repo):** cognitive engine, onboarding, plugin server, guardrail handlers, oracle (root-node only), embedded config, keyfile (root.key/boot.key decryption). To inspect or change this logic you need the sibling `KNIRV_CORP` checkout — it is not present in `KNIRV_NETWORK`.

**KNIRVCHAIN internals:** `internal/mining/` · `internal/validation/` · `internal/auth/` · `internal/cache/` · `internal/database/` · `internal/agent/` · `internal/pricing/` · `internal/classifier/` · `internal/resilience/` · `internal/security/` · `internal/tracing/`

**KNIRVGATEWAY internals:** `internal/server/` · `internal/config/` · `internal/turnserver/` · `internal/embedded/`

**Node Transformation Flows (KNIRVGRAPH):**
- `ErrorNode → SkillNode` mining → `skill.md` file
- `ContextNode → CapabilityNode` minting → MCP Server Pointer
- `IdeaNode → PropertyNode` making → Inference NFT Pointer

**KNIRVARENA (Dataset Forge):** Human Architects craft datasets for active error nodes; submitted datasets + `skill.md` files drive error resolution and reward distribution. Knowledge is stored as `skill.md` markdown files, not LoRA adapters.

**Oracle (KNIRVSERVER root nodes only):** Loads `packages/KNIRVSERVER/bin/root.key` (AES-encrypted). Password via `ORACLE_KEY_PASSWORD` env or stdin prompt. Routes mounted at `/oracle/` only when key present. Missing key = normal operation, no oracle.

## Common Commands

```bash
make tests                              # run all tests
make testnet-build                      # build KNIRVSERVER
make testnet-start                      # start KNIRVSERVER (testnet is the default mode)
make testnet-stop                       # stop the local testnet instance
make testnet-status                     # show KNIRVSERVER health/status
make testnet-tests                      # start testnet + run integration tests
make health-check                       # check KNIRVSERVER health
make build-all                          # build every package in packages/
make test-modp                          # run ModP formal-verification tests
```

Run `make help` for the full target list — it's long (build-/test- targets per package, ModP variants, protobuf/binary sync). `make docs` and `make deploy-full` do **not** exist; don't invoke them.

**Testnet entry point:** there is no separate `--testnet` binary invocation — KNIRVSERVER defaults to testnet mode unless you pass `-prod`, `-dev`, or `-ent`.
**Testnet config:** `packages/KNIRVSERVER/config/testnet.yaml`
**Confirmed service ports:** KNIRVSERVER wrapper `:8090` · embedded backend API `:8082` · KNIRVGATEWAY `:8080` by default, `:8888` in `.env.testnet`/`.env.production`. KNIRVCHAIN and KNIRVGRAPH mostly communicate over Unix sockets rather than fixed TCP ports when run under KNIRVSERVER — don't assume a port for them without checking `packages/KNIRVSERVER/config/*.yaml` first.
**Health check:** `curl http://localhost:8090/health`

## Go Tests (Per Package)

```bash
cd packages/KNIRVSERVER && go test -v ./integration-tests/...   # wrapper-level only; backend_server's own tests live in KNIRV_CORP
cd packages/KNIRVCHAIN && go test -v ./tests/unit/...
cd packages/KNIRVGATEWAY && go test -v ./...
cd packages/KNIRVGRAPH && go test -v ./...
cd packages/KNIRVORACLE && go test -v ./...
cd integration-tests && go test -v ./...
```

## Formal Verification (ModP / P Language)

P models in `modp/` verified with dotnet PChecker via `modp/KnirvNetwork.pproj`.

```bash
make test-modp                 # via root Makefile
bash modp/scripts/run-tests.sh # direct P tests
```

**P model layout:** `modp/events/network_events.p` · `modp/components/base/base_layer.p` · `modp/components/chain/skill_registry.p` · `modp/components/oracle/governance_machine.p` · `modp/monitors/network_invariants.p`

Results: `modp/results/` · `modp/PCheckerOutput/BugFinding/`

ModP integration test: `integration-tests/modp_formal_verification_test.go`

## Integration Tests

Entry: `integration-tests/cmd/validate_network/main.go` · `integration-tests/utils/service_discovery.go`

Config: `integration-tests/config/service-discovery.yaml`

Scripts: `integration-tests/run-modp-tests.sh` · `scripts/run-full-demo.sh` · `scripts/validate-testnet-complete.sh`

## KNIRVARENA (3D Client)

Source is flat under `packages/KNIRVARENA/src/` — there is no `packages/KNIRVARENA/packages/ts_client_2/` layer.

Key files: `src/components/KNIRVANAGameVisualization.tsx` · `src/components/game/GameScene.tsx` · `src/networking/ArenaClient.ts`

`packages/KNIRVARENA/Makefile` and `packages/KNIRVARENA/scripts/run-all-tests.sh` do **not** exist; use `make build-knirvarena` / `make test-knirvarena` from the repo root instead.

**Asset paths:** KNIRVARENA is served at the `/arena/` sub-path — always use `` `${import.meta.env.BASE_URL}assets/...` `` for public asset references, not root-relative paths, or they 404 and Three.js reports a confusing JSON parse error instead of a 404.

## KNIRVBASE SDK

- Go SDK (source of truth): `packages/KNIRVBASE/go/`
- Rust SDK: `packages/KNIRVBASE/rust/`
- TS SDK: `packages/KNIRVBASE/ts/` — dist at `packages/KNIRVBASE/ts/dist/lib/index.js`
- No top-level doc currently explains how the three relate beyond the go-is-source-of-truth convention above — check `packages/KNIRVBASE/go/` first when in doubt.

## Websites

- `websites/KNIRV.NETWORK/` — KNIRV network hub (`health-monitor.html`, `index.html`, `developer-portal/`, `documentation/`, `forum/`)

## KNIRVSERVER Frontend

Next.js: `packages/KNIRVSERVER/frontend/` · built to `packages/KNIRVSERVER/frontend/out/` (embedded into the Go binary via `go:embed` — frontend changes are invisible in a running server until both `npm run build` here AND `go build` in KNIRVSERVER complete)

Key components: `src/components/dashboard/badge-lab-panel.tsx` · `src/components/dashboard/policy-editor.tsx` · `src/components/onboarding/onboarding-guide.tsx`

Desktop app: `packages/KNIRVSERVER/desktop/` (Electron — `renderer.js`, `index.html`)

## Conventions

- Each `packages/KNIRV*` is independent: `go mod tidy` and builds run inside the package, not root
- No cross-package Go imports — inter-service communication via HTTP/gRPC/Unix sockets only
- TypeScript: `unknown` over `any`; imports from `packages/KNIRVBASE/ts/dist/`
- Integration tests hit real services — no mock DB/network (see `integration-tests/`)
- New async protocols require corresponding P machine in `modp/components/`
- Oracle routes only when `packages/KNIRVSERVER/bin/root.key` present
- KNIRVGATEWAY does not launch/manage the oracle process (KNIRVSERVER does) — but it does proxy `/oracle/*` when a root node's socket is present, or forward to a public upstream gateway otherwise

## Production / Docs

`packages/KNIRVSERVER/config/production.yaml` · `packages/KNIRVSERVER/docs/SYSTEMD_SERVICE.md` · `packages/KNIRVSERVER/docs/Production_Deployment_Architecture.md` · `packages/KNIRVSERVER/docs/TESTING_PRIVILEGED.md` · `packages/KNIRVSERVER/docs/eBPF_Integration_Guide.md`

Note: `production.yaml` has historically shipped with eBPF monitoring, Cognitive Engine, and blockchain integration **disabled by default via feature flags**, even though the underlying subsystems are implemented in code. If a "missing" production feature turns out to be implemented but inert, check feature flags here before assuming it's unbuilt.

<!-- caliber:managed:pre-commit -->
## Before Committing

**IMPORTANT:** Before every git commit, you MUST ensure Caliber syncs agent configs with the latest code changes.

First, check if the pre-commit hook is already installed:
```bash
grep -q "caliber" .git/hooks/pre-commit 2>/dev/null && echo "hook-active" || echo "no-hook"
```

- If **hook-active**: the hook handles sync automatically — just commit normally. Tell the user: "Caliber will sync your agent configs automatically via the pre-commit hook."
- If **no-hook**: run Caliber manually before committing:
  1. Tell the user: "Caliber: Syncing agent configs with your latest changes..."
  2. Run: `caliber refresh && git add CLAUDE.md .claude/ .cursor/ .cursorrules .github/copilot-instructions.md .github/instructions/ AGENTS.md CALIBER_LEARNINGS.md .agents/ .opencode/ 2>/dev/null`
  3. After it completes, briefly tell the user what Caliber updated. Then proceed with the commit.

If `caliber` is not found, tell the user: "This project uses Caliber for agent config sync. Run /setup-caliber to get set up."
<!-- /caliber:managed:pre-commit -->

<!-- caliber:managed:learnings -->
## Session Learnings

Read `CALIBER_LEARNINGS.md` for patterns and anti-patterns learned from previous sessions.
These are auto-extracted from real tool usage — treat them as project-specific rules.
<!-- /caliber:managed:learnings -->

<!-- caliber:managed:sync -->
## Context Sync

This project uses [Caliber](https://github.com/caliber-ai-org/ai-setup) to keep AI agent configs in sync across Claude Code, Cursor, Copilot, and Codex.
Configs update automatically before each commit via `caliber refresh`.
If the pre-commit hook is not set up, run `/setup-caliber` to configure everything automatically.

**Caveat observed 2026-07-28:** this file (`CLAUDE.md`) was found significantly out of sync with the actual repo (phantom `devtools/` paths, a removed `KNIRVHEART` package, wrong service ports, a wrong KNIRVARENA source layout) despite the pre-commit hook being active (`caliber refresh` runs, `caliber` binary present). If this file drifts again, don't assume the hook is catching it — spot-check a few concrete claims (do the referenced paths exist?) before trusting it wholesale.
<!-- /caliber:managed:sync -->

## Codebase Search (SocratiCode)

This project is indexed with SocratiCode. Always use its MCP tools to explore the codebase
before reading any files directly.

### Workflow

1. **Start most explorations with `codebase_search`.**
   Hybrid semantic + keyword search (vector + BM25, RRF-fused) runs in a single call.
   - Use broad, conceptual queries for orientation: "how is authentication handled",
     "database connection setup", "error handling patterns".
   - Use precise queries for symbol lookups: exact function names, constants, type names.
   - Prefer search results to infer which files to read — do not speculatively open files.
   - **When to use grep instead**: If you already know the exact identifier, error string,
     or regex pattern, grep/ripgrep is faster and more precise — no semantic gap to bridge.
     Use `codebase_search` when you're exploring, asking conceptual questions, or don't
     know which files to look in.

2. **Follow the graph before following imports.**
   Use `codebase_graph_query` to see what a file imports and what depends on it before
   diving into its contents. This prevents unnecessary reading of transitive dependencies.

3. **Read files only after narrowing down via search.**
   Once search results clearly point to 1–3 files, read only the relevant sections.
   Never read a file just to find out if it's relevant — search first.

4. **Use `codebase_graph_circular` when debugging unexpected behavior.**
   Circular dependencies cause subtle runtime issues; check for them proactively.

5. **Check `codebase_status` if search returns no results.**
   The project may not be indexed yet. Run `codebase_index` if needed, then wait for
   `codebase_status` to confirm completion before searching.

6. **Leverage context artifacts for non-code knowledge.**
   Projects can define a `.socraticodecontextartifacts.json` config to expose database
   schemas, API specs, infrastructure configs, architecture docs, and other project
   knowledge that lives outside source code. These artifacts are auto-indexed alongside
   code during `codebase_index` and `codebase_update`.
   - Run `codebase_context` early to see what artifacts are available.
   - Use `codebase_context_search` to find specific schemas, endpoints, or configs
     before asking about database structure or API contracts.
   - If `codebase_status` shows artifacts are stale, run `codebase_context_index` to
     refresh them.

### When to use each tool

| Goal | Tool |
|------|------|
| Understand what a codebase does / where a feature lives | `codebase_search` (broad query) |
| Find a specific function, constant, or type | `codebase_search` (exact name) or grep if you know already the exact string |
| Find exact error messages, log strings, or regex patterns | grep / ripgrep |
| See what a file imports or what depends on it | `codebase_graph_query` |
| Spot architectural problems | `codebase_graph_circular`, `codebase_graph_stats` |
| Visualise module structure | `codebase_graph_visualize` |
| Verify index is up to date | `codebase_status` |
| Discover what project knowledge (schemas, specs, configs) is available | `codebase_context` |
| Find database tables, API endpoints, infra configs | `codebase_context_search` |
